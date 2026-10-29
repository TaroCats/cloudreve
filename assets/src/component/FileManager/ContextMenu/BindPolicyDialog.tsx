import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  SelectChangeEvent,
  Typography,
} from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getUserPolicies, sendPatchPolicyBinding } from "../../../api/api";
import { UserPolicy } from "../../../api/dashboard";
import { FileResponse } from "../../../api/explorer";
import { useAppDispatch } from "../../../redux/hooks";

export interface BindPolicyDialogProps {
  open: boolean;
  /** Folder the binding applies to. */
  file: FileResponse | null;
  onClose: () => void;
  onSaved?: () => void;
}

/** Sentinel meaning "inherit from the group default" — unbinds any override. */
const POLICY_DEFAULT = 0;

/**
 * BindPolicyDialog pins a folder to a storage policy. Files created under the
 * folder inherit the binding; subfolders inherit it too until they set their own.
 */
const BindPolicyDialog = ({ open, file, onClose, onSaved }: BindPolicyDialogProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [policies, setPolicies] = useState<UserPolicy[]>([]);
  const [policyId, setPolicyId] = useState<number>(POLICY_DEFAULT);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) return;
    dispatch(getUserPolicies()).then((res) => {
      const list = res.policies ?? [];
      setPolicies(list);
      // The file manager's storage_policy.id is a hashid string; match it
      // against the policy list to find the numeric id.
      const currentHash = file?.extended_info?.storage_policy?.id;
      const matched = currentHash ? list.find((p) => p.hash_id === currentHash) : undefined;
      setPolicyId(matched?.id ?? POLICY_DEFAULT);
    });
  }, [open, file, dispatch]);

  const handleChange = useCallback((e: SelectChangeEvent<unknown>) => {
    setPolicyId(e.target.value as number);
  }, []);

  const handleSubmit = useCallback(() => {
    if (!file) return;
    setSubmitting(true);
    dispatch(sendPatchPolicyBinding({ uri: file.path, policy_id: policyId }))
      .then(() => {
        onSaved?.();
        onClose();
      })
      .finally(() => setSubmitting(false));
  }, [dispatch, file, policyId, onSaved, onClose]);

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{t("dashboard:group.bindPolicy")}</DialogTitle>
      <DialogContent>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {t("dashboard:group.bindPolicyDes")}
        </Typography>
        <FormControl fullWidth>
          <InputLabel>{t("dashboard:group.bindPolicy")}</InputLabel>
          <Select value={policyId} label={t("dashboard:group.bindPolicy")} onChange={handleChange}>
            <MenuItem value={POLICY_DEFAULT}>{t("dashboard:group.bindPolicyDefault")}</MenuItem>
            {policies.map((p) => (
              <MenuItem key={p.id} value={p.id}>
                {p.name}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={submitting}>
          {t("common:cancel")}
        </Button>
        <Button variant="contained" onClick={handleSubmit} disabled={submitting}>
          {t("common:confirm")}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default BindPolicyDialog;
