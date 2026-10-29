import { DialogContent, Stack } from "@mui/material";
import { enqueueSnackbar } from "notistack";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendReportShare } from "../../../../api/api.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { FilledTextField } from "../../../Common/StyledComponents.tsx";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";

export interface ReportShareDialogProps {
  open: boolean;
  onClose: () => void;
  shareID?: string;
}

const ReportShareDialog = ({ open, onClose, shareID }: ReportShareDialogProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [reason, setReason] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      setReason("");
    }
  }, [open]);

  const onAccept = useCallback(() => {
    if (!shareID || !reason.trim()) return;

    setLoading(true);
    dispatch(sendReportShare(shareID, reason.trim()))
      .then(() => {
        enqueueSnackbar({
          message: t("application:modals.reportShareSuccess"),
          variant: "success",
          action: DefaultCloseAction,
        });
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  }, [dispatch, shareID, reason, onClose, t]);

  return (
    <DraggableDialog
      title={t("application:modals.reportShare")}
      showActions
      showCancel
      loading={loading}
      disabled={!reason.trim() || !shareID}
      onAccept={onAccept}
      dialogProps={{
        open: open,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "xs",
      }}
    >
      <DialogContent>
        <Stack spacing={1}>
          <FilledTextField
            variant={"filled"}
            label={t("application:modals.reportShareReason")}
            placeholder={t("application:modals.reportShareReasonPlaceholder")}
            fullWidth
            multiline
            minRows={4}
            required
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            inputProps={{ maxLength: 1000 }}
          />
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};

export default ReportShareDialog;
