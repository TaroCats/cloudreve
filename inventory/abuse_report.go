package inventory

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/abusereport"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
)

type (
	AbuseReportClient interface {
		TxOperator
		// Create files a new abuse report against the given share. reporterID is
		// optional; 0 means the report is filed anonymously.
		Create(ctx context.Context, shareID, reporterID int, reason string) (*ent.AbuseReport, error)
		// List returns a list of abuse reports with the given args. Share and
		// reporter edges are eager loaded.
		List(ctx context.Context, args *ListAbuseReportParameters) (*ListAbuseReportResult, error)
		// DeleteBatch dismisses (soft deletes) the abuse reports with the given ids.
		DeleteBatch(ctx context.Context, ids []int) error
	}

	ListAbuseReportParameters struct {
		*PaginationArgs
		// ShareID filters by the reported share. 0 means no filtering.
		ShareID int
		// ReporterID filters by reporter. A pointer to 0 matches reports filed
		// anonymously (without a reporter). Nil means no filtering.
		ReporterID *int
	}

	ListAbuseReportResult struct {
		*PaginationResults
		AbuseReports []*ent.AbuseReport
	}
)

func NewAbuseReportClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) AbuseReportClient {
	return &abuseReportClient{
		client:      client,
		hasher:      hasher,
		maxSQlParam: sqlParamLimit(dbType),
	}
}

type abuseReportClient struct {
	maxSQlParam int
	client      *ent.Client
	hasher      hashid.Encoder
}

func (c *abuseReportClient) SetClient(newClient *ent.Client) TxOperator {
	return &abuseReportClient{client: newClient, hasher: c.hasher, maxSQlParam: c.maxSQlParam}
}

func (c *abuseReportClient) GetClient() *ent.Client {
	return c.client
}

func (c *abuseReportClient) Create(ctx context.Context, shareID, reporterID int, reason string) (*ent.AbuseReport, error) {
	query := c.client.AbuseReport.Create().
		SetShareID(shareID).
		SetReason(reason)
	if reporterID > 0 {
		query.SetReporterID(reporterID)
	}

	report, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create abuse report for share %d: %w", shareID, err)
	}

	return report, nil
}

func (c *abuseReportClient) DeleteBatch(ctx context.Context, ids []int) error {
	_, err := c.client.AbuseReport.Delete().Where(abusereport.IDIn(ids...)).Exec(ctx)
	return err
}

func (c *abuseReportClient) List(ctx context.Context, args *ListAbuseReportParameters) (*ListAbuseReportResult, error) {
	query := c.listQuery(args)
	pageSize := capPageSize(c.maxSQlParam, args.PageSize, 10)
	query.Order(getAbuseReportOrderOption(args)...)

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count abuse reports: %w", err)
	}

	reports, err := query.Limit(pageSize).Offset(args.Page * args.PageSize).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list abuse reports: %w", err)
	}

	return &ListAbuseReportResult{
		AbuseReports: reports,
		PaginationResults: &PaginationResults{
			PageSize:   pageSize,
			TotalItems: total,
			Page:       args.Page,
		},
	}, nil
}

func (c *abuseReportClient) listQuery(args *ListAbuseReportParameters) *ent.AbuseReportQuery {
	query := c.client.AbuseReport.Query().
		WithShare(func(q *ent.ShareQuery) {
			q.WithFile().WithUser()
		}).
		WithReporter()

	if args.ShareID > 0 {
		query.Where(abusereport.ShareIDEQ(args.ShareID))
	}

	if args.ReporterID != nil {
		if *args.ReporterID == 0 {
			query.Where(abusereport.Or(abusereport.ReporterIDIsNil(), abusereport.ReporterIDEQ(0)))
		} else {
			query.Where(abusereport.ReporterIDEQ(*args.ReporterID))
		}
	}

	return query
}

func getAbuseReportOrderOption(args *ListAbuseReportParameters) []abusereport.OrderOption {
	orderTerm := getOrderTerm(args.Order)
	switch args.OrderBy {
	case abusereport.FieldCreatedAt:
		return []abusereport.OrderOption{abusereport.ByCreatedAt(orderTerm), abusereport.ByID(orderTerm)}
	case abusereport.FieldReason:
		return []abusereport.OrderOption{abusereport.ByReason(orderTerm), abusereport.ByID(orderTerm)}
	default:
		return []abusereport.OrderOption{abusereport.ByID(orderTerm)}
	}
}
