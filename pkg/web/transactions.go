package web

import (
	"net/http"
	api "self-hosted-node/pkg/web/api/v1"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func (s *Server) ListTransactions(c *gin.Context) {
	var out api.AccountsList
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_list.html",
	})
}

func (s *Server) CreateTransaction(c *gin.Context) {
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     struct{}{},
		HTMLName: "transaction_create.html",
	})
}

func (s *Server) TransactionDetail(c *gin.Context) {
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     struct{}{},
		HTMLName: "transaction_detail.html",
	})
}

func (s *Server) UpdateTransaction(c *gin.Context) {
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     struct{}{},
		HTMLName: "transaction_update.html",
	})
}

func (s *Server) DeleteTransaction(c *gin.Context) {
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     struct{}{},
		HTMLName: "transaction_delete.html",
	})
}
