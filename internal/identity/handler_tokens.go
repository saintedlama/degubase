package identity

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
)

type TokenHandler struct{ Store Store }

// @Summary     List tokens
// @Tags        tokens
// @Produce     json
// @Param       wsCode  path      string  true  "workspace code"
// @Success     200     {array}   models.WorkspaceToken
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tokens [get]
func (h *TokenHandler) List(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	tokens, err := h.Store.ListWorkspaceTokens(r.Context(), ws.ID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if tokens == nil {
		tokens = []models.WorkspaceToken{}
	}
	httplib.Respond(w, http.StatusOK, tokens)
}

// @Summary     Create token
// @Tags        tokens
// @Accept      json
// @Produce     json
// @Param       wsCode  path      string              true  "workspace code"
// @Param       body    body      CreateTokenRequest  true  "token"
// @Success     201     {object}  object
// @Failure     400     {object}  http.ErrorResponse
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tokens [post]
func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)

	var body CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}

	token, hash, err := httplib.GenerateAPIToken()
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	prefix := token[:13]

	var createdBy *int64
	if u := httplib.UserFromCtx(r); u != nil {
		createdBy = &u.ID
	}

	t, err := h.Store.CreateWorkspaceToken(r.Context(), ws.ID, body.Name, hash, prefix, createdBy)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}

	httplib.Respond(w, http.StatusCreated, map[string]any{
		"id":           t.ID,
		"workspace_id": t.WorkspaceID,
		"name":         t.Name,
		"prefix":       t.Prefix,
		"created_at":   t.CreatedAt,
		"token":        token,
	})
}

// @Summary     Delete token
// @Tags        tokens
// @Param       wsCode   path  string  true  "workspace code"
// @Param       tokenID  path  int     true  "token ID"
// @Success     204
// @Failure     400  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tokens/{tokenID} [delete]
func (h *TokenHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "tokenID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid token id")
		return
	}
	if err := h.Store.DeleteWorkspaceToken(r.Context(), id); err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
