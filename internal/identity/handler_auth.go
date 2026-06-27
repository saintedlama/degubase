package identity

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	Store       Store
	JWTSecret   []byte
	DisableAuth bool
	DefaultUser *models.User
}

// @Summary     Auth setup status
// @Tags        auth
// @Produce     json
// @Success     200  {object}  map[string]bool
// @Router      /auth/setup [get]
func (h *AuthHandler) SetupStatus(w http.ResponseWriter, r *http.Request) {
	if h.DisableAuth {
		httplib.Respond(w, http.StatusOK, map[string]any{"setup_required": false, "auth_disabled": true})
		return
	}
	n, err := h.Store.CountUsers(r.Context())
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusOK, map[string]bool{"setup_required": n == 0})
}

// @Summary     Login
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body  body      LoginRequest  true  "credentials"
// @Success     200   {object}  models.User
// @Failure     400   {object}  http.ErrorResponse
// @Failure     401   {object}  http.ErrorResponse
// @Router      /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.DisableAuth {
		httplib.RespondErr(w, http.StatusNotFound, "auth_disabled", "authentication is disabled")
		return
	}
	var body LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httplib.BadRequest(w, "invalid request body")
		return
	}
	if body.Username == "" || body.Password == "" {
		httplib.BadRequest(w, "username and password are required")
		return
	}

	n, err := h.Store.CountUsers(r.Context())
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}

	var user *models.User

	if n == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			httplib.InternalErr(w, err)
			return
		}
		user, err = h.Store.CreateUser(r.Context(), body.Username, body.Username, string(hash), true)
		if err != nil {
			httplib.InternalErr(w, err)
			return
		}
	} else {
		hash, err := h.Store.GetUserPasswordHashByUsername(r.Context(), body.Username)
		if err != nil || hash == "" {
			httplib.Unauthorized(w)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)); err != nil {
			httplib.Unauthorized(w)
			return
		}
		user, err = h.Store.GetUserByUsername(r.Context(), body.Username)
		if err != nil || user == nil {
			httplib.Unauthorized(w)
			return
		}
	}

	signed, err := httplib.SignJWT(h.JWTSecret, user.ID, user.IsAdmin)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.SetAuthCookie(w, signed)
	httplib.Respond(w, http.StatusOK, user)
}

// @Summary     Logout
// @Tags        auth
// @Success     204
// @Router      /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if h.DisableAuth {
		httplib.RespondErr(w, http.StatusNotFound, "auth_disabled", "authentication is disabled")
		return
	}
	httplib.ClearAuthCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// @Summary     Current user
// @Tags        auth
// @Produce     json
// @Success     200  {object}  models.User
// @Failure     401  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	u := httplib.UserFromCtx(r)
	if u == nil {
		if h.DisableAuth && h.DefaultUser != nil {
			httplib.Respond(w, http.StatusOK, h.DefaultUser)
			return
		}
		httplib.Unauthorized(w)
		return
	}
	user, _ := h.Store.GetUserByID(r.Context(), u.ID)
	if user != nil {
		httplib.Respond(w, http.StatusOK, user)
		return
	}
	httplib.Unauthorized(w)
}

// @Summary     List users
// @Tags        admin
// @Produce     json
// @Success     200  {array}   models.User
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /admin/users [get]
func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if h.DisableAuth {
		httplib.NotFound(w)
		return
	}
	users, err := h.Store.ListUsers(r.Context())
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if users == nil {
		users = []models.User{}
	}
	httplib.Respond(w, http.StatusOK, users)
}

// @Summary     Create user
// @Tags        admin
// @Accept      json
// @Produce     json
// @Param       body  body      CreateUserRequest  true  "user"
// @Success     201   {object}  models.User
// @Failure     400   {object}  http.ErrorResponse
// @Failure     500   {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /admin/users [post]
func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if h.DisableAuth {
		httplib.NotFound(w)
		return
	}
	var body CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" || body.Password == "" {
		httplib.BadRequest(w, "username and password are required")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	u, err := h.Store.CreateUser(r.Context(), body.Username, body.Name, string(hash), body.IsAdmin)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusCreated, u)
}

// @Summary     Delete user
// @Tags        admin
// @Param       userID  path  int  true  "user ID"
// @Success     204
// @Failure     400  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /admin/users/{userID} [delete]
func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if h.DisableAuth {
		httplib.NotFound(w)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid user id")
		return
	}
	if err := h.Store.DeleteUser(r.Context(), id); err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
