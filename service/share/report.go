package share

import (
	"strconv"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/abusereport"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/audit"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

const (
	// reportThrottleWindow is the minimum interval between two abuse reports
	// filed by the same reporter against the same share.
	reportThrottleWindow = time.Hour
	// reportThrottleCacheTTL is the lifetime (in seconds) of the per-IP throttle
	// entry used for anonymous reporters.
	reportThrottleCacheTTL = 3600
	// reportThrottleKeyPrefix is the cache key prefix for anonymous reporters.
	reportThrottleKeyPrefix = "abuse_report_throttle:"
)

type (
	// ReportShareService files an abuse report against a share.
	ReportShareService struct {
		Reason string `json:"reason" binding:"required,max=1000"`
	}
	ReportShareParamCtx struct{}
)

// Report files an abuse report against the share whose hash id is carried in
// the request context. At most one report per share per reporter/IP is allowed
// within an hour. Logged-in reporters are throttled against the timestamp of
// their latest report; guests are throttled per client IP since their identity
// is not persisted in the report.
func (service *ReportShareService) Report(c *gin.Context) error {
	dep := dependency.FromContext(c)
	shareID := hashid.FromContext(c)
	shareClient := dep.ShareClient()
	abuseClient := dep.AbuseReportClient()

	if _, err := shareClient.GetByID(c, shareID); err != nil {
		if ent.IsNotFound(err) {
			return serializer.NewError(serializer.CodeShareLinkNotFound, "Share not found", err)
		}
		return serializer.NewError(serializer.CodeDBError, "Failed to get share", err)
	}

	reporter := inventory.UserFromContext(c)
	reporterID := 0
	if reporter != nil && !inventory.IsAnonymousUser(reporter) {
		reporterID = reporter.ID
	}

	throttleKey := ""
	if reporterID > 0 {
		latest, err := abuseClient.List(c, &inventory.ListAbuseReportParameters{
			PaginationArgs: &inventory.PaginationArgs{
				PageSize: 1,
				OrderBy:  abusereport.FieldCreatedAt,
				Order:    inventory.OrderDirectionDesc,
			},
			ShareID:    shareID,
			ReporterID: &reporterID,
		})
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to check recent reports", err)
		}

		if len(latest.AbuseReports) > 0 && time.Since(latest.AbuseReports[0].CreatedAt) < reportThrottleWindow {
			return serializer.NewError(serializer.CodeTooManyRequests, "You have already reported this share recently", nil)
		}
	} else {
		throttleKey = reportThrottleKeyPrefix + strconv.Itoa(shareID) + ":" + c.ClientIP()
		if _, throttled := dep.KV().Get(throttleKey); throttled {
			return serializer.NewError(serializer.CodeTooManyRequests, "You have already reported this share recently", nil)
		}
	}

	if _, err := abuseClient.Create(c, shareID, reporterID, service.Reason); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to file abuse report", err)
	}

	audit.Log(c, types.AuditLogTypeReportAbuse, map[string]string{
		"share_id": strconv.Itoa(shareID),
		"reason":   service.Reason,
	})

	if throttleKey != "" {
		// Best-effort throttle record for guests; the report itself is already filed.
		_ = dep.KV().Set(throttleKey, time.Now().Unix(), reportThrottleCacheTTL)
	}

	return nil
}
