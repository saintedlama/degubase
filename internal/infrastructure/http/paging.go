package http

import (
	"net/http"
	"strconv"
)

const (
	DefaultPageSize = 200
	MaxPageSize     = 200
)

type PagingParameters struct {
	Page     int
	PageSize int
}

func ParsePaging(r *http.Request) PagingParameters {
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}

	return PagingParameters{Page: page, PageSize: pageSize}
}

func (p PagingParameters) Offset() int {
	return (p.Page - 1) * p.PageSize
}
