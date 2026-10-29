package admin

import (
	"context"
	"strconv"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

const (
	auditTypeCondition = "event_type"
	auditUserCondition = "event_user_id"
)

type (
	ListAuditLogResponse struct {
		Pagination *inventory.PaginationResults `json:"pagination"`
		Logs       []GetAuditLogResponse        `json:"logs"`
	}
	GetAuditLogResponse struct {
		*ent.AuditLog
		AuditHashID string `json:"audit_hash_id,omitempty"`
		UserHashID  string `json:"user_hash_id,omitempty"`
		UserNick    string `json:"user_nick,omitempty"`
	}
)

// AuditLogs lists audit log entries with optional type/user filters.
func (s *AdminListService) AuditLogs(c *gin.Context) (*ListAuditLogResponse, error) {
	dep := dependency.FromContext(c)
	hasher := dep.HashIDEncoder()

	var (
		err     error
		userID  int
		logType []int
	)

	if v := s.Conditions[auditUserCondition]; v != "" {
		userID, err = strconv.Atoi(v)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid user ID", err)
		}
	}

	if v := s.Conditions[auditTypeCondition]; v != "" {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			t, err := strconv.Atoi(part)
			if err != nil {
				return nil, serializer.NewError(serializer.CodeParamErr, "Invalid event type", err)
			}
			logType = append(logType, t)
		}
	}

	ctx := context.WithValue(c, inventory.LoadAuditLogUser{}, true)
	res, err := dep.AuditLogClient().List(ctx, &inventory.ListAuditLogArgs{
		PaginationArgs: &inventory.PaginationArgs{
			Page:     s.Page - 1,
			PageSize: s.PageSize,
			OrderBy:  s.OrderBy,
			Order:    inventory.OrderDirection(s.OrderDirection),
		},
		Types:  logType,
		UserID: userID,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list audit logs", err)
	}

	return &ListAuditLogResponse{
		Pagination: res.PaginationResults,
		Logs: lo.Map(res.Logs, func(l *ent.AuditLog, _ int) GetAuditLogResponse {
			response := GetAuditLogResponse{
				AuditLog:    l,
				AuditHashID: hashid.EncodeAuditLogID(hasher, l.ID),
			}
			if u := l.Edges.User; u != nil {
				response.UserHashID = hashid.EncodeUserID(hasher, u.ID)
				response.UserNick = u.Nick
			}
			return response
		}),
	}, nil
}

type (
	BatchAuditLogService struct {
		LogIDs []int `json:"ids" binding:"required"`
	}
	BatchAuditLogParamCtx struct{}
)

// Delete removes the audit log entries with the given ids.
func (s *BatchAuditLogService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	if err := dep.AuditLogClient().DeleteBatch(c, s.LogIDs); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete audit logs", err)
	}

	return nil
}
