package pagination

import (
	"net/http"
	"strconv"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 25
	MaxPerPage     = 100
)

type Params struct {
	Page    int
	PerPage int
	Offset  int
}

func Parse(r *http.Request) Params {
	page := DefaultPage
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}

	perPage := DefaultPerPage
	if v := r.URL.Query().Get("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			perPage = n
		}
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return Params{
		Page:    page,
		PerPage: perPage,
		Offset:  (page - 1) * perPage,
	}
}
