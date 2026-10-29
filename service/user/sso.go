package user

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

const (
	// SSOProviderOIDC is the generic OIDC provider. "logto" is a preset of
	// it: both share the same oidc_* settings and binding slot.
	SSOProviderOIDC  = "oidc"
	SSOProviderLogto = "logto"
	SSOProviderQQ    = "qq"

	ssoStateKeyPrefix = "sso_state_"
	ssoStateTTL       = 600
	ssoHTTPTimeout    = 10 * time.Second
	ssoMaxBodySize    = 1 << 20
)

type (
	SSORedirectParameterCtx struct{}
	// SSORedirectService starts an SSO sign-in by redirecting the browser to
	// the identity provider.
	SSORedirectService struct {
		Provider string `uri:"provider" binding:"required"`
	}
	SSOCallbackParameterCtx struct{}
	// SSOCallbackService finishes an SSO sign-in after the identity provider
	// redirects back with an authorization code.
	SSOCallbackService struct {
		Provider string `uri:"provider" binding:"required"`
	}
)

// ssoSession is the short-lived record of one SSO authorization round trip,
// stored in KV and keyed by the OAuth state parameter.
type ssoSession struct {
	Provider         string `json:"provider"`
	Nonce            string `json:"nonce"`
	TokenEndpoint    string `json:"token_endpoint"`
	UserInfoEndpoint string `json:"userinfo_endpoint"`
	// ReturnTo is a safe relative path to land on after sign-in (e.g. the
	// OAuth authorization page). Empty falls back to /session.
	ReturnTo string `json:"return_to,omitempty"`
}

// oidcProviderMeta is the subset of the OIDC discovery document we need.
type oidcProviderMeta struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
}

// ssoUserInfo is the subset of OIDC user claims we consume. Email is
// mandatory: it is the account identifier on the Cloudreve side.
type ssoUserInfo struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	Nickname          string `json:"nickname"`
	PreferredUsername string `json:"preferred_username"`
	Nonce             string `json:"nonce,omitempty"`
}

// oidcReady reports whether the generic OIDC provider is configured well
// enough to start a sign-in flow.
func oidcReady(s *setting.SSOSetting) bool {
	return s.OIDCEnabled && s.OIDCIssuer != "" && s.OIDCClientID != ""
}

// qqReady reports whether QQ Connect sign-in is configured.
func qqReady(s *setting.SSOSetting) bool {
	return s.QQEnabled && s.QQAppID != "" && s.QQAppSecret != ""
}

// normalizeProvider maps URL provider names to binding keys.
func normalizeProvider(provider string) (string, error) {
	switch strings.ToLower(provider) {
	case SSOProviderOIDC, SSOProviderLogto:
		return SSOProviderOIDC, nil
	case SSOProviderQQ:
		return SSOProviderQQ, nil
	default:
		return "", serializer.NewError(serializer.CodeNotFound, "Unknown SSO provider", nil)
	}
}

// ssoCallbackURL builds the redirect_uri that must be registered at the
// identity provider.
func ssoCallbackURL(c *gin.Context, provider string) *url.URL {
	base := dependency.FromContext(c).SettingProvider().SiteURL(c)
	route, _ := url.Parse(constants.APIPrefix + "/session/sso/" + provider + "/callback")
	return base.ResolveReference(route)
}

// ssoLoginURL builds the frontend sign-in URL where the flow continues.
func ssoLoginURL(c *gin.Context) *url.URL {
	base := dependency.FromContext(c).SettingProvider().SiteURL(c)
	route, _ := url.Parse("/session")
	return base.ResolveReference(route)
}

// LoginRedirectURL builds the frontend sign-in URL carrying an error message
// in the fragment for the login page to display. Used by controllers when the
// SSO flow fails on a browser navigation.
func LoginRedirectURL(c *gin.Context, err error) string {
	msg := err.Error()
	var appErr serializer.AppError
	if errors.As(err, &appErr) {
		msg = appErr.Msg
	}
	u := ssoLoginURL(c)
	u.Fragment = "sso_error=" + url.QueryEscape(msg)
	return u.String()
}

