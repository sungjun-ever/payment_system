package domain

type Resource struct {
	Succeeded    bool   `json:"succeeded"`
	FailedReason string `json:"failed_reason"`
	Retryable    bool   `json:"retryable"`
}

// NewResource retryable true의 경우 같은 멱등키로 재시도 가능
func NewResource(succeeded bool, failedReason string, retry bool) *Resource {
	return &Resource{succeeded, failedReason, retry}
}
