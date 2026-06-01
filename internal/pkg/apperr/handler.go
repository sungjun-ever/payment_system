package apperr

type ApiError struct {
	Level   ErrorLevel        `json:"level,default:INFO"`
	Status  int               `json:"status"`
	Code    ErrorCode         `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func (e *ApiError) Error() string { return e.Message }
func (e *ApiError) Unwrap() error { return nil }

func NewApiError(level ErrorLevel, status int, code ErrorCode, message string, details map[string]string) *ApiError {
	return &ApiError{
		Level:   level,
		Status:  status,
		Code:    code,
		Message: message,
		Details: details,
	}
}
