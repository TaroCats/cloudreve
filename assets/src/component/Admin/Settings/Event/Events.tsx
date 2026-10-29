import { Box, Checkbox, FormControlLabel, FormGroup, Stack, Switch, Typography } from "@mui/material";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { isTrueVal } from "../../../../session/utils.ts";
import { eventCategories, getEventName } from "../../Event/eventTypes.ts";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../Settings.tsx";
import { SettingContext } from "../SettingWrapper.tsx";

/** Audit log event selection settings. */
const Events = () => {
  const { t } = useTranslation("dashboard");
  const { setSettings, values } = useContext(SettingContext);

  const enabled = isTrueVal(values.audit_log_enabled ?? "0");
  const selectedTypes = useMemo(() => {
    try {
      const parsed = JSON.parse(values.audit_log_types ?? "[]");
      return Array.isArray(parsed) ? (parsed as number[]) : [];
    } catch {
      return [];
    }
  }, [values.audit_log_types]);

  const updateTypes = (types: number[]) => {
    const sorted = Array.from(new Set(types)).sort((a, b) => a - b);
    setSettings({ audit_log_types: JSON.stringify(sorted) });
  };

  const toggleType = (type: number, checked: boolean) => {
    if (checked) {
      updateTypes([...selectedTypes, type]);
    } else {
      updateTypes(selectedTypes.filter((v) => v !== type));
    }
  };

  const toggleCategory = (types: number[], checked: boolean) => {
    if (checked) {
      updateTypes([...selectedTypes, ...types]);
    } else {
      updateTypes(selectedTypes.filter((v) => !types.includes(v)));
    }
  };

  return (
    <Box component={"form"} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.auditLog")}
          </Typography>
          <SettingSectionContent>
            <Box>
              <FormControlLabel
                control={
                  <Switch
                    checked={enabled}
                    onChange={(e) => setSettings({ audit_log_enabled: e.target.checked ? "1" : "0" })}
                  />
                }
                label={t("settings.auditLogEnabled")}
              />
              <NoMarginHelperText>{t("settings.auditLogEnabledDes")}</NoMarginHelperText>
            </Box>
            <Box>
              <Typography variant="subtitle2" gutterBottom>
                {t("settings.auditLogTypes")}
              </Typography>
              <NoMarginHelperText sx={{ mb: 1 }}>{t("settings.auditLogTypesDes")}</NoMarginHelperText>
              <Stack spacing={2}>
                {eventCategories.map((category) => {
                  const selectedInCategory = category.types.filter((type) => selectedTypes.includes(type));
                  return (
                    <Box key={category.id}>
                      <FormControlLabel
                        control={
                          <Checkbox
                            size="small"
                            disabled={!enabled}
                            checked={selectedInCategory.length === category.types.length}
                            indeterminate={
                              selectedInCategory.length > 0 && selectedInCategory.length < category.types.length
                            }
                            onChange={(e) => toggleCategory(category.types, e.target.checked)}
                          />
                        }
                        label={t(category.titleKey)}
                      />
                      <NoMarginHelperText>{t(category.descKey)}</NoMarginHelperText>
                      <FormGroup row sx={{ ml: 2 }}>
                        {category.types.map((type) => (
                          <FormControlLabel
                            key={type}
                            control={
                              <Checkbox
                                size="small"
                                disabled={!enabled}
                                checked={selectedTypes.includes(type)}
                                onChange={(e) => toggleType(type, e.target.checked)}
                              />
                            }
                            label={t(`settings.event.${getEventName(type)}`, { defaultValue: getEventName(type) })}
                          />
                        ))}
                      </FormGroup>
                    </Box>
                  );
                })}
              </Stack>
            </Box>
          </SettingSectionContent>
        </SettingSection>
      </Stack>
    </Box>
  );
};

export default Events;
