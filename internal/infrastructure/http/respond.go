package http

import (
	"encoding/json"
	nethttp "net/http"
)

// ErrorResponse is the standard error envelope returned by all API error responses.
type ErrorResponse struct {
	Error string `json:"error" example:"not found"`
	Code  string `json:"code"  example:"not_found"`
}

func Respond(w nethttp.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func RespondErr(w nethttp.ResponseWriter, status int, code, msg string) {
	Respond(w, status, ErrorResponse{Error: msg, Code: code})
}

func NotFound(w nethttp.ResponseWriter) {
	RespondErr(w, nethttp.StatusNotFound, "not_found", "not found")
}

func BadRequest(w nethttp.ResponseWriter, msg string) {
	RespondErr(w, nethttp.StatusBadRequest, "bad_request", msg)
}

func InternalErr(w nethttp.ResponseWriter, err error) {
	RespondErr(w, nethttp.StatusInternalServerError, "internal_error", err.Error())
}

func Forbidden(w nethttp.ResponseWriter) {
	RespondErr(w, nethttp.StatusForbidden, "forbidden", "forbidden")
}

func Unauthorized(w nethttp.ResponseWriter) {
	RespondErr(w, nethttp.StatusUnauthorized, "unauthorized", "authentication required")
}
