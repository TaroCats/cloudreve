// Package audit records security- and activity-relevant events into the
// audit log, honoring the admin-configured enable switch and event type
// selection.
package audit

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
)

// clientIPGetter is implemented by *gin.Context.
type clientIPGetter interface {
	ClientIP() string
}

// Enabled reports whether events of the given type should be recorded.
// Logging is off until audit_log_enabled is turned on; an empty
// audit_log_types selection records all types.
func Enabled(ctx context.Context, eventType int) bool {
	dep, ok := ctx.Value(dependency.DepCtx{}).(dependency.Dep)
	if !ok {
		return false
	}

	settings := dep.SettingProvider()
	if !settings.AuditLogEnabled(ctx) {
		return false
	}

	allowed := settings.AuditLogTypes(ctx)
	if len(allowed) == 0 {
		return true
	}

	for _, t := range allowed {
		if t == eventType {
			return true
		}
	}

	return false
}

// Log records an audit event. Failures are only logged and never affect the
// main request flow.
func Log(ctx context.Context, eventType int, detail map[string]string) {
	if !Enabled(ctx, eventType) {
		return
	}

	dep, ok := ctx.Value(dependency.DepCtx{}).(dependency.Dep)
	if !ok {
		return
	}

	ip := ""
	if getter, ok := ctx.(clientIPGetter); ok {
		ip = getter.ClientIP()
	}

	if _, err := dep.AuditLogClient().Create(ctx, &inventory.CreateAuditLogParams{
		UserID: inventory.UserIDFromContext(ctx),
		Type:   eventType,
		Detail: detail,
		IP:     ip,
	}); err != nil {
		dep.Logger().Warning("Failed to create audit log for event %d: %s", eventType, err)
	}
}