// ssoCredentialRedirect builds the frontend URL carrying the freshly issued
// credential in the fragment. The fragment is never sent to any server.
func ssoCredentialRedirect(c *gin.Context, login *BuiltinLoginResponse, returnTo string) (string, error) {
	payload, err := json.Marshal(login)
	if err != nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Failed to encode login response", err)
	}
	u := loginLandingURL(c, returnTo)
	u.Fragment = "sso_credential=" + base64.RawURLEncoding.EncodeToString(payload)
	return u.String(), nil
}

// Redirect stores the OAuth state and returns the identity provider's
// authorization endpoint URL.
func (s *SSORedirectService) Redirect(c *gin.Context) (string, error) {
	provider, err := normalizeProvider(s.Provider)
	if err != nil {
		return "", err
	}

	dep := dependency.FromContext(c)
	cfg := dep.SettingProvider().SSO(c)

	returnTo := safeReturnTo(c.Query("return"))

	var authEndpoint string
	session := ssoSession{
		Provider: provider,
		ReturnTo: returnTo,
	}

	switch provider {
	case SSOProviderQQ:
		if !qqReady(cfg) {
			return "", serializer.NewError(serializer.CodeNotFound, "SSO provider is not enabled", nil)
		}
		authEndpoint = "https://graph.qq.com/oauth2.0/authorize"
	default:
		if !oidcReady(cfg) {
			return "", serializer.NewError(serializer.CodeNotFound, "SSO provider is not enabled", nil)
		}

		meta, err := discoverOIDC(c, cfg.OIDCIssuer)
		if err != nil {
			return "", err
		}
		session.Nonce = util.RandStringRunesCrypto(32)
		session.TokenEndpoint = meta.TokenEndpoint
		session.UserInfoEndpoint = meta.UserInfoEndpoint
		authEndpoint = meta.AuthorizationEndpoint
	}

	state := util.RandStringRunesCrypto(32)
	encoded, err := json.Marshal(session)
	if err != nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Failed to encode SSO state", err)
	}
	if err := dep.KV().Set(ssoStateKeyPrefix+state, string(encoded), ssoStateTTL); err != nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Failed to store SSO state", err)
	}

	authorize, err := url.Parse(authEndpoint)
	if err != nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Invalid authorization endpoint", err)
	}
	query := authorize.Query()
	query.Set("response_type", "code")
	query.Set("redirect_uri", ssoCallbackURL(c, provider).String())
	query.Set("state", state)
	if provider == SSOProviderQQ {
		query.Set("client_id", cfg.QQAppID)
	} else {
		query.Set("client_id", cfg.OIDCClientID)
		query.Set("scope", "openid profile email")
		query.Set("nonce", session.Nonce)
	}
	authorize.RawQuery = query.Encode()
	return authorize.String(), nil
}

// safeReturnTo accepts only same-site relative paths for post-login landing,
// preventing open redirects through the ?return= parameter.
func safeReturnTo(raw string) string {
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "\\") {
		return ""
	}
	return raw
}

// loginLandingURL builds the frontend sign-in URL to land on after sign-in.
// A validated return target is passed as ?redirect= so the login page's
// existing post-login navigation picks it up after processing the fragment
// credential (the fragment must land on /session to be handled at all).
func loginLandingURL(c *gin.Context, returnTo string) *url.URL {
	u := ssoLoginURL(c)
	if returnTo != "" && safeReturnTo(returnTo) == returnTo {
		q := u.Query()
		q.Set("redirect", returnTo)
		u.RawQuery = q.Encode()
	}
	return u
}

