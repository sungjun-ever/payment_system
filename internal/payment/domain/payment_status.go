package payment

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusRequested Status = "REQUESTED"
	StatusSucceeded Status = "SUCCEEDED"
	StatusFailed    Status = "FAILED"
	StatusCanceled  Status = "CANCELED"
	StatusRefunded  Status = "REFUNDED"
)
