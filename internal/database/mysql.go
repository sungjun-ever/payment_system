package database

import (
	"fmt"
	"log"
	"order_system/internal/config"
	idempotencyDomain "order_system/internal/idempotency/domain"
	orderDomain "order_system/internal/order/domain"
	paymentDomain "order_system/internal/payment/domain"
	productDomain "order_system/internal/product/domain"
	userDomain "order_system/internal/user/domain"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewMysql(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MysqlDBUser,
		cfg.MysqlDBPass,
		cfg.MysqlDBHost,
		cfg.MysqlDBPort,
		cfg.MysqlDBName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt:    true,
		TranslateError: true,
	})

	if err != nil {
		log.Fatalf("Database connection failed: %s", err)
	}

	_ = db.AutoMigrate(
		&userDomain.User{},
		&productDomain.Product{},
		&orderDomain.Order{},
		&orderDomain.OrderItem{},
		&productDomain.Inventory{},
		&paymentDomain.Payment{},
		&idempotencyDomain.IdempotencyKey{},
	)

	return db
}
