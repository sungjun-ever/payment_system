package domain

type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "ACTIVE"
	ProductStatusInactive ProductStatus = "INACTIVE"
	ProductStatusSoldOut  ProductStatus = "SOLD_OUT"
	ProductStatusDeleted  ProductStatus = "DELETED"
)
