package skills

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
)

type SkillHandler struct {
	Store       Store
	Tbls        TableReader
	Rows        RowReader
	Ident       WorkspaceTokenCreator
	DisableAuth bool
}

// @Summary     List skills
// @Tags        skills
// @Produce     json
// @Param       wsCode  path      string  true  "workspace code"
// @Success     200     {array}   models.Skill
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/skills [get]
func (h *SkillHandler) List(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	skills, err := h.Store.ListSkills(r.Context(), ws.ID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if skills == nil {
		skills = []models.Skill{}
	}
	httplib.Respond(w, http.StatusOK, skills)
}

// @Summary     Create skill
// @Tags        skills
// @Accept      json
// @Produce     json
// @Param       wsCode  path      string              true  "workspace code"
// @Param       body    body      CreateSkillRequest  true  "skill"
// @Success     201     {object}  object
// @Failure     400     {object}  http.ErrorResponse
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/skills [post]
func (h *SkillHandler) Create(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)

	var body CreateSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httplib.BadRequest(w, "invalid request body")
		return
	}
	if body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}

	for _, tid := range body.TableIDs {
		t, err := h.Tbls.GetTable(r.Context(), tid)
		if err != nil {
			httplib.InternalErr(w, err)
			return
		}
		if t == nil || t.WorkspaceID != ws.ID {
			httplib.BadRequest(w, "invalid table_id")
			return
		}
	}

	var tokenID *int64
	var rawToken string
	if body.TokenName != "" {
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
		t, err := h.Ident.CreateWorkspaceToken(r.Context(), ws.ID, body.TokenName, hash, prefix, createdBy)
		if err != nil {
			httplib.InternalErr(w, err)
			return
		}
		tokenID = &t.ID
		rawToken = token
	}

	description := body.Description
	if description == "" {
		description = "Access " + ws.Name + " data in DeguBase"
	}

	sk, err := h.Store.CreateSkill(r.Context(), ws.ID, body.Name, description, body.TableIDs, body.Operations, tokenID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}

	resp := map[string]any{"skill": sk}
	if rawToken != "" {
		resp["token"] = rawToken
	}
	httplib.Respond(w, http.StatusCreated, resp)
}

// @Summary     Preview skill (without persisting)
// @Tags        skills
// @Accept      json
// @Produce     text/markdown
// @Param       wsCode  path      string               true  "workspace code"
// @Param       body    body      PreviewSkillRequest  true  "skill preview params"
// @Success     200
// @Failure     400  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/skills/preview [post]
func (h *SkillHandler) Preview(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	var body PreviewSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httplib.BadRequest(w, "invalid request body")
		return
	}
	name := body.Name
	if name == "" {
		name = ws.Name
	}
	in, err := h.buildInput(r, ws, body.TableIDs, name, body.Description, body.Operations)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(generateSkillContent(in)))
}

// @Summary     Get skill
// @Tags        skills
// @Produce     json
// @Param       wsCode   path      string  true  "workspace code"
// @Param       skillID  path      int     true  "skill ID"
// @Success     200      {object}  models.Skill
// @Failure     404      {object}  http.ErrorResponse
// @Failure     500      {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/skills/{skillID} [get]
func (h *SkillHandler) Get(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	sk, err := h.resolveSkill(w, r, ws.ID)
	if sk == nil || err != nil {
		return
	}
	httplib.Respond(w, http.StatusOK, sk)
}

// @Summary     Update skill
// @Tags        skills
// @Accept      json
// @Produce     json
// @Param       wsCode   path      string              true  "workspace code"
// @Param       skillID  path      int                 true  "skill ID"
// @Param       body     body      UpdateSkillRequest  true  "skill"
// @Success     200      {object}  models.Skill
// @Failure     400      {object}  http.ErrorResponse
// @Failure     404      {object}  http.ErrorResponse
// @Failure     500      {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/skills/{skillID} [patch]
func (h *SkillHandler) Update(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	sk, err := h.resolveSkill(w, r, ws.ID)
	if sk == nil || err != nil {
		return
	}
	var body UpdateSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httplib.BadRequest(w, "invalid request body")
		return
	}
	if body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}
	for _, tid := range body.TableIDs {
		t, err := h.Tbls.GetTable(r.Context(), tid)
		if err != nil {
			httplib.InternalErr(w, err)
			return
		}
		if t == nil || t.WorkspaceID != ws.ID {
			httplib.BadRequest(w, "invalid table_id")
			return
		}
	}
	updated, err := h.Store.UpdateSkill(r.Context(), sk.ID, body.Name, body.Description, body.TableIDs, body.Operations)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusOK, updated)
}

