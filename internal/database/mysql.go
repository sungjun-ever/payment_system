package database

import (
	"fmt"
	"log"
	"payment_system/internal/config"
	"payment_system/internal/model"

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
		&model.User{},
		&model.Product{},
		&model.Order{},
		&model.OrderItem{},
		&model.Inventory{},
		&model.Payment{},
		&model.IdempotencyKey{},
	)

	return db
}