// Callback validates the state, exchanges the authorization code for tokens,
// resolves the identity to a user (registering one when needed) and returns
// the frontend URL to land on.
func (s *SSOCallbackService) Callback(c *gin.Context) (string, error) {
	provider, err := normalizeProvider(s.Provider)
	if err != nil {
		return "", err
	}

	dep := dependency.FromContext(c)
	cfg := dep.SettingProvider().SSO(c)
	if provider == SSOProviderQQ {
		if !qqReady(cfg) {
			return "", serializer.NewError(serializer.CodeNotFound, "SSO provider is not enabled", nil)
		}
	} else if !oidcReady(cfg) {
		return "", serializer.NewError(serializer.CodeNotFound, "SSO provider is not enabled", nil)
	}

	if oauthErr := c.Query("error"); oauthErr != "" {
		return "", serializer.NewError(serializer.CodeParamErr, "Identity provider returned error: "+oauthErr, nil)
	}

	code, state := c.Query("code"), c.Query("state")
	if code == "" || state == "" {
		return "", serializer.NewError(serializer.CodeParamErr, "Missing authorization code in SSO callback", nil)
	}

	kv := dep.KV()
	sessionRaw, ok := kv.Get(ssoStateKeyPrefix + state)
	if !ok {
		return "", serializer.NewError(serializer.CodeNotFound, "SSO state is invalid or expired", nil)
	}
	_ = kv.Delete(ssoStateKeyPrefix + state)

	var session ssoSession
	sessionStr, ok := sessionRaw.(string)
	if !ok {
		return "", serializer.NewError(serializer.CodeNotFound, "SSO state is invalid", nil)
	}
	if err := json.Unmarshal([]byte(sessionStr), &session); err != nil || session.Provider != provider {
		return "", serializer.NewError(serializer.CodeNotFound, "SSO state is invalid", nil)
	}

	var info *ssoUserInfo
	if provider == SSOProviderQQ {
		info, err = resolveQQIdentity(c, cfg, code, ssoCallbackURL(c, provider).String())
	} else {
		var accessToken, idToken string
		accessToken, idToken, err = exchangeOIDCToken(c, cfg, session.TokenEndpoint, code, ssoCallbackURL(c, provider).String())
		if err != nil {
			return "", err
		}

		if session.UserInfoEndpoint != "" {
			info, err = fetchOIDCUserInfo(c, session.UserInfoEndpoint, accessToken)
		} else {
			// Provider exposes no userinfo endpoint; fall back to the id_token
			// claims. It arrives directly from the token endpoint over TLS, so
			// the payload can be trusted in the code flow. The nonce binds it
			// to this authorization round trip.
			info, err = parseIDTokenClaims(idToken)
			if err == nil && session.Nonce != "" && info.Nonce != session.Nonce {
				err = serializer.NewError(serializer.CodeParamErr, "id_token nonce mismatch", nil)
			}
		}
	}
	if err != nil {
		return "", err
	}
	if info.Sub == "" {
		return "", serializer.NewError(serializer.CodeParamErr, "Identity provider returned no subject", nil)
	}

	nick := info.Name
	if nick == "" {
		nick = info.Nickname
	}
	if nick == "" {
		nick = info.PreferredUsername
	}
	u, err := ssoLogin(c, provider, info.Sub, info.Email, nick)
	if err != nil {
		return "", err
	}

	if u.TwoFactorSecret != "" {
		twoFaSessionID := uuid.Must(uuid.NewV4())
		if err := kv.Set(fmt.Sprintf("user_2fa_%s", twoFaSessionID), u.ID, 600); err != nil {
			return "", serializer.NewError(serializer.CodeInternalSetting, "Failed to create 2FA session", err)
		}
		target := loginLandingURL(c, session.ReturnTo)
		target.Fragment = "sso_2fa=" + twoFaSessionID.String()
		return target.String(), nil
	}

	util.WithValue(c, inventory.UserCtx{}, u)
	login, err := IssueToken(c)
	if err != nil {
		return "", err
	}
	return ssoCredentialRedirect(c, login, session.ReturnTo)
}

// ssoLogin resolves an SSO identity to a Cloudreve user: an existing account
// with the same email is bound to (or checked against) the provider subject;
// otherwise a new active account in the default group is registered.
func ssoLogin(c *gin.Context, provider, subject, email, nick string) (*ent.User, error) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, serializer.NewError(serializer.CodeParamErr, "Identity provider did not return an email address", nil)
	}

	ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	u, err := dep.UserClient().GetByEmail(ctx, email)
	if err != nil {
		// SSO sign-in is exempt from the registration email domain filter
		// (the setting only gates public self-registration).
		created, err := dep.UserClient().Create(c, &inventory.NewUserArgs{
			Email:    email,
			Nick:     nick,
			Status:   user.StatusActive,
			GroupID:  settings.DefaultGroup(c),
			Language: browserLanguage(c),
		})
		if err != nil {
			return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to create user", err)
		}

		// Reload with eager loaded edges for token issuing.
		u, err = dep.UserClient().GetByID(ctx, created.ID)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to load created user", err)
		}
		if err := bindSSO(c, u, provider, subject); err != nil {
			return nil, err
		}
		return u, nil
	}

	if u.Status == user.StatusManualBanned || u.Status == user.StatusSysBanned {
		return nil, serializer.NewError(serializer.CodeUserBaned, "This account has been blocked", nil)
	}
	if u.Status == user.StatusInactive {
		return nil, serializer.NewError(serializer.CodeUserNotActivated, "This account is not activated", nil)
	}

	if existing, ok := u.Settings.SSOBindings[provider]; ok && existing != subject {
		return nil, serializer.NewError(serializer.CodeOpenIDBindConflict,
			"This account is linked to a different identity at the provider", nil)
	}
	if err := bindSSO(c, u, provider, subject); err != nil {
		return nil, err
	}
	return u, nil
}

