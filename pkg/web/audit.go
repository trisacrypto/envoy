package web

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/scene"
	"go.rtnl.ai/ulid"
)

func (s *Server) ListComplianceAuditLogs(c *gin.Context) {
	var (
		err  error
		in   *api.ComplianceAuditLogQuery
		page *models.ComplianceAuditLogPage
		out  *api.ComplianceAuditLogList
	)

	// Parse the URL parameters from the input request
	in = &api.ComplianceAuditLogQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse page query request"))
		return
	}

	if err = in.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// TODO: implement better pagination mechanism (with pagination tokens)

	if page, err = s.store.ListComplianceAuditLogs(c.Request.Context(), in.Query()); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process compliance audit log list request"))
		return
	}

	// Convert the counterparties page into an api response
	if out, err = api.NewComplianceAuditLogList(page); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process compliance audit log list request"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/complianceauditlogs/list.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) ComplianceAuditLogDetail(c *gin.Context) {
	var (
		err   error
		logID ulid.ULID
		log   *models.ComplianceAuditLog
		out   *api.ComplianceAuditLog
	)

	// Parse the logID passed in from the URL
	if logID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("compliance audit log not found"))
		return
	}

	// Fetch the model from the database
	if log, err = s.store.RetrieveComplianceAuditLog(c.Request.Context(), logID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("compliance audit log not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model into an API response
	out = api.NewComplianceAuditLog(log)

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/complianceauditlogs/detail.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}
