import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem,
  Select,
  SelectChangeEvent,
  Switch,
  Typography,
} from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getUserPolicies, sendMigratePolicy } from "../../../api/api";
import { UserPolicy } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";

export interface MigratePolicyDialogProps {
  open: boolean;
  onClose: () => void;
}

/**
 * MigratePolicyDialog lets users with the migrate_policy group permission
 * start a background task that moves their own files from one storage
 * policy to another. Admins use the policy page for policy-wide migration.
 */
const MigratePolicyDialog = ({ open, onClose }: MigratePolicyDialogProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [policies, setPolicies] = useState<UserPolicy[]>([]);
  const [srcPolicy, setSrcPolicy] = useState<number>(0);
  const [dstPolicy, setDstPolicy] = useState<number>(0);
  const [deleteSource, setDeleteSource] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) return;
    dispatch(getUserPolicies()).then((res) => {
      const list = res.policies ?? [];
      setPolicies(list);
      setSrcPolicy(0);
      setDstPolicy(0);
      setDeleteSource(false);
    });
  }, [open, dispatch]);

  const onSrcChange = useCallback((e: SelectChangeEvent<unknown>) => setSrcPolicy(e.target.value as number), []);
  const onDstChange = useCallback((e: SelectChangeEvent<unknown>) => setDstPolicy(e.target.value as number), []);

  const canSubmit = srcPolicy > 0 && dstPolicy > 0 && srcPolicy !== dstPolicy && !submitting;

  const handleSubmit = useCallback(() => {
    if (!canSubmit) return;
    setSubmitting(true);
    dispatch(sendMigratePolicy({ src_policy_id: srcPolicy, dst_policy_id: dstPolicy, delete_source: deleteSource }))
      .then(() => {
        onClose();
      })
      .finally(() => setSubmitting(false));
  }, [canSubmit, dispatch, srcPolicy, dstPolicy, deleteSource, onClose]);

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{t("dashboard:group.migratePolicy")}</DialogTitle>
      <DialogContent>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {t("fileManager.migratePolicyUserDes")}
        </Typography>
        <FormControl fullWidth sx={{ mb: 2 }}>
          <InputLabel>{t("fileManager.migratePolicySrc")}</InputLabel>
          <Select value={srcPolicy} label={t("fileManager.migratePolicySrc")} onChange={onSrcChange}>
            {policies.map((p) => (
              <MenuItem key={p.id} value={p.id}>
                {p.name}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <FormControl fullWidth sx={{ mb: 2 }}>
          <InputLabel>{t("dashboard:group.migratePolicyDst")}</InputLabel>
          <Select value={dstPolicy} label={t("dashboard:group.migratePolicyDst")} onChange={onDstChange}>
            {policies.map((p) => (
              <MenuItem key={p.id} value={p.id}>
                {p.name}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <FormControl fullWidth>
          <FormControlLabel
            control={<Switch checked={deleteSource} onChange={(e) => setDeleteSource(e.target.checked)} />}
            label={t("dashboard:group.migratePolicyDeleteSource")}
          />
          <Typography variant="caption" color="text.secondary">
            {t("dashboard:group.migratePolicyDeleteSourceWarn")}
          </Typography>
        </FormControl>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t("common:cancel")}</Button>
        <Button variant="contained" disabled={!canSubmit} onClick={handleSubmit}>
          {t("dashboard:group.migratePolicyRun")}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default MigratePolicyDialog;
