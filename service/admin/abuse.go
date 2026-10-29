package admin

import (
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

const abuseShareIDCondition = "share_id"

// ListAbuseReports lists filed abuse reports with share and reporter details.
func (s *AdminListService) ListAbuseReports(c *gin.Context) (*ListAbuseReportResponse, error) {
	dep := dependency.FromContext(c)
	abuseClient := dep.AbuseReportClient()
	hasher := dep.HashIDEncoder()

	var (
		shareID int
		err     error
	)
	if s.Conditions[abuseShareIDCondition] != "" {
		shareID, err = strconv.Atoi(s.Conditions[abuseShareIDCondition])
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid share ID", err)
		}
	}

	res, err := abuseClient.List(c, &inventory.ListAbuseReportParameters{
		PaginationArgs: &inventory.PaginationArgs{
			Page:     s.Page - 1,
			PageSize: s.PageSize,
			OrderBy:  s.OrderBy,
			Order:    inventory.OrderDirection(s.OrderDirection),
		},
		ShareID: shareID,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list abuse reports", err)
	}

	siteUrl := dep.SettingProvider().SiteURL(c)

	return &ListAbuseReportResponse{
		Pagination: res.PaginationResults,
		Reports: lo.Map(res.AbuseReports, func(report *ent.AbuseReport, _ int) GetAbuseReportResponse {
			response := GetAbuseReportResponse{AbuseReport: report}

			if share, err := report.Edges.ShareOrErr(); err == nil {
				response.ShareHashID = hashid.EncodeShareID(hasher, share.ID)
				response.ShareLink = routes.MasterShareUrl(siteUrl, response.ShareHashID, share.Password).String()

				if file, err := share.Edges.FileOrErr(); err == nil {
					response.FileName = file.Name
				}

				if owner, err := share.Edges.UserOrErr(); err == nil {
					response.OwnerHashID = hashid.EncodeUserID(hasher, owner.ID)
					response.OwnerName = owner.Nick
				}
			}

			if reporter, err := report.Edges.ReporterOrErr(); err == nil {
				response.ReporterHashID = hashid.EncodeUserID(hasher, reporter.ID)
				response.ReporterName = reporter.Nick
			}

			return response
		}),
	}, nil
}

type (
	BatchAbuseReportService struct {
		ReportIDs []int `json:"ids" binding:"required"`
	}
	BatchAbuseReportParamCtx struct{}
)

// DeleteAbuseReports dismisses (soft deletes) the abuse reports with the given ids.
func DeleteAbuseReports(c *gin.Context, ids []int) error {
	dep := dependency.FromContext(c)
	abuseClient := dep.AbuseReportClient()

	if err := abuseClient.DeleteBatch(c, ids); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to dismiss abuse reports", err)
	}

	return nil
}
