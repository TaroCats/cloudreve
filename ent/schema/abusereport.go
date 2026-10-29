package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// AbuseReport holds abuse reports filed against shares. Dismissing a report
// is a soft delete via CommonMixin.
type AbuseReport struct {
	ent.Schema
}

// Fields of the AbuseReport.
func (AbuseReport) Fields() []ent.Field {
	return []ent.Field{
		field.Int("share_id"),
		field.Int("reporter_id").Optional(),
		field.Text("reason"),
	}
}

// Edges of the AbuseReport.
func (AbuseReport) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("share", Share.Type).
			Ref("abuse_reports").
			Field("share_id").
			Unique().
			Required(),
		edge.From("reporter", User.Type).
			Ref("abuse_reports").
			Field("reporter_id").
			Unique(),
	}
}

func (AbuseReport) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
