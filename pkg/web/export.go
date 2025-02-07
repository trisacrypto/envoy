package web

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

const (
	ContentDisposition = "Content-Disposition"
	ContentType        = "Content-Type"
	AcceptLength       = "Accept-Length"
	ContentTypeCSV     = "text/csv"
)

var TransactionsHeader = []string{
	"ID", "Status", "Counterparty", "Originator", "Originator Address",
	"Beneficiary", "Beneficiary Address", "Virtual Asset", "Amount",
	"Last Update", "Created", "Number of Envelopes", "HMAC Signature",
}

func (s *Server) ExportTransactions(c *gin.Context) {
	var (
		err  error
		page *models.TransactionPage
		info *models.PageInfo
	)

	// Create the filename for export based on the current date
	filename := fmt.Sprintf("transactions-%s.csv", time.Now().Format("2006-01-02"))

	// Prepare the header for writing
	c.Header(ContentDisposition, "attachment; filename="+filename)
	c.Header(ContentType, ContentTypeCSV)
	c.Writer.WriteHeader(http.StatusOK)

	// Stream the csv to the specified endpoint
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write the header to the CSV file
	if err = writer.Write(TransactionsHeader); err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("error writing header to download stream: %w", err))
		return
	}

	// Fetch 50 records per page from the database
	info = &models.PageInfo{PageSize: 50, NextPageID: ulid.Null}

	// TODO: we'll probably want to load more information from the secure envelope
	// besides what's in the transaction and if we do that, we'll want a transaction
	// iterator with a link to the database rather than loading them a page at a time.
transactionsIterator:
	for {
		if page, err = s.store.ListTransactions(c.Request.Context(), info); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		for i, transaction := range page.Transactions {
			record := []string{
				transaction.ID.String(),
				transaction.Status,
				transaction.Counterparty,
				transaction.Originator.String,
				transaction.OriginatorAddress.String,
				transaction.Beneficiary.String,
				transaction.BeneficiaryAddress.String,
				transaction.VirtualAsset,
				strconv.FormatFloat(transaction.Amount, 'f', -1, 64),
				transaction.LastUpdate.Time.Format(time.RFC3339),
				transaction.Created.Format(time.RFC3339),
				strconv.FormatInt(transaction.NumEnvelopes(), 10),
				"",
			}

			if err = writer.Write(record); err != nil {
				c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("error writing row %d to download stream: %w", i+1, err))
				return
			}
		}

		// If there is no next page ID, then stop iterating
		info.NextPageID = page.Page.NextPageID
		if info.NextPageID.IsZero() {
			break transactionsIterator
		}
	}
}
