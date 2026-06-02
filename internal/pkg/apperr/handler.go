package apperr

type AppError struct {
	Level   ErrorLevel        `json:"level,default:INFO"`
	Status  int               `json:"status"`
	Code    ErrorCode         `json:"code"`
	Message error             `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func (e *AppError) Error() string { return e.Message.Error() }
func (e *AppError) Unwrap() error { return nil }

func NewAppError(level ErrorLevel, status int, code ErrorCode, message error, details map[string]string) *AppError {
	return &AppError{
		Level:   level,
		Status:  status,
		Code:    code,
		Message: message,
		Details: details,
	}
}