// @Summary     Delete skill
// @Tags        skills
// @Param       wsCode   path  string  true  "workspace code"
// @Param       skillID  path  int     true  "skill ID"
// @Success     204
// @Failure     404  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/skills/{skillID} [delete]
func (h *SkillHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	sk, err := h.resolveSkill(w, r, ws.ID)
	if sk == nil || err != nil {
		return
	}
	if err := h.Store.DeleteSkill(r.Context(), sk.ID); err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// buildInput fetches tables, columns, and sample rows for the given tableIDs
// and assembles a skillGenInput. It is shared by the live-content and preview paths.
func (h *SkillHandler) buildInput(r *http.Request, ws *models.Workspace, tableIDs []int64, name, description string, operations []string) (skillGenInput, error) {
	tables := make([]models.Table, 0, len(tableIDs))
	sampleRows := make(map[int64][]models.Row, len(tableIDs))

	for _, tid := range tableIDs {
		t, err := h.Tbls.GetTable(r.Context(), tid)
		if err != nil {
			return skillGenInput{}, err
		}
		if t == nil || t.WorkspaceID != ws.ID {
			continue
		}
		cols, err := h.Tbls.ListColumns(r.Context(), t.ID)
		if err != nil {
			return skillGenInput{}, err
		}
		t.Columns = cols
		tables = append(tables, *t)

		if h.Rows != nil {
			rows, _, _ := h.Rows.ListRows(r.Context(), t.ID, 5, 0, nil, nil, cols, "")
			sampleRows[t.ID] = rows
		}
	}

	return skillGenInput{
		Workspace:   ws,
		Tables:      tables,
		SampleRows:  sampleRows,
		BaseURL:     requestBaseURL(r),
		SkillName:   name,
		Description: description,
		Operations:  operations,
		NoAuth:      h.DisableAuth,
	}, nil
}

func (h *SkillHandler) buildLiveContent(r *http.Request, ws *models.Workspace, sk *models.Skill) (string, error) {
	in, err := h.buildInput(r, ws, sk.TableIDs, sk.Name, sk.Description, sk.Operations)
	if err != nil {
		return "", err
	}
	return generateSkillContent(in), nil
}

func (h *SkillHandler) resolvePublicSkill(w http.ResponseWriter, r *http.Request) (*models.Workspace, *models.Skill) {
	ws, err := h.Ident.GetWorkspaceByCode(r.Context(), chi.URLParam(r, "wsCode"))
	if err != nil {
		httplib.InternalErr(w, err)
		return nil, nil
	}
	if ws == nil {
		httplib.NotFound(w)
		return nil, nil
	}
	sk, err := h.resolveSkill(w, r, ws.ID)
	if sk == nil || err != nil {
		return nil, nil
	}
	return ws, sk
}

// @Summary     Download skill as Markdown
// @Tags        skills
// @Produce     text/markdown
// @Param       wsCode   path  string  true  "workspace code"
// @Param       skillID  path  int     true  "skill ID"
// @Success     200
// @Failure     404  {object}  http.ErrorResponse
// @Router      /workspaces/{wsCode}/skills/{skillID}/skill.md [get]
func (h *SkillHandler) ServeSkillMd(w http.ResponseWriter, r *http.Request) {
	ws, sk := h.resolvePublicSkill(w, r)
	if ws == nil {
		return
	}
	content, err := h.buildLiveContent(r, ws, sk)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="skill.md"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

// @Summary     Skill context (LLM-ready)
// @Tags        skills
// @Produce     text/markdown
// @Param       wsCode   path  string  true  "workspace code"
// @Param       skillID  path  int     true  "skill ID"
// @Success     200
// @Failure     404  {object}  http.ErrorResponse
// @Router      /workspaces/{wsCode}/skills/{skillID}/context [get]
func (h *SkillHandler) ServeContext(w http.ResponseWriter, r *http.Request) {
	ws, sk := h.resolvePublicSkill(w, r)
	if ws == nil {
		return
	}
	content, err := h.buildLiveContent(r, ws, sk)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func (h *SkillHandler) resolveSkill(w http.ResponseWriter, r *http.Request, workspaceID int64) (*models.Skill, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "skillID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid skill id")
		return nil, err
	}
	sk, err := h.Store.GetSkill(r.Context(), id)
	if err != nil {
		httplib.InternalErr(w, err)
		return nil, err
	}
	if sk == nil || sk.WorkspaceID != workspaceID {
		httplib.NotFound(w)
		return nil, nil
	}
	return sk, nil
}

func requestBaseURL(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + r.Host
}
