package domain

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
	StatusSoldOut  Status = "SOLD_OUT"
	StatusDeleted  Status = "DELETED"
)

func (s Status) String() string {
	return string(s)
}
