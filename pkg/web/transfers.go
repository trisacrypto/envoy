package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func (s *Server) LoadTransferPreview(c *gin.Context) {
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     struct{}{},
		HTMLName: "transfer_preview.html",
	})
}
