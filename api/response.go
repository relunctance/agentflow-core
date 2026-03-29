package api

import (
	"encoding/json"
	"net/http"
)

// Response is the standard JSON response wrapper
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo describes an error
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PaginatedResponse holds paginated list data
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	Total      int         `json:"total"`
	TotalPages int         `json:"total_pages"`
}

// WriteJSON writes a JSON response with the given status code
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteSuccess writes a success response with data
func WriteSuccess(w http.ResponseWriter, status int, data interface{}) {
	WriteJSON(w, status, Response{
		Success: true,
		Data:    data,
	})
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, status int, code string, message string) {
	WriteJSON(w, status, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// WritePaginated writes a paginated response
func WritePaginated(w http.ResponseWriter, data interface{}, page, pageSize, total int) {
	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}
	WriteJSON(w, http.StatusOK, PaginatedResponse{
		Success:    true,
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// Common error codes
const (
	ErrCodeBadRequest     = "BAD_REQUEST"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeInternalError  = "INTERNAL_ERROR"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeInvalidJSON    = "INVALID_JSON"
	ErrCodeMissingField   = "MISSING_FIELD"
	ErrCodeValidation     = "VALIDATION_ERROR"
)
