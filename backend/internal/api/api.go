package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"ws-json-rpc/backend/pkg/apitypes"
	"ws-json-rpc/backend/pkg/router"
	"ws-json-rpc/backend/pkg/utils"

	"github.com/google/uuid"
)

type Server struct {
	l  *slog.Logger
	db *sql.DB
}

func NewAPIServer(l *slog.Logger, db *sql.DB) *Server {
	return &Server{
		l:  l.With(slog.String("component", "http-api")),
		db: db,
	}
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

const (
	MaxBodySize     = 1048576 // 1MB
	RequestIDHeader = "X-Request-ID"
)

func NewHTTPError(statusCode int, message string) *apitypes.HTTPHandlerError {
	return &apitypes.HTTPHandlerError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// ErrorHandler wraps handlers with error handling
func ErrorHandler(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := GetLogger(r.Context())
		requestID := GetRequestID(r.Context())

		err := fn(w, r)
		if err == nil {
			return
		}

		// This is an expected HTTP error, we return the actual error to the client
		if httpErr, ok := err.(*apitypes.HTTPHandlerError); ok {
			httpErr.RequestID = requestID
			l.Warn("handler returned HTTP error", slog.Int("status", httpErr.StatusCode), slog.String("message", httpErr.Message))
			RespondJSON(w, r, httpErr.StatusCode, httpErr)
			return
		}

		// Internal errors get logged with full context, but we return a generic message to the client
		l.Error("internal error", utils.ErrAttr(err))
		RespondJSON(w, r, http.StatusInternalServerError, apitypes.ServerErrorResponse{
			RequestID: requestID,
			Message:   "Internal server error",
		})
	}
}

// RespondJSON sends a JSON response with given status code
func RespondJSON(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data == nil {
		return
	}

	l := GetLogger(r.Context())
	if err := utils.ToJSONStream(w, data); err != nil {
		l.Error("failed to encode JSON response", utils.ErrAttr(err))
	}
}

// DecodeJSON decodes JSON from request body with error handling
func DecodeJSON[T any](r *http.Request) (T, error) {
	var zero T

	r.Body = http.MaxBytesReader(nil, r.Body, MaxBodySize)

	res, err := utils.FromJSONStream[T](r.Body)
	if err != nil {
		// FIXME: on Go 1.26 use errors.AsType[...]()
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError
		var extraDataError *utils.ExtraDataAfterJSONError

		switch {
		case errors.As(err, &syntaxError):
			return zero, NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid JSON syntax at position %d", syntaxError.Offset))

		case errors.As(err, &unmarshalTypeError):
			return zero, NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid type for field '%s'", unmarshalTypeError.Field))

		case errors.Is(err, io.EOF):
			return zero, NewHTTPError(http.StatusBadRequest, "Request body is empty")

		case errors.Is(err, io.ErrUnexpectedEOF):
			return zero, NewHTTPError(http.StatusBadRequest, "Malformed JSON")

		case errors.As(err, &maxBytesError):
			return zero, NewHTTPError(http.StatusRequestEntityTooLarge, fmt.Sprintf("Request body too large (max %dMB)", MaxBodySize/(1024*1024)))

		case errors.As(err, &extraDataError):
			return zero, NewHTTPError(http.StatusBadRequest, "Request body contains multiple JSON objects")

		case strings.Contains(err.Error(), "json: unknown field"):
			// json package formats this as: json: unknown field "fieldname"
			return zero, NewHTTPError(http.StatusBadRequest, err.Error())

		default:
			return zero, NewHTTPError(http.StatusBadRequest, "Invalid JSON payload")
		}
	}

	return res, nil
}

var zeroUUID = uuid.MustParse("00000000-0000-0000-0000-000000000000").String()

// MakeResponses adds standard error responses to the given responses map
func MakeResponses(responses map[int]router.ResponseSpec) map[int]router.ResponseSpec {
	responses[http.StatusRequestEntityTooLarge] = router.ResponseSpec{
		Description: "Request entity too large",
		Type:        apitypes.ServerErrorResponse{},
		Examples: map[string]any{
			"Request Entity Too Large": apitypes.ServerErrorResponse{
				RequestID: zeroUUID,
				Message:   "Request Entity Too Large",
			},
		},
	}

	responses[http.StatusInternalServerError] = router.ResponseSpec{
		Description: "Internal server error",
		Type:        apitypes.ServerErrorResponse{},
		Examples: map[string]any{
			"Internal Server Error": apitypes.ServerErrorResponse{
				RequestID: zeroUUID,
				Message:   "Internal Server Error",
			},
		},
	}

	return responses
}
