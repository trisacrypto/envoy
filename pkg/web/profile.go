package web

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	api "github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	"github.com/trisacrypto/envoy/pkg/web/scene"
	"go.rtnl.ai/ulid"
)

func (s *Server) ProfileDetail(c *gin.Context) {
	var (
		err  error
		user *models.User
		out  *api.User
	)

	// Retreive the user from the request context; note that this method will handle
	// any error responses if the retrieval fails in someway.
	if user, err = s.retrieveProfile(c); err != nil {
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
		HTMLName: "partials/profile/form.html",
	})
}

func (s *Server) UpdateProfile(c *gin.Context) {
	var (
		err  error
		user *models.User
		in   *api.User
		out  *api.User
	)

	in = &api.User{}
	if err = c.BindJSON(in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error("could not parse user profile"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Retreive the user from the request context; note that this method will handle
	// any error responses if the retrieval fails in someway.
	if user, err = s.retrieveProfile(c); err != nil {
		return
	}

	// The only thing the user can update with this form is their name and email address.
	user.Name = sql.NullString{String: in.Name, Valid: in.Name != ""}
	user.Email = in.Email

	if err = s.store.UpdateUser(c.Request.Context(), user); err != nil {
		// TODO: handle email unique constraint violation.
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to update user profile"))
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
		HTMLName: "partials/profile/form.html",
	})
}

func (s *Server) ChangeProfilePassword(c *gin.Context) {
	c.HTML(http.StatusOK, "profile/password.html", scene.New(c))
}

// Helper method to retrieve the profile being managed by the currently logged in user.
// If there is a failure in retrieving the user profile, this method handles it.
func (s *Server) retrieveProfile(c *gin.Context) (*models.User, error) {
	var (
		err     error
		claims  *auth.Claims
		subject auth.SubjectType
		userID  ulid.ULID
		user    *models.User
	)

	if claims, err = auth.GetClaims(c); err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, api.Error(auth.ErrNotAuthorized))
		return nil, err
	}

	if subject, userID, err = claims.SubjectID(); err != nil {
		c.Error(fmt.Errorf("could not parse subject ID from claims: %w", err))
		c.JSON(http.StatusBadRequest, api.Error("could not process profile request"))
		return nil, err
	}

	if subject != auth.SubjectUser {
		c.JSON(http.StatusNotFound, api.Error("profile not found"))
		return nil, dberr.ErrNotFound
	}

	// Fetch the model from the database
	if user, err = s.store.RetrieveUser(c.Request.Context(), userID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("user not found"))
			return nil, err
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to fetch specified user"))
		return nil, err
	}

	return user, nil
}
