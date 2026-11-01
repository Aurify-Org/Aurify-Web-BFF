package utils

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCError represents a parsed gRPC error with HTTP-friendly status code and message.
type GRPCError struct {
	HTTPStatus int
	Message    string
}

// grpcErrorRule defines a pattern-based rule to map gRPC error descriptions
// to a specific HTTP status code and user-facing message.
type grpcErrorRule struct {
	// pattern is the substring to search for (case-insensitive) in the gRPC error description.
	pattern string
	// httpStatus is the HTTP status code to return.
	httpStatus int
	// message is the user-facing message to return.
	message string
}

// msgServiceUnavailable is extracted to avoid duplicating the literal (go:S1192).
const msgServiceUnavailable = "Service is temporarily unavailable"

// grpcCodeToHTTP maps gRPC status codes to HTTP status codes.
var grpcCodeToHTTP = map[codes.Code]int{
	codes.OK:                 fiber.StatusOK,
	codes.Canceled:           fiber.StatusRequestTimeout,
	codes.InvalidArgument:    fiber.StatusBadRequest,
	codes.NotFound:           fiber.StatusNotFound,
	codes.AlreadyExists:      fiber.StatusConflict,
	codes.PermissionDenied:   fiber.StatusForbidden,
	codes.Unauthenticated:    fiber.StatusUnauthorized,
	codes.ResourceExhausted:  fiber.StatusTooManyRequests,
	codes.FailedPrecondition: fiber.StatusBadRequest,
	codes.Aborted:            fiber.StatusConflict,
	codes.OutOfRange:         fiber.StatusBadRequest,
	codes.Unimplemented:      fiber.StatusNotImplemented,
	codes.Internal:           fiber.StatusInternalServerError,
	codes.Unavailable:        fiber.StatusServiceUnavailable,
	codes.DataLoss:           fiber.StatusInternalServerError,
	codes.DeadlineExceeded:   fiber.StatusGatewayTimeout,
	codes.Unknown:            fiber.StatusInternalServerError,
}

// errorRules contains pattern-based rules ordered by specificity.
// More specific patterns should come first.
var errorRules = []grpcErrorRule{
	// ── Not Found errors ──
	{pattern: "transfer not found", httpStatus: fiber.StatusNotFound, message: "Transfer not found"},
	{pattern: "transaction not found", httpStatus: fiber.StatusNotFound, message: "Transaction not found"},
	{pattern: "wallet not found", httpStatus: fiber.StatusNotFound, message: "Wallet not found"},
	{pattern: "category not found", httpStatus: fiber.StatusNotFound, message: "Category not found"},
	{pattern: "attachment not found", httpStatus: fiber.StatusNotFound, message: "Attachment not found"},
	{pattern: "attachment with file", httpStatus: fiber.StatusNotFound, message: "Attachment not found"},
	{pattern: "investment not found", httpStatus: fiber.StatusNotFound, message: "Investment not found"},
	{pattern: "wallet type not found", httpStatus: fiber.StatusNotFound, message: "Wallet type not found"},
	{pattern: "source wallet not found", httpStatus: fiber.StatusNotFound, message: "Source wallet not found"},
	{pattern: "destination wallet not found", httpStatus: fiber.StatusNotFound, message: "Destination wallet not found"},
	{pattern: "new wallet not found", httpStatus: fiber.StatusNotFound, message: "Target wallet not found"},
	{pattern: "not found", httpStatus: fiber.StatusNotFound, message: "Resource not found"},

	// ── Insufficient balance / quantity ──
	{pattern: "insufficient wallet balance", httpStatus: fiber.StatusBadRequest, message: "Insufficient wallet balance"},
	{pattern: "insufficient investment quantity", httpStatus: fiber.StatusBadRequest, message: "Insufficient investment quantity"},
	{pattern: "insufficient", httpStatus: fiber.StatusBadRequest, message: "Insufficient balance"},

	// ── Transfers ──
	{pattern: "transfer leg cannot be edited on its own", httpStatus: fiber.StatusBadRequest, message: "A transfer is edited as a whole, not one side at a time"},
	{pattern: "invalid transfer amount", httpStatus: fiber.StatusBadRequest, message: "Transfer amount must be greater than 0 and the admin fee cannot be negative"},

	// ── Validation errors ──
	{pattern: "invalid date format", httpStatus: fiber.StatusBadRequest, message: "Invalid date format"},
	{pattern: "invalid category id", httpStatus: fiber.StatusBadRequest, message: "Invalid category ID"},
	{pattern: "invalid wallet id", httpStatus: fiber.StatusBadRequest, message: "Invalid wallet ID"},
	{pattern: "invalid user id", httpStatus: fiber.StatusBadRequest, message: "Invalid user ID"},
	{pattern: "invalid wallet type id", httpStatus: fiber.StatusBadRequest, message: "Invalid wallet type ID"},
	{pattern: "invalid transaction type", httpStatus: fiber.StatusBadRequest, message: "Invalid transaction type"},
	{pattern: "invalid from wallet id", httpStatus: fiber.StatusBadRequest, message: "Invalid source wallet ID"},
	{pattern: "invalid to wallet id", httpStatus: fiber.StatusBadRequest, message: "Invalid destination wallet ID"},
	{pattern: "invalid from category id", httpStatus: fiber.StatusBadRequest, message: "Invalid source category ID"},
	{pattern: "invalid to category id", httpStatus: fiber.StatusBadRequest, message: "Invalid destination category ID"},
	{pattern: "invalid attachment status", httpStatus: fiber.StatusBadRequest, message: "Invalid attachment action status"},
	{pattern: "invalid", httpStatus: fiber.StatusBadRequest, message: "Invalid request"},

	// ── Business logic errors ──
	{pattern: "wallet balance must be zero before deletion", httpStatus: fiber.StatusBadRequest, message: "Wallet balance must be zero before it can be deleted"},
	{pattern: "source wallet and destination wallet cannot be the same", httpStatus: fiber.StatusBadRequest, message: "Source and destination wallets must be different"},
	{pattern: "does not belong to transaction", httpStatus: fiber.StatusBadRequest, message: "Attachment does not belong to this transaction"},
	{pattern: "no files to upload", httpStatus: fiber.StatusBadRequest, message: "No files provided for upload"},
	{pattern: "no files to delete", httpStatus: fiber.StatusBadRequest, message: "No files provided for deletion"},
	{pattern: "file is empty", httpStatus: fiber.StatusBadRequest, message: "Uploaded file is empty"},
	{pattern: "transaction ID is required", httpStatus: fiber.StatusBadRequest, message: "Transaction ID is required"},

	// ── Upload / storage errors ──
	{pattern: "upload file", httpStatus: fiber.StatusBadRequest, message: "Failed to upload attachment"},

	// ── Timeout ──
	{pattern: "context deadline exceeded", httpStatus: fiber.StatusGatewayTimeout, message: "Request timed out. Please try again"},
	{pattern: "context canceled", httpStatus: fiber.StatusRequestTimeout, message: "Request was cancelled"},

	// ── Connectivity ──
	{pattern: "connection refused", httpStatus: fiber.StatusServiceUnavailable, message: msgServiceUnavailable},
	{pattern: "unavailable", httpStatus: fiber.StatusServiceUnavailable, message: msgServiceUnavailable},
}

