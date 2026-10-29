import { Button, Dialog, DialogActions, DialogContent, DialogTitle, Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useAppSelector } from "../../redux/hooks.ts";

const SEEN_KEY_PREFIX = "announcementSeen_";

/**
 * AnnouncementDialog shows the site announcement to signed-in users.
 * It re-appears whenever the announcement content (hash) changes.
 */
const AnnouncementDialog = () => {
  const { t } = useTranslation();
  const config = useAppSelector((state) => state.siteConfig.basic.config);
  const user = config.user;
  const announcement = config.announcement ?? "";
  const hash = config.announcement_hash ?? "";
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!user || !announcement || !hash) {
      return;
    }
    try {
      if (localStorage.getItem(SEEN_KEY_PREFIX + hash)) {
        return;
      }
    } catch (e) {
      // localStorage unavailable: always show.
    }
    setOpen(true);
  }, [user, announcement, hash]);

  const onClose = () => {
    setOpen(false);
    try {
      localStorage.setItem(SEEN_KEY_PREFIX + hash, "1");
    } catch (e) {
      // ignore
    }
  };

  if (!announcement) {
    return null;
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{t("application:announcement.title")}</DialogTitle>
      <DialogContent>
        <Typography variant="body1" sx={{ whiteSpace: "pre-wrap" }}>
          {announcement}
        </Typography>
      </DialogContent>
      <DialogActions>
        <Button variant="contained" onClick={onClose}>
          {t("application:announcement.gotIt")}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default AnnouncementDialog;
