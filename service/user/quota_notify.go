package user

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/crontab"
	"github.com/cloudreve/Cloudreve/v4/pkg/email"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

func init() {
	crontab.Register(setting.CronTypeQuotaNotify, NotifyOverQuotaUsers)
}

// NotifyOverQuotaUsers emails active users whose storage usage exceeds their
// group quota. Each over-quota episode triggers one notification; the flag
// resets automatically once usage drops back under the quota.
func NotifyOverQuotaUsers(ctx context.Context) {
	dep := dependency.FromContext(ctx)
	l := dep.Logger()
	userClient := dep.UserClient()
	groupClient := dep.GroupClient()
	settings := dep.SettingProvider()

	groupCache := make(map[int]*ent.Group)
	page := 0
	const pageSize = 100

	for {
		res, err := userClient.ListUsers(ctx, &inventory.ListUserParameters{
			PaginationArgs: &inventory.PaginationArgs{
				Page:     page,
				PageSize: pageSize,
			},
			Status: user.StatusActive,
		})
		if err != nil {
			l.Warning("Quota notify: failed to list users: %s", err)
			return
		}

		for _, u := range res.Users {
			group, ok := groupCache[u.GroupUsers]
			if !ok {
				group, err = groupClient.GetByID(ctx, u.GroupUsers)
				if err != nil {
					continue
				}
				groupCache[u.GroupUsers] = group
			}

			quota := group.MaxStorage
			over := quota > 0 && u.Storage > quota

			if over && !u.Settings.QuotaNotified {
				title, body, err := email.NewQuotaExceededEmail(ctx, settings, u)
				if err != nil {
					l.Warning("Quota notify: failed to render email for user %d: %s", u.ID, err)
					continue
				}
				if err := dep.EmailClient(ctx).Send(ctx, u.Email, title, body); err != nil {
					l.Warning("Quota notify: failed to send email to user %d: %s", u.ID, err)
					continue
				}
				u.Settings.QuotaNotified = true
				if err := userClient.SaveSettings(ctx, u); err != nil {
					l.Warning("Quota notify: failed to save settings for user %d: %s", u.ID, err)
				}
			} else if !over && u.Settings.QuotaNotified {
				u.Settings.QuotaNotified = false
				if err := userClient.SaveSettings(ctx, u); err != nil {
					l.Warning("Quota notify: failed to reset flag for user %d: %s", u.ID, err)
				}
			}
		}

		if len(res.Users) < pageSize {
			break
		}
		page++
	}
}
