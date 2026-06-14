package apperr

type ErrorCode string

const (
	A001 ErrorCode = "EXPIRED_TOKEN"
	A002 ErrorCode = "INVALID_TOKEN"
	A003 ErrorCode = "AUTH_REQUIRED"
	A004 ErrorCode = "TOKEN_CONFLICT"
	A005 ErrorCode = "INVALID_CREDENTIALS"

	C001 ErrorCode = "INVALID_INPUT"
	C002 ErrorCode = "RESOURCE_NOT_FOUND"
	C003 ErrorCode = "CONFLICT"
	C004 ErrorCode = "TOO_MANY_REQUEST"

	S001 ErrorCode = "SYSTEM_ERROR"
	S002 ErrorCode = "SYSTEM_TIMEOUT"
	S003 ErrorCode = "SYSTEM_MAINTENANCE"

	I001 ErrorCode = "IDEMPOTENCY_NOT_FOUND"
	I002 ErrorCode = "IDEMPOTENCY_CONFLICT"
)

func (e ErrorCode) String() string {
	return string(e)
}
