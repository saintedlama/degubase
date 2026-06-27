package automation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
)

type ScriptHandler struct{ Store Store }

// @Summary     List scripts
// @Tags        scripts
// @Produce     json
// @Param       wsCode  path      string  true  "workspace code"
// @Success     200     {array}   models.Script
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/scripts [get]
func (h *ScriptHandler) List(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	scripts, err := h.Store.ListScripts(r.Context(), ws.ID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if scripts == nil {
		scripts = []models.Script{}
	}
	httplib.Respond(w, http.StatusOK, scripts)
}

// @Summary     Create script
// @Tags        scripts
// @Accept      json
// @Produce     json
// @Param       wsCode  path      string               true  "workspace code"
// @Param       body    body      CreateScriptRequest  true  "script"
// @Success     201     {object}  models.Script
// @Failure     400     {object}  http.ErrorResponse
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/scripts [post]
func (h *ScriptHandler) Create(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	var body CreateScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}
	if body.EventType == "" {
		httplib.BadRequest(w, "event_type is required")
		return
	}
	if body.Code == "" {
		httplib.BadRequest(w, "code is required")
		return
	}
	sc, err := h.Store.CreateScript(r.Context(), ws.ID, body.TableIDs, body.Name, body.EventType, body.Code, body.Enabled)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusCreated, sc)
}

// @Summary     Get script
// @Tags        scripts
// @Produce     json
// @Param       wsCode    path      string  true  "workspace code"
// @Param       scriptID  path      int     true  "script ID"
// @Success     200       {object}  models.Script
// @Failure     404       {object}  http.ErrorResponse
// @Failure     500       {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/scripts/{scriptID} [get]
func (h *ScriptHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "scriptID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid script id")
		return
	}
	sc, err := h.Store.GetScript(r.Context(), id)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if sc == nil {
		httplib.NotFound(w)
		return
	}
	httplib.Respond(w, http.StatusOK, sc)
}

// @Summary     Update script
// @Tags        scripts
// @Accept      json
// @Produce     json
// @Param       wsCode    path      string               true  "workspace code"
// @Param       scriptID  path      int                  true  "script ID"
// @Param       body      body      CreateScriptRequest  true  "script"
// @Success     200       {object}  models.Script
// @Failure     400       {object}  http.ErrorResponse
// @Failure     500       {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/scripts/{scriptID} [put]
func (h *ScriptHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "scriptID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid script id")
		return
	}
	var body CreateScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}
	sc, err := h.Store.UpdateScript(r.Context(), id, body.TableIDs, body.Name, body.EventType, body.Code, body.Enabled)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusOK, sc)
}

// @Summary     Delete script
// @Tags        scripts
// @Param       wsCode    path  string  true  "workspace code"
// @Param       scriptID  path  int     true  "script ID"
// @Success     204
// @Failure     400  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/scripts/{scriptID} [delete]
func (h *ScriptHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "scriptID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid script id")
		return
	}
	if err := h.Store.DeleteScript(r.Context(), id); err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary     List script env vars
// @Tags        scripts
// @Produce     json
// @Param       wsCode  path      string  true  "workspace code"
// @Success     200     {array}   models.ScriptEnvVar
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/scripts/env [get]
func (h *ScriptHandler) ListEnv(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	vars, err := h.Store.ListScriptEnvVars(r.Context(), ws.ID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if vars == nil {
		vars = []models.ScriptEnvVar{}
	}
	httplib.Respond(w, http.StatusOK, vars)
}

// @Summary     Upsert script env var
// @Tags        scripts
// @Accept      json
// @Produce     json
// @Param       wsCode  path      string           true  "workspace code"
// @Param       body    body      UpsertEnvRequest true  "env var"
// @Success     200     {object}  models.ScriptEnvVar
// @Failure     400     {object}  http.ErrorResponse
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/scripts/env [post]
func (h *ScriptHandler) UpsertEnv(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	var body UpsertEnvRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
		httplib.BadRequest(w, "key is required")
		return
	}
	ev, err := h.Store.UpsertScriptEnvVar(r.Context(), ws.ID, body.Key, body.Value, body.IsSecret)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusOK, ev)
}

// @Summary     Delete script env var
// @Tags        scripts
// @Param       wsCode  path  string  true  "workspace code"
// @Param       envID   path  int     true  "env var ID"
// @Success     204
// @Failure     400  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/scripts/env/{envID} [delete]
func (h *ScriptHandler) DeleteEnv(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "envID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid env var id")
		return
	}
	if err := h.Store.DeleteScriptEnvVar(r.Context(), id); err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary     List script executions
// @Tags        scripts
// @Produce     json
// @Param       wsCode    path      string  true   "workspace code"
// @Param       scriptID  path      int     true   "script ID"
// @Param       limit     query     int     false  "max results"
// @Success     200       {array}   models.ScriptExecution
// @Failure     400       {object}  http.ErrorResponse
// @Failure     500       {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/scripts/{scriptID}/executions [get]
func (h *ScriptHandler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "scriptID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid script id")
		return
	}
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	execs, err := h.Store.ListScriptExecutions(r.Context(), id, limit)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if execs == nil {
		execs = []models.ScriptExecution{}
	}
	httplib.Respond(w, http.StatusOK, execs)
}

// @Summary     Record script execution
// @Tags        scripts
// @Accept      json
// @Produce     json
// @Param       wsCode    path      string                 true  "workspace code"
// @Param       scriptID  path      int                    true  "script ID"
// @Param       body      body      models.ScriptExecution true  "execution result"
// @Success     201       {object}  models.ScriptExecution
// @Failure     400       {object}  http.ErrorResponse
// @Failure     500       {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/scripts/{scriptID}/executions [post]
func (h *ScriptHandler) CreateExecution(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "scriptID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid script id")
		return
	}
	var body models.ScriptExecution
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httplib.BadRequest(w, "invalid request body")
		return
	}
	body.ScriptID = id
	ex, err := h.Store.CreateScriptExecution(r.Context(), body)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusCreated, ex)
}
