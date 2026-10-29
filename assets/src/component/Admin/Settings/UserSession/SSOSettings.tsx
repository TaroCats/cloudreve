import ExpandMore from "@mui/icons-material/ExpandMore";
import {
  Accordion,
  AccordionDetails,
  AccordionSummaryProps,
  Box,
  FormControl,
  FormControlLabel,
  Link,
  Stack,
  styled,
  Switch,
  Typography,
} from "@mui/material";
import MuiAccordionSummary from "@mui/material/AccordionSummary";
import { useEffect, useMemo, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { getSettings, sendSetSetting } from "../../../../api/api.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { isTrueVal } from "../../../../session/utils.ts";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../Settings.tsx";
import { SavingFloat } from "../SettingWrapper.tsx";

export const AccordionSummary = styled((props: AccordionSummaryProps) => <MuiAccordionSummary {...props} />)(
  ({ theme }) => ({
    fontSize: theme.typography.body2.fontSize,
    paddingLeft: theme.spacing(4),
    "& .MuiFormControlLabel-label": {
      fontSize: theme.typography.body2.fontSize,
    },
    "& .MuiCheckbox-root": {
      marginRight: theme.spacing(2),
    },
  }),
);

export const StyledAccordion = styled(Accordion)(({ theme }) => ({
  boxShadow: "none",
  border: `1px solid ${theme.palette.divider}`,
  "&::before": {
    display: "none",
  },
}));

export interface SettingSectionProps {}

const settingKeys = [
  "oidc_enabled",
  "oidc_issuer",
  "oidc_client_id",
  "oidc_client_secret",
  "oidc_direct_sign_in",
  "qq_enabled",
  "qq_app_id",
  "qq_app_secret",
];

// OIDCFields renders the shared oidc_* form fields. Logto is a preset of the
// generic OIDC provider, so both accordions edit the same settings; only the
// enable label and helper text differ.
const OIDCFields = ({
  values,
  setSettings,
  preset,
}: {
  values: { [key: string]: string };
  setSettings: (settings: { [key: string]: string }) => void;
  preset: "logto" | "oidc";
}) => {
  const { t } = useTranslation("dashboard");
  const callbackUrl = `${window.location.origin}/api/v4/session/sso/oidc/callback`;
  return (
    <Stack spacing={2} sx={{ maxWidth: 640 }}>
      <FormControl fullWidth>
        <FormControlLabel
          control={
            <Switch
              checked={isTrueVal(values.oidc_enabled ?? "0")}
              onChange={(e) => setSettings({ oidc_enabled: e.target.checked ? "1" : "0" })}
            />
          }
          label={preset === "logto" ? t("settings.logto") : t("settings.oidc")}
        />
        <NoMarginHelperText>
          {preset === "logto" ? (
            <Trans
              i18nKey="settings.logtoDes"
              ns={"dashboard"}
              values={{ url: callbackUrl }}
              components={[
                <Link href={"https://logto.io"} target={"_blank"} underline={"hover"} />,
                <Link href={callbackUrl} underline={"hover"} />,
              ]}
            />
          ) : (
            <Trans
              i18nKey="settings.oidcDes"
              ns={"dashboard"}
              values={{ url: callbackUrl }}
              components={[
                <Link href={callbackUrl} underline={"hover"} />,
                <Link href={"https://openid.net/connect/"} target={"_blank"} underline={"hover"} />,
              ]}
            />
          )}
        </NoMarginHelperText>
      </FormControl>
      <FormControl fullWidth>
        <DenseFilledTextField
          value={values.oidc_issuer ?? ""}
          onChange={(e) => setSettings({ oidc_issuer: e.target.value })}
          label={t("settings.oidcWellknownUrl")}
          required
        />
        <NoMarginHelperText>{t("settings.oidcWellknownDes")}</NoMarginHelperText>
      </FormControl>
      <FormControl fullWidth>
        <DenseFilledTextField
          value={values.oidc_client_id ?? ""}
          onChange={(e) => setSettings({ oidc_client_id: e.target.value })}
          label={t("settings.clientID")}
          required
        />
        <NoMarginHelperText>{t("settings.clientIDDes")}</NoMarginHelperText>
      </FormControl>
      <FormControl fullWidth>
        <DenseFilledTextField
          value={values.oidc_client_secret ?? ""}
          onChange={(e) => setSettings({ oidc_client_secret: e.target.value })}
          label={t("settings.clientSecret")}
          type={"password"}
        />
        <NoMarginHelperText>{t("settings.clientSecretDes")}</NoMarginHelperText>
      </FormControl>
      <FormControl fullWidth>
        <FormControlLabel
          control={
            <Switch
              checked={isTrueVal(values.oidc_direct_sign_in ?? "0")}
              onChange={(e) => setSettings({ oidc_direct_sign_in: e.target.checked ? "1" : "0" })}
            />
          }
          label={t("settings.ssoDirectSignIn")}
        />
        <NoMarginHelperText>{t("settings.ssoDirectSignInDes")}</NoMarginHelperText>
      </FormControl>
    </Stack>
  );
};

// SSOSettings is the admin form for third-party sign-in providers. It loads
// and saves its keys through the admin settings API directly, so it does not
// depend on the surrounding SettingsWrapper context.
const SSOSettings = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const formRef = useRef<HTMLFormElement>(null);
  const [values, setValues] = useState<{ [key: string]: string }>({});
  const [modifiedValues, setModifiedValues] = useState<{ [key: string]: string }>({});
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  const setSettings = (settings: { [key: string]: string }) => {
    setModifiedValues((prev) => ({ ...prev, ...settings }));
  };

  useEffect(() => {
    dispatch(getSettings({ keys: settingKeys }))
      .then((res) => {
        setValues(res);
        setModifiedValues(res);
      })
      .finally(() => setLoading(false));
  }, []);

  const showSaveButton = useMemo(() => {
    return JSON.stringify(modifiedValues) !== JSON.stringify(values);
  }, [modifiedValues, values]);

  const revert = () => {
    setModifiedValues(values);
  };

  const submit = () => {
    if (formRef.current) {
      if (!formRef.current.checkValidity()) {
        formRef.current.reportValidity();
        return;
      }
    }

    const modified: { [key: string]: string } = {};
    Object.keys(modifiedValues).forEach((key) => {
      if (modifiedValues[key] !== values[key]) {
        modified[key] = modifiedValues[key];
      }
    });

    setSubmitting(true);
    dispatch(sendSetSetting({ settings: modified }))
      .then((res) => {
        setValues((s) => ({ ...s, ...res }));
        setModifiedValues((s) => ({ ...s, ...res }));
      })
      .finally(() => setSubmitting(false));
  };

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("settings.thirdPartySignIn")}
      </Typography>
      <SettingSectionContent>
        <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
          <Stack spacing={2}>
            <StyledAccordion disableGutters>
              <AccordionSummary expandIcon={<ExpandMore />}>{t("settings.qqConnect")}</AccordionSummary>
              <AccordionDetails>
                <Stack spacing={2} sx={{ maxWidth: 640 }}>
                  <FormControl fullWidth>
                    <FormControlLabel
                      control={
                        <Switch
                          checked={isTrueVal(modifiedValues.qq_enabled ?? "0")}
                          onChange={(e) => setSettings({ qq_enabled: e.target.checked ? "1" : "0" })}
                          disabled={loading}
                        />
                      }
                      label={t("settings.qqConnect")}
                    />
                    <NoMarginHelperText>{t("settings.qqConnectDes")}</NoMarginHelperText>
                  </FormControl>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      value={modifiedValues.qq_app_id ?? ""}
                      onChange={(e) => setSettings({ qq_app_id: e.target.value })}
                      label={t("settings.qqAppID")}
                      disabled={loading}
                    />
                  </FormControl>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      value={modifiedValues.qq_app_secret ?? ""}
                      onChange={(e) => setSettings({ qq_app_secret: e.target.value })}
                      label={t("settings.qqAppSecret")}
                      type={"password"}
                      disabled={loading}
                    />
                  </FormControl>
                </Stack>
              </AccordionDetails>
            </StyledAccordion>
            <StyledAccordion disableGutters>
              <AccordionSummary expandIcon={<ExpandMore />}>{t("settings.logto")}</AccordionSummary>
              <AccordionDetails>
                <OIDCFields values={modifiedValues} setSettings={setSettings} preset={"logto"} />
              </AccordionDetails>
            </StyledAccordion>
            <StyledAccordion disableGutters>
              <AccordionSummary expandIcon={<ExpandMore />}>{t("settings.oidc")}</AccordionSummary>
              <AccordionDetails>
                <OIDCFields values={modifiedValues} setSettings={setSettings} preset={"oidc"} />
              </AccordionDetails>
            </StyledAccordion>
          </Stack>
          <SavingFloat in={showSaveButton && !loading} submitting={submitting} revert={revert} submit={submit} />
        </Box>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default SSOSettings;
