package inventory

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/group"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/require"
)

// TestGroupSinglePolicyToM2MMigration verifies the patch that fans a legacy
// groups.storage_policy_id column out into the storage_policy_groups join
// table, and pins it as the group default so resolution order is preserved.
func TestGroupSinglePolicyToM2MMigration(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", filepath.Join(t.TempDir(), "migration.db"))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	logger := logging.NewConsoleLogger(logging.LevelError)

	// Seed two storage policies.
	p1, err := client.StoragePolicy.Create().SetName("local-primary").SetType("local").Save(ctx)
	require.NoError(t, err)
	p2, err := client.StoragePolicy.Create().SetName("s3-backup").SetType("s3").Save(ctx)
	require.NoError(t, err)

	// Recreate a legacy-shaped groups table: new schema plus the old FK column.
	_, err = client.ExecContext(ctx, "ALTER TABLE groups ADD COLUMN storage_policy_id INTEGER")
	require.NoError(t, err)

	// Legacy group A bound to policy 1; legacy group B bound to policy 2.
	gA := client.Group.Create().
		SetName("LegacyA").
		SetPermissions(&boolset.BooleanSet{}).
		SetSettings(&types.GroupSetting{}).
		SaveX(ctx)
	gB := client.Group.Create().
		SetName("LegacyB").
		SetPermissions(&boolset.BooleanSet{}).
		SetSettings(&types.GroupSetting{}).
		SaveX(ctx)

	_, err = client.ExecContext(ctx,
		"UPDATE groups SET storage_policy_id = ? WHERE id = ?", p1.ID, gA.ID)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx,
		"UPDATE groups SET storage_policy_id = ? WHERE id = ?", p2.ID, gB.ID)
	require.NoError(t, err)

	// Run the patch under test.
	patch := findPatch("migrate_group_single_policy_to_m2m")
	require.NotNil(t, patch, "patch must be registered")
	require.NoError(t, patch.Func(logger, client, ctx))

	// Group A must now be bound to policy 1 and default to it.
	a, err := client.Group.Query().Where(group.ID(gA.ID)).WithStoragePolicies().Only(ctx)
	require.NoError(t, err)
	require.Len(t, a.Edges.StoragePolicies, 1)
	require.Equal(t, p1.ID, a.Edges.StoragePolicies[0].ID)
	require.Equal(t, p1.ID, a.Settings.DefaultPolicyID)

	// Group B likewise.
	b, err := client.Group.Query().Where(group.ID(gB.ID)).WithStoragePolicies().Only(ctx)
	require.NoError(t, err)
	require.Len(t, b.Edges.StoragePolicies, 1)
	require.Equal(t, p2.ID, b.Edges.StoragePolicies[0].ID)
	require.Equal(t, p2.ID, b.Settings.DefaultPolicyID)

	// Idempotency: running again must not duplicate bindings.
	require.NoError(t, patch.Func(logger, client, ctx))
	count, err := client.Group.Query().Where(group.ID(gA.ID)).QueryStoragePolicies().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// A group with no legacy binding must stay unbound.
	gC := client.Group.Create().
		SetName("LegacyC").
		SetPermissions(&boolset.BooleanSet{}).
		SetSettings(&types.GroupSetting{}).
		SaveX(ctx)
	require.NoError(t, patch.Func(logger, client, ctx))
	c, err := client.Group.Query().Where(group.ID(gC.ID)).WithStoragePolicies().Only(ctx)
	require.NoError(t, err)
	require.Empty(t, c.Edges.StoragePolicies)
	require.Zero(t, c.Settings.DefaultPolicyID)
}

// TestGroupMultiPolicyBindingRoundTrip covers the M:N write path used by the
// admin API: a group can bind several policies and name one as default.
func TestGroupMultiPolicyBindingRoundTrip(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", filepath.Join(t.TempDir(), "binding.db"))
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	p1 := client.StoragePolicy.Create().SetName("a").SetType("local").SaveX(ctx)
	p2 := client.StoragePolicy.Create().SetName("b").SetType("s3").SaveX(ctx)

	g := client.Group.Create().
		SetName("Multi").
		SetPermissions(&boolset.BooleanSet{}).
		SetSettings(&types.GroupSetting{DefaultPolicyID: p2.ID, PolicyQuotas: map[int]int64{p1.ID: 1 << 30}}).
		AddStoragePolicyIDs(p1.ID, p2.ID).
		SaveX(ctx)

	loaded, err := client.Group.Query().Where(group.ID(g.ID)).WithStoragePolicies().Only(ctx)
	require.NoError(t, err)
	require.Len(t, loaded.Edges.StoragePolicies, 2)
	require.Equal(t, p2.ID, loaded.Settings.DefaultPolicyID)
	require.Equal(t, int64(1<<30), loaded.Settings.PolicyQuotas[p1.ID])

	// Reverse side: policy 1 must see both groups when bound twice.
	p1Loaded, err := client.StoragePolicy.Query().Where(storagepolicy.ID(p1.ID)).WithGroups().Only(ctx)
	require.NoError(t, err)
	require.Len(t, p1Loaded.Edges.Groups, 1)

	// Replacing the binding set must drop the removed one.
	g2 := client.Group.UpdateOne(loaded).
		ClearStoragePolicies().
		AddStoragePolicyIDs(p2.ID).
		SaveX(ctx)
	require.Len(t, g2.Edges.StoragePolicies, 0) // edges not reloaded
	final, err := client.Group.Query().Where(group.ID(g.ID)).WithStoragePolicies().Only(ctx)
	require.NoError(t, err)
	require.Len(t, final.Edges.StoragePolicies, 1)
	require.Equal(t, p2.ID, final.Edges.StoragePolicies[0].ID)
}

func findPatch(name string) *Patch {
	for i := range patches {
		if patches[i].Name == name {
			return &patches[i]
		}
	}
	return nil
}
