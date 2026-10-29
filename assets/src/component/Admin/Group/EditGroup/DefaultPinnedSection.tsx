import { Box, FormControl, Link, Stack, TextField, Typography } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { GroupEnt } from "../../../../api/dashboard";
import SharesInput, { ShareOption } from "../../Common/SharesInput";
import SettingForm from "../../../Pages/Setting/SettingForm";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings";
import { AnonymousGroupID } from "../GroupRow";
import { GroupSettingContext } from "./GroupSettingWrapper";

const DefaultPinnedSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setGroup } = useContext(GroupSettingContext);

  const pinned = values.settings?.default_pinned ?? [];

  const shareOptions = useMemo<ShareOption[]>(
    () => pinned.map((p) => ({ id: p.share_id, name: p.name })),
    [pinned],
  );

  const onSharesChange = useCallback(
    (options: ShareOption[]) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        settings: {
          ...p.settings,
          default_pinned: options.map((o) => {
            const existing = pinned.find((x) => x.share_id === o.id);
            return { share_id: o.id, name: existing?.name ?? o.name ?? "" };
          }),
        },
      }));
    },
    [setGroup, pinned],
  );

  const onNameChange = useCallback(
    (shareId: number, name: string) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        settings: {
          ...p.settings,
          default_pinned: (p.settings?.default_pinned ?? []).map((x) =>
            x.share_id === shareId ? { ...x, name } : x,
          ),
        },
      }));
    },
    [setGroup],
  );

  if (values?.id == AnonymousGroupID) {
    return null;
  }

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("group.defaultPinned")}
      </Typography>
      <SettingSectionContent>
        <SettingForm lgWidth={5}>
          <FormControl fullWidth>
            <SharesInput value={shareOptions} onChange={onSharesChange} />
            <NoMarginHelperText>
              <Trans
                i18nKey="group.defaultPinnedDes"
                ns={"dashboard"}
                components={[<Link component={RouterLink} to={"/admin/share"} />]}
              />
            </NoMarginHelperText>
          </FormControl>
        </SettingForm>
        {pinned.length > 0 && (
          <SettingForm title={t("group.pinnedDisplayName")} lgWidth={5}>
            <Stack spacing={1}>
              {pinned.map((p) => (
                <Box key={p.share_id} sx={{ display: "flex", gap: 1, alignItems: "center" }}>
                  <Typography variant="body2" color="text.secondary" sx={{ minWidth: 80 }}>
                    #{p.share_id}
                  </Typography>
                  <TextField
                    size="small"
                    fullWidth
                    value={p.name ?? ""}
                    placeholder={t("group.pinnedShare")}
                    onChange={(e) => onNameChange(p.share_id, e.target.value)}
                  />
                </Box>
              ))}
            </Stack>
          </SettingForm>
        )}
      </SettingSectionContent>
    </SettingSection>
  );
};

export default DefaultPinnedSection;
