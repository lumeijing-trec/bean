// Copyright The RAI Inc.
// The RAI Authors
package error

import "errors"

type ErrorCode string

const (
	API_SUCCESS ErrorCode = "000000"

	// API general error code
	PROBLEM_PARSING_JSON     ErrorCode = "100001"
	UNAUTHORIZED_ACCESS      ErrorCode = "100002"
	RESOURCE_NOT_FOUND       ErrorCode = "100003"
	INTERNAL_SERVER_ERROR    ErrorCode = "100004"
	REQUEST_ENTITY_TOO_LARGE ErrorCode = "100005"
	METHOD_NOT_ALLOWED       ErrorCode = "100006"
	TOO_MANY_REQUESTS        ErrorCode = "100010"
	UNKNOWN_ERROR_CODE       ErrorCode = "100098"
	TIMEOUT                  ErrorCode = "100099"

	// API parameter error code
	API_DATA_VALIDATION_FAILED ErrorCode = "200001"
)

var (
	ErrInternalServer      = errors.New("internal server error")
	ErrInvalidJsonResponse = errors.New("invalid JSON response")
	ErrContextExtraction   = errors.New("some data is missing in the context")
	ErrParamMissing        = errors.New("parameters are missing")
	ErrUpstreamTimeout     = errors.New("timeout from upstream server")
	ErrTimeout             = errors.New("timeout")
)
