package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// AuditLog records security- and activity-relevant events. Type matches the
// AuditLogType values used by the frontend; Detail carries event-specific
// key/value context.
type AuditLog struct {
	ent.Schema
}

// Fields of the AuditLog.
func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").Optional(),
		field.Int("type"),
		field.JSON("detail", map[string]string{}).Optional(),
		field.String("ip").Optional(),
	}
}

// Edges of the AuditLog.
func (AuditLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("audit_logs").
			Field("user_id").
			Unique(),
	}
}

func (AuditLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
