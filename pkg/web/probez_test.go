package web_test

import "context"

func (w *webTestSuite) TestServerStatus() {
	w.Run("SuccessNoAuth", func() {
		//setup
		require := w.Require()
		ctx := context.Background()

		//test
		status, err := w.ClientNoAuth().Status(ctx)
		require.NoError(err, "client request error")
		require.NotNil(status, "expected a non-nil response object")
		require.Equalf("ok", status.Status, "expected an 'ok' status, got %s", status.Status)
	})

	w.Run("SuccessAuthNoPermissions", func() {
		//setup
		require := w.Require()
		ctx := context.Background()
		permissions := []string{}

		//test
		status, err := w.ClientWithPermissions(permissions).Status(ctx)
		require.NoError(err, "client request error")
		require.NotNil(status, "expected a non-nil response object")
		require.Equalf("ok", status.Status, "expected an 'ok' status, got %s", status.Status)
	})

	w.Run("SuccessAuthAllPermissions", func() {
		//setup
		require := w.Require()
		ctx := context.Background()
		permissions := AllPermissions

		//test
		status, err := w.ClientWithPermissions(permissions).Status(ctx)
		require.NoError(err, "client request error")
		require.NotNil(status, "expected a non-nil response object")
		require.Equalf("ok", status.Status, "expected an 'ok' status, got %s", status.Status)
	})
}

func (w *webTestSuite) TestServerDBInfo() {
	w.Run("SuccessAuthTailoredPermissions", func() {
		//setup
		require := w.Require()
		ctx := context.Background()
		permissions := []string{"config:view"}

		//test
		dbinfo, err := w.ClientWithPermissions(permissions).DBInfo(ctx)
		require.Error(err, "expected a client request error")
		require.ErrorContains(err, "store does not implement stats", "mock store shouldn't implement stats")
		require.Nil(dbinfo, "expected a nil response object")
	})

	w.Run("SuccessAuthAllPermissions", func() {
		//setup
		require := w.Require()
		ctx := context.Background()
		permissions := AllPermissions

		//test
		dbinfo, err := w.ClientWithPermissions(permissions).DBInfo(ctx)
		require.Error(err, "expected a client request error")
		require.ErrorContains(err, "store does not implement stats", "mock store shouldn't implement stats")
		require.Nil(dbinfo, "expected a nil response object")
	})

	w.Run("FailureAuthNoPermissions", func() {
		//setup
		require := w.Require()
		ctx := context.Background()
		permissions := []string{}

		//test
		dbinfo, err := w.ClientWithPermissions(permissions).DBInfo(ctx)
		require.Error(err, "expected a client request error")
		require.ErrorContains(err, "user does not have permission to perform this operation", "the user should not be authorized")
		require.Nil(dbinfo, "expected a nil response object")
	})

	w.Run("FailureNoAuth", func() {
		//setup
		require := w.Require()
		ctx := context.Background()

		//test
		dbinfo, err := w.ClientNoAuth().DBInfo(ctx)
		require.Error(err, "expected a client request error")
		require.ErrorContains(err, "this endpoint requires authentication", "the user should not be authenticated")
		require.Nil(dbinfo, "expected a nil response object")
	})
}
