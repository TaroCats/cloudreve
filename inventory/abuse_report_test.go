package inventory

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent/abusereport"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/schema"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/stretchr/testify/require"
)

func TestAbuseReportClient(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", filepath.Join(t.TempDir(), "abuse.db"))
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	group := client.Group.Create().SetName("test").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	reporter := client.User.Create().SetEmail("reporter@test.com").SetNick("reporter").
		SetStatus(user.StatusActive).SetGroupID(group.ID).SaveX(ctx)

	share1 := client.Share.Create().SaveX(ctx)
	share2 := client.Share.Create().SaveX(ctx)

	c := NewAbuseReportClient(client, conf.SQLiteDB, nil)

	// File an anonymous report and a logged-in report against share1.
	anonymous, err := c.Create(ctx, share1.ID, 0, "illegal content")
	require.NoError(t, err)
	require.Zero(t, anonymous.ReporterID)

	loggedIn, err := c.Create(ctx, share1.ID, reporter.ID, "copyright infringement")
	require.NoError(t, err)
	require.Equal(t, reporter.ID, loggedIn.ReporterID)

	_, err = c.Create(ctx, share2.ID, 0, "spam")
	require.NoError(t, err)

	// List with share filter and eager loaded share edge.
	res, err := c.List(ctx, &ListAbuseReportParameters{
		PaginationArgs: &PaginationArgs{PageSize: 10},
		ShareID:        share1.ID,
	})
	require.NoError(t, err)
	require.Equal(t, 2, res.TotalItems)
	require.Len(t, res.AbuseReports, 2)
	for _, r := range res.AbuseReports {
		share, err := r.Edges.ShareOrErr()
		require.NoError(t, err)
		require.Equal(t, share1.ID, share.ID)
	}

	// List by reporter and by anonymous reporter.
	res, err = c.List(ctx, &ListAbuseReportParameters{
		PaginationArgs: &PaginationArgs{PageSize: 10},
		ReporterID:     &reporter.ID,
	})
	require.NoError(t, err)
	require.Equal(t, 1, res.TotalItems)
	require.Equal(t, loggedIn.ID, res.AbuseReports[0].ID)
	reporterEdge, err := res.AbuseReports[0].Edges.ReporterOrErr()
	require.NoError(t, err)
	require.Equal(t, reporter.ID, reporterEdge.ID)

	anonymousID := 0
	res, err = c.List(ctx, &ListAbuseReportParameters{
		PaginationArgs: &PaginationArgs{PageSize: 10},
		ReporterID:     &anonymousID,
	})
	require.NoError(t, err)
	require.Equal(t, 2, res.TotalItems)

	// Latest report first, as used by the report throttle check.
	res, err = c.List(ctx, &ListAbuseReportParameters{
		PaginationArgs: &PaginationArgs{PageSize: 1, OrderBy: abusereport.FieldCreatedAt, Order: OrderDirectionDesc},
		ShareID:        share1.ID,
		ReporterID:     &reporter.ID,
	})
	require.NoError(t, err)
	require.Len(t, res.AbuseReports, 1)
	require.Equal(t, loggedIn.ID, res.AbuseReports[0].ID)
	require.WithinDuration(t, time.Now(), res.AbuseReports[0].CreatedAt, time.Minute)

	// Dismiss hides the report from subsequent listings but keeps the row.
	require.NoError(t, c.DeleteBatch(ctx, []int{anonymous.ID}))
	res, err = c.List(ctx, &ListAbuseReportParameters{
		PaginationArgs: &PaginationArgs{PageSize: 10},
		ShareID:        share1.ID,
	})
	require.NoError(t, err)
	require.Equal(t, 1, res.TotalItems)
	require.Equal(t, loggedIn.ID, res.AbuseReports[0].ID)

	kept, err := client.AbuseReport.Query().Where(abusereport.ID(anonymous.ID)).Only(schema.SkipSoftDelete(ctx))
	require.NoError(t, err)
	require.NotNil(t, kept.DeletedAt)
}
