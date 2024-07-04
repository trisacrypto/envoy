package web

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/oklog/ulid/v2"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth"
)

func (s *Server) ListUsers(c *gin.Context) {
	var (
		err   error
		in    *api.PageQuery
		query *models.PageInfo
		page  *models.UserPage
		out   *api.UserList
	)

	// Parse the URL parameters from the input request
	in = &api.PageQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse page query request"))
		return
	}

	// TODO: implement better pagination mechanism (with pagination tokens)

	// Fetch the list of users from the database
	if page, err = s.store.ListUsers(c.Request.Context(), query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process user list request"))
		return
	}

	// Convert the users page into a users list object
	if out, err = api.NewUserList(page); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process user list request"))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "user_list.html",
	})
}

func (s *Server) CreateUser(c *gin.Context) {
	var (
		err      error
		in       *api.User
		user     *models.User
		role     *models.Role
		password string
		out      *api.User
	)

	// Parse the model from the POST request
	in = &api.User{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse user data"))
		return
	}

	if !ulids.IsZero(in.ID) {
		c.JSON(http.StatusBadRequest, api.Error("cannot specify an id when creating a user"))
		return
	}

	if err = in.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Validate the role in the database
	if role, err = s.store.LookupRole(c.Request.Context(), in.Role); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusBadRequest, api.Error(api.ValidationError(nil, api.IncorrectField("role", "unknown role - specify one of admin, compliance, or observer"))))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create user request"))
		return
	}

	// Convert the API serializer into a database model
	if user, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Set the role on the user
	user.SetRole(role)

	// Create a password for the user -- the user cannot specify one themselves, but
	// the password will be returned to the user after the API call.
	password = auth.AlphaNumeric(12)
	if user.Password, err = auth.CreateDerivedKey(password); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete create user request"))
		return
	}

	// Create the model in the database (which will update the pointer)
	if err = s.store.CreateUser(c.Request.Context(), user); err != nil {
		// TODO: handle other error types that would return a 400
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewUser(user); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create user request"))
		return
	}

	// Ensure the created password is returned back to the user
	out.Passsword = password

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "user_create.html",
	})
}

func (s *Server) UserDetail(c *gin.Context) {
	var (
		err    error
		userID ulid.ULID
		user   *models.User
		out    *api.User
	)

	// Parse the userID from the URL
	if userID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("user not found"))
		return
	}

	// Fetch the model from the database
	if user, err = s.store.RetrieveUser(c.Request.Context(), userID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("user not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to fetch specified user"))
		return
	}

	if out, err = api.NewUser(user); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to fetch specified user"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "user_detail.html",
	})
}

func (s *Server) UpdateUser(c *gin.Context) {
	var (
		err    error
		userID ulid.ULID
		user   *models.User
		role   *models.Role
		in     *api.User
		out    *api.User
	)

	// Parse the userID from the URL
	if userID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("user not found"))
		return
	}

	// Parse the user data for the update request
	in = &api.User{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse user data"))
		return
	}

	// Sanity check
	if err = ulids.CheckIDMatch(in.ID, userID); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Validation
	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Extract the model
	if user, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Validate the role in the database
	if role, err = s.store.LookupRole(c.Request.Context(), in.Role); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusBadRequest, api.Error(api.ValidationError(nil, api.IncorrectField("role", "unknown role - specify one of admin, compliance, or observer"))))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create user request"))
		return
	}

	// Set the role on the user for update
	user.SetRole(role)

	if err = s.store.UpdateUser(c.Request.Context(), user); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("user not found"))
			return
		}

		// TODO: handle other types of database errors and constraints
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert model back to an API response
	if out, err = api.NewUser(user); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "user_update.html",
	})
}

func (s *Server) DeleteUser(c *gin.Context) {
	var (
		err    error
		userID ulid.ULID
	)

	// Parse the userID from the URL
	if userID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("user not found"))
		return
	}

	// Delete the user from the database
	if err = s.store.DeleteUser(c.Request.Context(), userID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("user not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLData: gin.H{"UserID": userID},
		JSONData: api.Reply{Success: true},
		HTMLName: "user_delete.html",
	})
}

func (s *Server) ChangeUserPassword(c *gin.Context) {
	var (
		err        error
		userID     ulid.ULID
		in         *api.UserPassword
		derivedKey string
	)

	// Parse the userID from the URL
	if userID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("user not found"))
		return
	}

	// Parse the user data for the update request
	in = &api.UserPassword{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse password change request"))
		return
	}

	// Validation
	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Create derived key from requested password reset
	if derivedKey, err = auth.CreateDerivedKey(in.Password); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete change user password request"))
		return
	}

	// Set the password for the specified user
	if err = s.store.SetUserPassword(c.Request.Context(), userID, derivedKey); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("user not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete change user password request"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLData: gin.H{"UserID": userID},
		JSONData: api.Reply{Success: true},
		HTMLName: "user_password_changed.html",
	})
}
