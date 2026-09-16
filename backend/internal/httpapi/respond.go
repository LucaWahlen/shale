package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"shale/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errorBody struct {
	Error errDetail `json:"error"`
}

type errDetail struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details []domain.Detail `json:"details,omitempty"`
}

func statusFor(k domain.Kind) int {
	switch k {
	case domain.KindInvalid:
		return http.StatusBadRequest
	case domain.KindNotFound:
		return http.StatusNotFound
	case domain.KindConflict:
		return http.StatusConflict
	case domain.KindUnauthorized:
		return http.StatusUnauthorized
	case domain.KindForbidden:
		return http.StatusForbidden
	case domain.KindNotAttendable, domain.KindUnprocessable:
		return http.StatusUnprocessableEntity
	case domain.KindRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, err error) {
	var derr *domain.Error
	if errors.As(err, &derr) {
		writeJSON(w, statusFor(derr.Kind), errorBody{errDetail{
			Code:    string(derr.Kind),
			Message: derr.Message,
			Details: derr.Details,
		}})
		return
	}
	writeJSON(w, http.StatusInternalServerError, errorBody{errDetail{
		Code:    "internal",
		Message: "internal server error",
	}})
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return domain.Errorf(domain.KindInvalid, "request body too large (max %d bytes)", maxErr.Limit)
		}
		return domain.NewError(domain.KindInvalid, "invalid JSON body")
	}
	return nil
}

func readRawBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, domain.Errorf(domain.KindInvalid, "request body too large (max %d bytes)", maxErr.Limit)
		}
		return nil, domain.NewError(domain.KindInvalid, "could not read request body")
	}
	if len(raw) == 0 {
		return nil, domain.NewError(domain.KindInvalid, "request body must not be empty")
	}
	return raw, nil
}

var errInternal = errors.New("internal server error")

var errNotFoundRoute = domain.NewError(domain.KindNotFound, "route not found")

func pathUUID(r *http.Request, name string) (string, error) {
	v := r.PathValue(name)
	if uuid.Validate(v) != nil {
		return "", domain.Errorf(domain.KindInvalid, "%s must be a valid UUID", name)
	}
	return v, nil
}