// bindSSO records the provider subject on the user account; it is a no-op
// when the account is already linked to the same subject.
func bindSSO(c *gin.Context, u *ent.User, provider, subject string) error {
	if u.Settings == nil {
		u.Settings = &types.UserSetting{}
	}
	if existing, ok := u.Settings.SSOBindings[provider]; ok && existing == subject {
		return nil
	}
	if u.Settings.SSOBindings == nil {
		u.Settings.SSOBindings = make(map[string]string)
	}
	u.Settings.SSOBindings[provider] = subject
	if err := dependency.FromContext(c).UserClient().SaveSettings(c, u); err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to save SSO binding", err)
	}
	return nil
}

// discoverOIDC fetches the provider metadata from the well-known endpoint.
func discoverOIDC(ctx context.Context, issuer string) (*oidcProviderMeta, error) {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	if issuer == "" {
		return nil, serializer.NewError(serializer.CodeParamErr, "OIDC issuer is empty", nil)
	}

	// Accept both the issuer URL and a full well-known document URL.
	discoveryURL := issuer + "/.well-known/openid-configuration"
	if strings.HasSuffix(issuer, "/.well-known/openid-configuration") {
		discoveryURL = issuer
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to build discovery request", err)
	}
	resp, err := (&http.Client{Timeout: ssoHTTPTimeout}).Do(req)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to fetch OIDC discovery document", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, serializer.NewError(serializer.CodeInternalSetting,
			fmt.Sprintf("OIDC discovery endpoint returned status %d", resp.StatusCode), nil)
	}

	var meta oidcProviderMeta
	if err := json.NewDecoder(io.LimitReader(resp.Body, ssoMaxBodySize)).Decode(&meta); err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to parse OIDC discovery document", err)
	}
	if meta.AuthorizationEndpoint == "" || meta.TokenEndpoint == "" {
		return nil, serializer.NewError(serializer.CodeInternalSetting,
			"OIDC discovery document misses authorization or token endpoint", nil)
	}
	return &meta, nil
}

// exchangeOIDCToken exchanges the authorization code for tokens using the
// authorization code grant with client_secret_post authentication.
func exchangeOIDCToken(ctx context.Context, cfg *setting.SSOSetting, tokenEndpoint, code, redirectURI string) (string, string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", cfg.OIDCClientID)
	if cfg.OIDCClientSecret != "" {
		form.Set("client_secret", cfg.OIDCClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", serializer.NewError(serializer.CodeInternalSetting, "Failed to build token request", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: ssoHTTPTimeout}).Do(req)
	if err != nil {
		return "", "", serializer.NewError(serializer.CodeInternalSetting, "Failed to exchange authorization code", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, ssoMaxBodySize))
	if err != nil {
		return "", "", serializer.NewError(serializer.CodeInternalSetting, "Failed to read token response", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", serializer.NewError(serializer.CodeInternalSetting,
			fmt.Sprintf("Token endpoint returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body))), nil)
	}

	var token struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &token); err != nil {
		return "", "", serializer.NewError(serializer.CodeInternalSetting, "Failed to parse token response", err)
	}
	if token.AccessToken == "" {
		return "", "", serializer.NewError(serializer.CodeInternalSetting, "Token response misses access_token", nil)
	}
	return token.AccessToken, token.IDToken, nil
}

// fetchOIDCUserInfo resolves the identity via the provider's userinfo
// endpoint.
func fetchOIDCUserInfo(ctx context.Context, userInfoEndpoint, accessToken string) (*ssoUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoEndpoint, nil)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to build userinfo request", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: ssoHTTPTimeout}).Do(req)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to fetch userinfo", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, ssoMaxBodySize))
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to read userinfo response", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, serializer.NewError(serializer.CodeInternalSetting,
			fmt.Sprintf("Userinfo endpoint returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body))), nil)
	}

	var info ssoUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to parse userinfo response", err)
	}
	return &info, nil
}

