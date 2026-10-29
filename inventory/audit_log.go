package inventory

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/auditlog"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
)

type (
	// Ctx keys for eager loading options.
	LoadAuditLogUser struct{}

	AuditLogClient interface {
		TxOperator
		// Create inserts a new audit log entry.
		Create(ctx context.Context, params *CreateAuditLogParams) (*ent.AuditLog, error)
		// List returns a paginated list of audit logs, with user edge eager-loaded.
		List(ctx context.Context, args *ListAuditLogArgs) (*ListAuditLogResult, error)
		// DeleteBatch deletes audit logs with the given ids.
		DeleteBatch(ctx context.Context, ids []int) error
	}

	CreateAuditLogParams struct {
		// UserID is the user who triggered the event. 0 means system event.
		UserID int
		Type   int
		Detail map[string]string
		IP     string
	}

	ListAuditLogArgs struct {
		*PaginationArgs
		// Types filters by event types. Empty means all types.
		Types []int
		// UserID filters by triggering user. 0 means all users.
		UserID int
	}

	ListAuditLogResult struct {
		*PaginationResults
		Logs []*ent.AuditLog
	}
)

func NewAuditLogClient(client *ent.Client, dbType conf.DBType) AuditLogClient {
	return &auditLogClient{
		client:      client,
		maxSQlParam: sqlParamLimit(dbType),
	}
}

type auditLogClient struct {
	maxSQlParam int
	client      *ent.Client
}

func (c *auditLogClient) SetClient(newClient *ent.Client) TxOperator {
	return &auditLogClient{client: newClient, maxSQlParam: c.maxSQlParam}
}

func (c *auditLogClient) GetClient() *ent.Client {
	return c.client
}

func (c *auditLogClient) Create(ctx context.Context, params *CreateAuditLogParams) (*ent.AuditLog, error) {
	query := c.client.AuditLog.Create().SetType(params.Type)
	if params.UserID > 0 {
		query.SetUserID(params.UserID)
	}
	if len(params.Detail) > 0 {
		query.SetDetail(params.Detail)
	}
	if params.IP != "" {
		query.SetIP(params.IP)
	}

	res, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit log: %w", err)
	}

	return res, nil
}

func (c *auditLogClient) DeleteBatch(ctx context.Context, ids []int) error {
	_, err := c.client.AuditLog.Delete().Where(auditlog.IDIn(ids...)).Exec(ctx)
	return err
}

func (c *auditLogClient) List(ctx context.Context, args *ListAuditLogArgs) (*ListAuditLogResult, error) {
	query := c.client.AuditLog.Query()
	if len(args.Types) > 0 {
		query.Where(auditlog.TypeIn(args.Types...))
	}
	if args.UserID > 0 {
		query.Where(auditlog.HasUserWith(user.ID(args.UserID)))
	}
	if v, ok := ctx.Value(LoadAuditLogUser{}).(bool); ok && v {
		query.WithUser(func(q *ent.UserQuery) {
			withUserEagerLoading(ctx, q)
		})
	}

	orderTerm := getOrderTerm(args.Order)
	switch args.OrderBy {
	case auditlog.FieldCreatedAt:
		query.Order(auditlog.ByCreatedAt(orderTerm), auditlog.ByID(orderTerm))
	case auditlog.FieldType:
		query.Order(auditlog.ByType(orderTerm), auditlog.ByID(orderTerm))
	default:
		query.Order(auditlog.ByID(orderTerm))
	}

	pageSize := args.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count audit logs: %w", err)
	}

	logs, err := query.Limit(pageSize).Offset(args.Page * pageSize).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return &ListAuditLogResult{
		PaginationResults: &PaginationResults{
			PageSize:   pageSize,
			TotalItems: total,
			Page:       args.Page,
		},
		Logs: logs,
	}, nil
}
