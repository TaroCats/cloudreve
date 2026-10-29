import { AuditLogType } from "../../../api/explorer.ts";

/** A group of audit event types rendered together in the settings UI. */
export interface EventCategory {
  id: string;
  /** i18n key of the category title. */
  titleKey: string;
  /** i18n key of the category description. */
  descKey: string;
  /** Chip color used when rendering an event of this category. */
  color: "default" | "primary" | "secondary" | "error" | "info" | "success" | "warning";
  types: number[];
}

/** Event categories; `types` values must match the frontend AuditLogType enum. */
export const eventCategories: EventCategory[] = [
  {
    id: "system",
    titleKey: "settings.systemEvents",
    descKey: "settings.systemEventsDes",
    color: "default",
    types: [AuditLogType.server_start],
  },
  {
    id: "user",
    titleKey: "settings.userEvents",
    descKey: "settings.userEventsDes",
    color: "primary",
    types: [
      AuditLogType.user_signup,
      AuditLogType.user_activated,
      AuditLogType.user_login,
      AuditLogType.user_login_failed,
      AuditLogType.user_token_refresh,
      AuditLogType.user_changed,
      AuditLogType.change_nick,
      AuditLogType.change_avatar,
      AuditLogType.change_password,
      AuditLogType.enable_2fa,
      AuditLogType.disable_2fa,
      AuditLogType.add_passkey,
      AuditLogType.remove_passkey,
      AuditLogType.link_account,
      AuditLogType.unlink_account,
      AuditLogType.oauth_grant_create,
      AuditLogType.oauth_token_exchange,
      AuditLogType.oauth_grant_revoke,
    ],
  },
  {
    id: "file",
    titleKey: "settings.fileEvents",
    descKey: "settings.fileEventsDes",
    color: "success",
    types: [
      AuditLogType.file_create,
      AuditLogType.file_imported,
      AuditLogType.file_rename,
      AuditLogType.entity_uploaded,
      AuditLogType.entity_downloaded,
      AuditLogType.copy_from,
      AuditLogType.copy_to,
      AuditLogType.move_to,
      AuditLogType.delete_file,
      AuditLogType.move_to_trash,
      AuditLogType.update_metadata,
      AuditLogType.get_direct_link,
      AuditLogType.delete_direct_link,
      AuditLogType.update_view,
    ],
  },
  {
    id: "share",
    titleKey: "settings.shareEvents",
    descKey: "settings.shareEventsDes",
    color: "info",
    types: [AuditLogType.share, AuditLogType.share_link_viewed, AuditLogType.edit_share, AuditLogType.delete_share],
  },
  {
    id: "version",
    titleKey: "settings.versionEvents",
    descKey: "settings.versionEventsDes",
    color: "secondary",
    types: [AuditLogType.set_current_version, AuditLogType.delete_version],
  },
  {
    id: "media",
    titleKey: "settings.mediaEvents",
    descKey: "settings.mediaEventsDes",
    color: "warning",
    types: [AuditLogType.thumb_generated, AuditLogType.live_photo_uploaded],
  },
  {
    id: "filesystem",
    titleKey: "settings.filesystemEvents",
    descKey: "settings.filesystemEventsDes",
    color: "default",
    types: [
      AuditLogType.mount,
      AuditLogType.relocate,
      AuditLogType.create_archive,
      AuditLogType.extract_archive,
    ],
  },
  {
    id: "webdav",
    titleKey: "settings.webdavEvents",
    descKey: "settings.webdavEventsDes",
    color: "secondary",
    types: [
      AuditLogType.webdav_login_failed,
      AuditLogType.webdav_account_create,
      AuditLogType.webdav_account_update,
      AuditLogType.webdav_account_delete,
    ],
  },
  {
    id: "email",
    titleKey: "settings.emailEvents",
    descKey: "settings.emailEventsDes",
    color: "info",
    types: [AuditLogType.email_sent],
  },
];

const typeNameByValue: Record<number, string> = {};
Object.entries(AuditLogType).forEach(([name, value]) => {
  typeNameByValue[value as number] = name;
});

const categoryByType: Record<number, EventCategory> = {};
eventCategories.forEach((category) => {
  category.types.forEach((type) => {
    categoryByType[type] = category;
  });
});

/**
 * Returns the event key name (e.g. "user_login") for a numeric event type,
 * falling back to a synthetic name for unknown types.
 */
export function getEventName(type: number): string {
  return typeNameByValue[type] ?? `type_${type}`;
}

/** Returns the category an event type belongs to, if any. */
export function getEventCategory(type: number): EventCategory | undefined {
  return categoryByType[type];
}
