package user

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-gonic/gin"
)

func TestNormalizeProvider(t *testing.T) {
	for in, want := range map[string]string{
		"oidc":   SSOProviderOIDC,
		"Logto":  SSOProviderOIDC,
		"LOGTO":  SSOProviderOIDC,
		"qq":     SSOProviderQQ,
		"QQ":     SSOProviderQQ,
	} {
		got, err := normalizeProvider(in)
		if err != nil || got != want {
			t.Fatalf("normalizeProvider(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := normalizeProvider("wechat"); err == nil {
		t.Fatal("normalizeProvider(wechat) should fail")
	}
}

func TestOIDCFlowHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": "http://" + r.Host + "/authorize",
			"token_endpoint":         "http://" + r.Host + "/token",
			"userinfo_endpoint":      "http://" + r.Host + "/userinfo",
		})
	})
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil || r.PostFormValue("grant_type") != "authorization_code" ||
			r.PostFormValue("code") != "test-code" || r.PostFormValue("client_secret") != "secret" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "at", "id_token": "idt"})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer at" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"sub": "user-1", "email": "a@b.c", "name": "Alice"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx := &gin.Context{Request: httptest.NewRequest(http.MethodGet, "/", nil)}

	meta, err := discoverOIDC(ctx, srv.URL)
	if err != nil {
		t.Fatalf("discoverOIDC: %v", err)
	}
	if meta.AuthorizationEndpoint != srv.URL+"/authorize" || meta.TokenEndpoint != srv.URL+"/token" {
		t.Fatalf("unexpected discovery result: %+v", meta)
	}

	cfg := &setting.SSOSetting{OIDCClientID: "cid", OIDCClientSecret: "secret"}
	at, idt, err := exchangeOIDCToken(ctx, cfg, meta.TokenEndpoint, "test-code", srv.URL+"/callback")
	if err != nil || at != "at" || idt != "idt" {
		t.Fatalf("exchangeOIDCToken = %q, %q, %v", at, idt, err)
	}

	info, err := fetchOIDCUserInfo(ctx, meta.UserInfoEndpoint, at)
	if err != nil {
		t.Fatalf("fetchOIDCUserInfo: %v", err)
	}
	if info.Sub != "user-1" || info.Email != "a@b.c" {
		t.Fatalf("unexpected userinfo: %+v", info)
	}
}

func TestParseIDTokenClaims(t *testing.T) {
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"s1","email":"e@x.y"}`))
	token := "eyJhbGciOiJub25lIn0." + payload + ".sig"
	info, err := parseIDTokenClaims(token)
	if err != nil {
		t.Fatalf("parseIDTokenClaims: %v", err)
	}
	if info.Sub != "s1" || info.Email != "e@x.y" {
		t.Fatalf("unexpected claims: %+v", info)
	}
	if _, err := parseIDTokenClaims("garbage"); err == nil {
		t.Fatal("parseIDTokenClaims should reject garbage")
	}
}

func TestBrowserLanguage(t *testing.T) {
	for header, want := range map[string]string{
		"zh-CN,zh;q=0.9":   "zh-CN",
		"zh-TW":            "zh-TW",
		"fr-FR,fr;q=0.8":   "fr-FR",
		"xx-YY":            "en-US",
		"":                 "en-US",
		"ja-JP,ja;q=0.9,en": "ja-JP",
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Accept-Language", header)
		if got := browserLanguage(c); got != want {
			t.Fatalf("browserLanguage(%q) = %q; want %q", header, got, want)
		}
	}
}
