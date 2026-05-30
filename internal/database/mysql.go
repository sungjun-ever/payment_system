package database

import (
	"fmt"
	"log"
	"payment_system/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Mysql struct {
	conn *gorm.DB
}

func NewMysql(cfg *config.Config) *Mysql {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MysqlDBUser,
		cfg.MysqlDBPass,
		cfg.MysqlDBHost,
		cfg.MysqlDBPort,
		cfg.MysqlDBName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})

	if err != nil {
		log.Fatalf("Database connection failed: %s", err)
	}

	return &Mysql{
		conn: db,
	}
}
