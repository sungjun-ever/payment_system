package payment

type Method string

const (
	Card           Method = "CARD"
	VirtualAccount Method = "VIRTUAL_ACCOUNT"
	Transfer       Method = "TRANSFER"
	Point          Method = "POINT"
)
