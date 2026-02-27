package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/albievan/clarity/clarity-api/internal/apierr"
)

// Envelope is the root response wrapper for all list endpoints.
type Envelope[T any] struct {
	Data       T               `json:"data"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

// PaginationMeta carries page info in list responses.
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ErrorBody matches the OpenAPI error envelope schema.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Details []apierr.FieldError `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("response encode error", "error", err)
	}
}

func OK(w http.ResponseWriter, v any) { JSON(w, http.StatusOK, v) }

func Created(w http.ResponseWriter, v any) { JSON(w, http.StatusCreated, v) }

func Accepted(w http.ResponseWriter, v any) { JSON(w, http.StatusAccepted, v) }

func NoContent(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, err error) {
	if apiErr, ok := apierr.As(err); ok {
		body := ErrorBody{Error: ErrorDetail{
			Code:    apiErr.Code,
			Message: apiErr.Message,
			Details: apiErr.Details,
		}}
		JSON(w, apiErr.HTTPStatus, body)
		return
	}
	body := ErrorBody{Error: ErrorDetail{
		Code:    apierr.CodeInternalServerError,
		Message: "an unexpected error occurred",
	}}
	JSON(w, http.StatusInternalServerError, body)
}

func PageOf[T any](w http.ResponseWriter, items T, page, perPage, total int) {
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	JSON(w, http.StatusOK, Envelope[T]{
		Data: items,
		Pagination: &PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}