// MapGRPCError extracts a clean HTTP status code and user-facing message
// from a gRPC error. It first checks pattern-based rules on the error
// description, then falls back to gRPC status code mapping.
//
// For non-gRPC errors, it falls back to 500 with a generic message.
func MapGRPCError(err error) GRPCError {
	if err == nil {
		return GRPCError{
			HTTPStatus: fiber.StatusOK,
			Message:    "OK",
		}
	}

	// Extract the full error description (this is the raw gRPC error message)
	desc := extractGRPCDescription(err)
	descLower := strings.ToLower(desc)

	// 1. Pattern-based matching (most specific first)
	for _, rule := range errorRules {
		if strings.Contains(descLower, rule.pattern) {
			return GRPCError{
				HTTPStatus: rule.httpStatus,
				Message:    rule.message,
			}
		}
	}

	// 2. Fall back to gRPC status code mapping
	if st, ok := status.FromError(err); ok {
		httpStatus, exists := grpcCodeToHTTP[st.Code()]
		if !exists {
			httpStatus = fiber.StatusInternalServerError
		}

		return GRPCError{
			HTTPStatus: httpStatus,
			Message:    fallbackMessage(st.Code()),
		}
	}

	// 3. Non-gRPC error — generic 500
	return GRPCError{
		HTTPStatus: fiber.StatusInternalServerError,
		Message:    "An unexpected error occurred",
	}
}

// extractGRPCDescription extracts the description from a gRPC error.
// If it's not a gRPC status error, returns the raw error string.
func extractGRPCDescription(err error) string {
	if st, ok := status.FromError(err); ok {
		return st.Message()
	}
	return err.Error()
}

// fallbackMessage returns a generic user-facing message for a gRPC status code.
func fallbackMessage(code codes.Code) string {
	switch code {
	case codes.InvalidArgument:
		return "Invalid request"
	case codes.NotFound:
		return "Resource not found"
	case codes.AlreadyExists:
		return "Resource already exists"
	case codes.PermissionDenied:
		return "Permission denied"
	case codes.Unauthenticated:
		return "Authentication required"
	case codes.ResourceExhausted:
		return "Too many requests. Please try again later"
	case codes.FailedPrecondition:
		return "Operation cannot be performed in current state"
	case codes.Aborted:
		return "Operation was aborted. Please retry"
	case codes.Unimplemented:
		return "This feature is not yet available"
	case codes.Unavailable:
		return msgServiceUnavailable
	case codes.DeadlineExceeded:
		return "Request timed out. Please try again"
	default:
		return "An unexpected error occurred"
	}
}
