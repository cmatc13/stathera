// pkg/errors/api.go
package errors

// API error codes
const (
	// APIErrBadRequest indicates a bad request
	APIErrBadRequest = "API_BAD_REQUEST"
	// APIErrUnauthorized indicates an unauthorized request
	APIErrUnauthorized = "API_UNAUTHORIZED"
	// APIErrForbidden indicates a forbidden request
	APIErrForbidden = "API_FORBIDDEN"
	// APIErrNotFound indicates a resource was not found
	APIErrNotFound = "API_NOT_FOUND"
	// APIErrMethodNotAllowed indicates a method is not allowed
	APIErrMethodNotAllowed = "API_METHOD_NOT_ALLOWED"
	// APIErrConflict indicates a conflict
	APIErrConflict = "API_CONFLICT"
	// APIErrInternalServer indicates an internal server error
	APIErrInternalServer = "API_INTERNAL_SERVER"
	// APIErrServiceUnavailable indicates a service is unavailable
	APIErrServiceUnavailable = "API_SERVICE_UNAVAILABLE"
	// APIErrRateLimitExceeded indicates a rate limit was exceeded
	APIErrRateLimitExceeded = "API_RATE_LIMIT_EXCEEDED"
	// APIErrValidation indicates a validation error
	APIErrValidation = "API_VALIDATION"
	// APIErrJWTInvalid indicates an invalid JWT
	APIErrJWTInvalid = "API_JWT_INVALID"
	// APIErrJWTExpired indicates an expired JWT
	APIErrJWTExpired = "API_JWT_EXPIRED"
)

// API domain name
const APIDomain = "api"

// API operations
const (
	OpHandleRequest         = "HandleRequest"
	OpAuthenticate          = "Authenticate"
	OpAuthorize             = "Authorize"
	OpValidateInput         = "ValidateInput"
	OpGenerateResponse      = "GenerateResponse"
	OpGenerateToken         = "GenerateToken"
	OpVerifyToken           = "VerifyToken"
	OpRegisterUser          = "RegisterUser"
	OpLoginUser             = "LoginUser"
	OpGetUserInfo           = "GetUserInfo"
	OpUpdateUserInfo        = "UpdateUserInfo"
	OpDeleteUser            = "DeleteUser"
	OpGetResource           = "GetResource"
	OpCreateResource        = "CreateResource"
	OpUpdateResource        = "UpdateResource"
	OpDeleteResource        = "DeleteResource"
	OpListResources         = "ListResources"
	OpHandleWebhook         = "HandleWebhook"
	OpProcessCallback       = "ProcessCallback"
	OpRateLimitCheck        = "RateLimitCheck"
	OpCORSCheck             = "CORSCheck"
	OpParseRequestBody      = "ParseRequestBody"
	OpSerializeResponse     = "SerializeResponse"
	OpLogRequest            = "LogRequest"
	OpLogResponse           = "LogResponse"
	OpHandleMiddleware      = "HandleMiddleware"
	OpRouteRequest          = "RouteRequest"
	OpStartServer           = "StartServer"
	OpShutdownServer        = "ShutdownServer"
	OpHealthCheck           = "HealthCheck"
	OpMetricsCollection     = "MetricsCollection"
	OpTraceRequest          = "TraceRequest"
	OpHandleError           = "HandleError"
	OpGenerateErrorResponse = "GenerateErrorResponse"
)

// NewAPIError creates a new API error
func NewAPIError(code string, message string, err error) error {
	return &Error{
		Domain:   APIDomain,
		Code:     code,
		Message:  message,
		Original: err,
	}
}

// APIErrorf creates a new API error with formatted message
func APIErrorf(code string, format string, args ...interface{}) error {
	return &Error{
		Domain:  APIDomain,
		Code:    code,
		Message: Sprintf(format, args...),
	}
}

// APIWrap wraps an error with API domain
func APIWrap(err error, operation string, message string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Domain:    APIDomain,
		Operation: operation,
		Message:   message,
		Original:  err,
	}
}

// APIWrapWithCode wraps an error with API domain and code
func APIWrapWithCode(err error, operation string, code string, message string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Domain:    APIDomain,
		Operation: operation,
		Code:      code,
		Message:   message,
		Original:  err,
	}
}

// IsAPIError checks if an error is an API error with the given code
func IsAPIError(err error, code string) bool {
	var domainErr *Error
	if As(err, &domainErr) {
		return domainErr.Domain == APIDomain && domainErr.Code == code
	}
	return false
}

// HTTPStatusFromAPIError returns the HTTP status code for an API error
func HTTPStatusFromAPIError(err error) int {
	var domainErr *Error
	if !As(err, &domainErr) || domainErr.Domain != APIDomain {
		return 500 // Internal Server Error
	}

	switch domainErr.Code {
	case APIErrBadRequest, APIErrValidation:
		return 400 // Bad Request
	case APIErrUnauthorized, APIErrJWTInvalid, APIErrJWTExpired:
		return 401 // Unauthorized
	case APIErrForbidden:
		return 403 // Forbidden
	case APIErrNotFound:
		return 404 // Not Found
	case APIErrMethodNotAllowed:
		return 405 // Method Not Allowed
	case APIErrConflict:
		return 409 // Conflict
	case APIErrRateLimitExceeded:
		return 429 // Too Many Requests
	case APIErrServiceUnavailable:
		return 503 // Service Unavailable
	default:
		return 500 // Internal Server Error
	}
}
