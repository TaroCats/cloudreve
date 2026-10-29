import {
  Alert,
  Button,
  Checkbox,
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
  Typography,
} from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getStoragePolicyList, sendMigratePolicy } from "../../../api/api";
import { StoragePolicy } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";

export interface MigratePolicyDialogProps {
  open: boolean;
  /** Policy entities are migrated away from. */
  sourcePolicy: StoragePolicy | null;
  onClose: () => void;
}

/**
 * MigratePolicyDialog starts a background job that copies every entity off one
 * storage policy onto another. The source policy is left in place unless
 * "delete source objects" is ticked.
 */
const MigratePolicyDialog = ({ open, sourcePolicy, onClose }: MigratePolicyDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [policies, setPolicies] = useState<StoragePolicy[]>([]);
  const [dstPolicyId, setDstPolicyId] = useState<number>(0);
  const [deleteSource, setDeleteSource] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) return;
    dispatch(getStoragePolicyList({ page: 1, page_size: 1000, order_by: "id", order_direction: "asc" })).then((res) =>
      setPolicies(res.policies ?? []),
    );
    setDstPolicyId(0);
    setDeleteSource(false);
  }, [open, dispatch]);

  const handleSubmit = useCallback(() => {
    if (!sourcePolicy || dstPolicyId <= 0) return;
    setSubmitting(true);
    dispatch(
      sendMigratePolicy({
        src_policy_id: sourcePolicy.id,
        dst_policy_id: dstPolicyId,
        delete_source: deleteSource,
      }),
    )
      .then(() => onClose())
      .finally(() => setSubmitting(false));
  }, [dispatch, sourcePolicy, dstPolicyId, deleteSource, onClose]);

  const onDstChange = useCallback((e: SelectChangeEvent<unknown>) => {
    setDstPolicyId(e.target.value as number);
  }, []);

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{t("group.migratePolicy")}</DialogTitle>
      <DialogContent>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {t("group.migratePolicyDes")}
        </Typography>

        <Alert severity="info" sx={{ mb: 2 }}>
          {t("group.migratePolicySource", { name: sourcePolicy?.name ?? "" })}
        </Alert>

        <FormControl fullWidth>
          <InputLabel>{t("group.migratePolicyDst")}</InputLabel>
          <Select value={dstPolicyId} label={t("group.migratePolicyDst")} onChange={onDstChange}>
            {policies
              .filter((p) => p.id !== sourcePolicy?.id)
              .map((p) => (
                <MenuItem key={p.id} value={p.id}>
                  {p.name}
                </MenuItem>
              ))}
          </Select>
        </FormControl>

        <FormControlLabel
          sx={{ mt: 2 }}
          control={<Checkbox checked={deleteSource} onChange={(e) => setDeleteSource(e.target.checked)} />}
          label={t("group.migratePolicyDeleteSource")}
        />
        {deleteSource && (
          <Alert severity="warning" sx={{ mt: 1 }}>
            {t("group.migratePolicyDeleteSourceWarn")}
          </Alert>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={submitting}>
          {t("common:cancel")}
        </Button>
        <Button variant="contained" onClick={handleSubmit} disabled={dstPolicyId <= 0 || submitting}>
          {t("group.migratePolicyRun")}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default MigratePolicyDialog;
