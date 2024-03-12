package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func (s *Server) ListTransactions(c *gin.Context) {
	// TODO: Implement transaction list type
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     struct{}{},
		HTMLName: "transaction_list.html",
	})
}
