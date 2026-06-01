package model

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email string `gorm:"type:varchar(100);not null;uniqueIndex;column:email"`
	Name  string `gorm:"type:varchar(50);not null;column:name"`

	Orders []Order `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