// parseIDTokenClaims decodes the payload of a JWT id_token without verifying
// its signature. Only used as a fallback when the provider exposes no userinfo
// endpoint; the id_token is obtained directly from the token endpoint over
// TLS in the code flow.
func parseIDTokenClaims(idToken string) (*ssoUserInfo, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 || idToken == "" {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Provider returned no userinfo endpoint and no id_token", nil)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to decode id_token payload", err)
	}
	var info ssoUserInfo
	if err := json.Unmarshal(payload, &info); err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to parse id_token claims", err)
	}
	return &info, nil
}

// browserLanguage picks the closest supported UI language from the
// Accept-Language header for newly registered users.
func browserLanguage(c *gin.Context) string {
	primary := map[string]string{
		"ar": "ar-AR",
		"de": "de-DE",
		"en": "en-US",
		"es": "es-ES",
		"fr": "fr-FR",
		"it": "it-IT",
		"ja": "ja-JP",
		"ko": "ko-KR",
		"pl": "pl-PL",
		"pt": "pt-BR",
		"ru": "ru-RU",
	}
	for _, part := range strings.Split(c.GetHeader("Accept-Language"), ",") {
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if tag == "" {
			continue
		}
		lower := strings.ToLower(tag)
		if lower == "zh-tw" || lower == "zh-hk" || lower == "zh-mo" {
			return "zh-TW"
		}
		if strings.HasPrefix(lower, "zh") {
			return "zh-CN"
		}
		if lang, ok := primary[strings.SplitN(lower, "-", 2)[0]]; ok {
			return lang
		}
	}
	return "en-US"
}

// resolveQQIdentity performs the QQ Connect token dance: exchange the code
// for an access token, resolve the openid and read the public profile. QQ
// does not return an email address, so a stable synthetic address is derived
// from the openid; SSO sign-in is exempt from the registration email domain
// filter by design.
func resolveQQIdentity(ctx context.Context, cfg *setting.SSOSetting, code, redirectURI string) (*ssoUserInfo, error) {
	get := func(endpoint string, query url.Values) ([]byte, error) {
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}
		u.RawQuery = query.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := (&http.Client{Timeout: ssoHTTPTimeout}).Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, ssoMaxBodySize))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("QQ endpoint returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return body, nil
	}

	// 1. access token
	q := url.Values{}
	q.Set("grant_type", "authorization_code")
	q.Set("client_id", cfg.QQAppID)
	q.Set("client_secret", cfg.QQAppSecret)
	q.Set("code", code)
	q.Set("redirect_uri", redirectURI)
	q.Set("fmt", "json")
	body, err := get("https://graph.qq.com/oauth2.0/access_token", q)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to exchange QQ authorization code", err)
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil || tokenResp.AccessToken == "" {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to parse QQ access token", err)
	}

	// 2. openid
	q = url.Values{}
	q.Set("access_token", tokenResp.AccessToken)
	q.Set("fmt", "json")
	body, err = get("https://graph.qq.com/oauth2.0/me", q)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to resolve QQ openid", err)
	}
	var openResp struct {
		OpenID string `json:"openid"`
	}
	if err := json.Unmarshal(body, &openResp); err != nil || openResp.OpenID == "" {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to parse QQ openid", err)
	}

	// 3. public profile (QQ returns no email; synthesize a stable address)
	q = url.Values{}
	q.Set("access_token", tokenResp.AccessToken)
	q.Set("oauth_consumer_key", cfg.QQAppID)
	q.Set("openid", openResp.OpenID)
	body, err = get("https://graph.qq.com/user/get_user_info", q)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to fetch QQ profile", err)
	}
	var profile struct {
		Nickname string `json:"nickname"`
	}
	_ = json.Unmarshal(body, &profile)

	nick := profile.Nickname
	if nick == "" {
		nick = "QQ User"
	}
	return &ssoUserInfo{
		Sub:      openResp.OpenID,
		Email:    "qq_" + openResp.OpenID + "@qq.sso.local",
		Name:     nick,
		Nickname: nick,
	}, nil
}
