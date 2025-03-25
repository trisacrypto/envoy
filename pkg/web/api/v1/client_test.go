package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/trisacrypto/envoy/pkg/web/api/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	ctx     = context.Background()
	page    = &api.TransactionListQuery{PageQuery: api.PageQuery{PageSize: 50}}
	success = map[string]interface{}{"success": true}
)

func TestStatus(t *testing.T) {
	t.Run("Available", func(t *testing.T) {
		fixture := &api.StatusReply{}
		err := loadFixture("testdata/status.json", fixture)
		require.NoError(t, err, "could not load status fixture")

		_, client := testServer(t, &testServerConfig{
			expectedMethod: http.MethodGet,
			expectedPath:   "/v1/status",
			fixture:        fixture,
			statusCode:     http.StatusOK,
		})

		rep, err := client.Status(ctx)
		require.NoError(t, err, "could not execute status request")
		require.Equal(t, fixture, rep, "expected reply to be equal to the fixture")
	})

	t.Run("Maintenance", func(t *testing.T) {
		fixture := &api.StatusReply{}
		err := loadFixture("testdata/status.json", fixture)
		require.NoError(t, err, "could not load status fixture")
		fixture.Status = "maintenance"

		_, client := testServer(t, &testServerConfig{
			expectedMethod: http.MethodGet,
			expectedPath:   "/v1/status",
			fixture:        fixture,
			statusCode:     http.StatusServiceUnavailable,
		})

		rep, err := client.Status(ctx)
		require.NoError(t, err, "could not execute status request")
		require.Equal(t, fixture, rep, "expected reply to be equal to the fixture")
	})

	t.Run("Internal", func(t *testing.T) {
		_, client := testServer(t, &testServerConfig{
			expectedMethod: http.MethodGet,
			expectedPath:   "/v1/status",
			fixture:        nil,
			statusCode:     http.StatusInternalServerError,
		})

		_, err := client.Status(ctx)
		CheckStatusError(t, err, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	})
}

func TestDBInfo(t *testing.T) {
	fixture := &api.DBInfo{}
	err := loadFixture("testdata/dbinfo.json", fixture)
	require.NoError(t, err, "could not load dbinfo fixture")

	_, client := testServer(t, &testServerConfig{
		expectedMethod: http.MethodGet,
		expectedPath:   "/v1/dbinfo",
		fixture:        fixture,
		statusCode:     http.StatusOK,
	})

	rep, err := client.DBInfo(ctx)
	require.NoError(t, err, "could not execute dbinfo request")
	require.Equal(t, fixture, rep, "expected reply to be equal to the fixture")
}

func TestListTransactions(t *testing.T) {
	fixture := &api.TransactionsList{}
	err := loadFixture("testdata/transaction_list.json", fixture)
	require.NoError(t, err, "could not load transaction list fixture")

	_, client := testServer(t, &testServerConfig{
		expectedMethod: http.MethodGet,
		expectedPath:   "/v1/transactions",
		fixture:        fixture,
		statusCode:     http.StatusOK,
	})

	rep, err := client.ListTransactions(ctx, page)
	require.NoError(t, err, "could not execute list transactions request")
	require.Equal(t, fixture, rep, "expected reply to be equal to the fixture")
}

func TestCreateTransaction(t *testing.T) {
	fixture := &api.Transaction{}
	err := loadFixture("testdata/transaction.json", fixture)
	require.NoError(t, err, "could not load transaction fixture")

	_, client := testServer(t, &testServerConfig{
		expectedMethod: http.MethodPost,
		expectedPath:   "/v1/transactions",
		fixture:        fixture,
		statusCode:     http.StatusCreated,
	})

	in := &api.Transaction{}
	rep, err := client.CreateTransaction(ctx, in)
	require.NoError(t, err, "could not execute create transaction request")
	require.Equal(t, fixture, rep, "expected reply to be equal to the fixture")
}

func TestTransactionDetail(t *testing.T) {
	fixture := &api.Transaction{}
	err := loadFixture("testdata/transaction.json", fixture)
	require.NoError(t, err, "could not load transaction fixture")

	_, client := testServer(t, &testServerConfig{
		expectedMethod: http.MethodGet,
		expectedPath:   "/v1/transactions/3b0ed85d-5eb4-406f-abca-57b199453343",
		fixture:        fixture,
		statusCode:     http.StatusOK,
	})

	rep, err := client.TransactionDetail(ctx, uuid.MustParse("3b0ed85d-5eb4-406f-abca-57b199453343"))
	require.NoError(t, err, "could not execute transaction detail request")
	require.Equal(t, fixture, rep, "expected reply to be equal to the fixture")
}

func TestUpdateTransaction(t *testing.T) {
	fixture := &api.Transaction{}
	err := loadFixture("testdata/transaction.json", fixture)
	require.NoError(t, err, "could not load transaction fixture")

	_, client := testServer(t, &testServerConfig{
		expectedMethod: http.MethodPut,
		expectedPath:   "/v1/transactions/3b0ed85d-5eb4-406f-abca-57b199453343",
		fixture:        fixture,
		statusCode:     http.StatusOK,
	})

	in := &api.Transaction{ID: uuid.MustParse("3b0ed85d-5eb4-406f-abca-57b199453343")}
	rep, err := client.UpdateTransaction(ctx, in)
	require.NoError(t, err, "could not execute update transaction request")
	require.Equal(t, fixture, rep, "expected reply to be equal to the fixture")
}

func TestDeleteTransaction(t *testing.T) {
	_, client := testServer(t, &testServerConfig{
		expectedMethod: http.MethodDelete,
		expectedPath:   "/v1/transactions/3b0ed85d-5eb4-406f-abca-57b199453343",
		fixture:        success,
		statusCode:     http.StatusOK,
	})

	err := client.DeleteTransaction(ctx, uuid.MustParse("3b0ed85d-5eb4-406f-abca-57b199453343"))
	require.NoError(t, err, "could not execute delete transaction request")
}

type testServerConfig struct {
	expectedMethod string
	expectedPath   string
	statusCode     int
	fixture        interface{}
}

func testServer(t *testing.T, conf *testServerConfig) (*httptest.Server, api.Client) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != conf.expectedMethod {
			http.Error(w, fmt.Sprintf("expected method %s got %s", conf.expectedMethod, r.Method), http.StatusExpectationFailed)
			return
		}

		if r.URL.Path != conf.expectedPath {
			http.Error(w, fmt.Sprintf("expected path %s got %s", conf.expectedPath, r.URL.Path), http.StatusExpectationFailed)
			return
		}

		if conf.statusCode == 0 {
			conf.statusCode = http.StatusOK
		}

		w.Header().Add("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(conf.statusCode)
		json.NewEncoder(w).Encode(conf.fixture)
	}))

	// Ensure the server is closed when the test is complete
	t.Cleanup(ts.Close)

	client, err := api.New(ts.URL)
	require.NoError(t, err, "could not create api client")
	return ts, client
}

func loadFixture(path string, v interface{}) (err error) {
	var f *os.File
	if f, err = os.Open(path); err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}

func CheckStatusError(t *testing.T, err error, code int, message string) {
	require.Error(t, err, "expected an error")

	serr, ok := err.(*api.StatusError)
	require.True(t, ok, "expected error to be a status error")

	require.Equal(t, code, serr.StatusCode, "unexpected status code")
	require.Equal(t, message, serr.Reply.Error, "unexpected status message")
}
