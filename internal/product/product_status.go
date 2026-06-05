package product

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
	StatusSoldOut  Status = "SOLD_OUT"
	StatusDeleted  Status = "DELETED"
)
