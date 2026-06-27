package schema

import (
	"encoding/json"
	"errors"
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
)

type ColumnHandler struct{ Store Store }

var TypeFamilies = map[models.ColumnType]string{
	models.ColumnTypeText:         "text",
	models.ColumnTypeLongText:     "text",
	models.ColumnTypeMarkdown:     "text",
	models.ColumnTypeEmail:        "text",
	models.ColumnTypeURL:          "text",
	models.ColumnTypeNumber:       "number",
	models.ColumnTypeCurrency:     "number",
	models.ColumnTypePercent:      "number",
	models.ColumnTypeRating:       "number",
	models.ColumnTypeDate:         "date",
	models.ColumnTypeDatetime:     "date",
	models.ColumnTypeCheckbox:     "checkbox",
	models.ColumnTypeSingleSelect: "select",
	models.ColumnTypeMultiSelect:  "select",
	models.ColumnTypeFile:         "file",
	models.ColumnTypeImage:        "file",
	models.ColumnTypeCreatedAt:    "system",
	models.ColumnTypeUpdatedAt:    "system",
	models.ColumnTypeRowLink:      "link",
}

// @Summary     List columns
// @Tags        columns
// @Produce     json
// @Param       wsCode     path      string  true  "workspace code"
// @Param       tableCode  path      string  true  "table code"
// @Success     200        {array}   models.Column
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/columns [get]
func (h *ColumnHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	cols, err := h.Store.ListColumns(r.Context(), t.ID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if cols == nil {
		cols = []models.Column{}
	}
	httplib.Respond(w, nethttp.StatusOK, cols)
}

// @Summary     Create column
// @Tags        columns
// @Accept      json
// @Produce     json
// @Param       wsCode     path      string               true  "workspace code"
// @Param       tableCode  path      string               true  "table code"
// @Param       body       body      CreateColumnRequest  true  "column"
// @Success     201        {object}  models.Column
// @Failure     400        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/columns [post]
func (h *ColumnHandler) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	var body CreateColumnRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}
	if !body.Type.Valid() {
		httplib.BadRequest(w, "invalid column type")
		return
	}
	col, err := h.Store.CreateColumn(r.Context(), t.ID, body.Name, body.Type, body.Options)
	if err != nil {
		if errors.Is(err, ErrCodeConflict) {
			httplib.RespondErr(w, nethttp.StatusConflict, "code_conflict", err.Error())
			return
		}
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, nethttp.StatusCreated, col)
}

// @Summary     Update column
// @Tags        columns
// @Accept      json
// @Produce     json
// @Param       wsCode     path      string               true  "workspace code"
// @Param       tableCode  path      string               true  "table code"
// @Param       colCode    path      string               true  "column code"
// @Param       body       body      UpdateColumnRequest  true  "column"
// @Success     200        {object}  models.Column
// @Failure     400        {object}  http.ErrorResponse
// @Failure     404        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/columns/{colCode} [put]
func (h *ColumnHandler) Update(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	colCode := chi.URLParam(r, "colCode")
	var body CreateColumnRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}

	current, err := h.Store.GetColumnByCode(r.Context(), t.ID, colCode)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if current == nil {
		httplib.NotFound(w)
		return
	}

	newType := body.Type
	if newType == "" {
		newType = current.Type
	} else if !newType.Valid() {
		httplib.BadRequest(w, "invalid column type")
		return
	} else if newType != current.Type {
		currentFamily := TypeFamilies[current.Type]
		newFamily := TypeFamilies[newType]
		if currentFamily == "system" || currentFamily != newFamily {
			httplib.RespondErr(w, nethttp.StatusUnprocessableEntity, "type_incompatible",
				"type change not allowed between incompatible families")
			return
		}
	}

	col, err := h.Store.UpdateColumn(r.Context(), current.ID, body.Name, newType, body.Options)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, nethttp.StatusOK, col)
}

// @Summary     Reorder columns
// @Tags        columns
// @Accept      json
// @Param       wsCode     path  string                 true  "workspace code"
// @Param       tableCode  path  string                 true  "table code"
// @Param       body       body  ReorderColumnsRequest  true  "ordered IDs"
// @Success     204
// @Failure     400  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/columns/reorder [put]
func (h *ColumnHandler) Reorder(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	var body ReorderColumnsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Codes) == 0 {
		httplib.BadRequest(w, "codes is required")
		return
	}
	cols, err := h.Store.ListColumns(r.Context(), t.ID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	codeToID := make(map[string]int64, len(cols))
	for _, c := range cols {
		codeToID[c.Code] = c.ID
	}
	ids := make([]int64, 0, len(body.Codes))
	for _, code := range body.Codes {
		if id, ok := codeToID[code]; ok {
			ids = append(ids, id)
		}
	}
	if err := h.Store.ReorderColumns(r.Context(), t.ID, ids); err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

// @Summary     Delete column
// @Tags        columns
// @Param       wsCode     path  string  true  "workspace code"
// @Param       tableCode  path  string  true  "table code"
// @Param       colCode    path  string  true  "column code"
// @Success     204
// @Failure     404  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/columns/{colCode} [delete]
func (h *ColumnHandler) Delete(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	colCode := chi.URLParam(r, "colCode")
	current, err := h.Store.GetColumnByCode(r.Context(), t.ID, colCode)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if current == nil {
		httplib.NotFound(w)
		return
	}
	if err := h.Store.DeleteColumn(r.Context(), current.ID); err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}
