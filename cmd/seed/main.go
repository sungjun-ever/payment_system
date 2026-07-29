package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"order_system/internal/config"
	"order_system/internal/database"
	"order_system/internal/pkg/rediskey"
	productdomain "order_system/internal/product/domain"
	appredis "order_system/internal/redis"
	userdomain "order_system/internal/user/domain"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	userSeedCount    = 100
	productSeedCount = 30
)

type seededProduct struct {
	product   productdomain.Product
	inventory productdomain.Inventory
}

func main() {
	cfg := config.Load()

	db := database.NewMysql(cfg)
	rdb := appredis.NewRedis(cfg)
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Printf("redis close failed: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	products, err := seedMysql(ctx, db)
	if err != nil {
		log.Fatalf("mysql seed failed: %v", err)
	}

	if err := seedRedis(ctx, rdb, products); err != nil {
		log.Fatalf("redis seed failed: %v", err)
	}

	log.Printf("seed completed: users=%d products=%d", userSeedCount, productSeedCount)
}

func seedMysql(ctx context.Context, db *gorm.DB) ([]seededProduct, error) {
	var products []seededProduct

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := truncateTables(tx); err != nil {
			return err
		}

		users := make([]userdomain.User, 0, userSeedCount)
		for i := 1; i <= userSeedCount; i++ {
			users = append(users, userdomain.User{
				Email:    fmt.Sprintf("user%03d@example.com", i),
				Name:     fmt.Sprintf("User %03d", i),
				Password: "password123!",
			})
		}

		if err := tx.CreateInBatches(&users, userSeedCount).Error; err != nil {
			return fmt.Errorf("create users: %w", err)
		}

		products = make([]seededProduct, 0, productSeedCount)
		for i := 1; i <= productSeedCount; i++ {
			description := fmt.Sprintf("Seed product %02d for local docker development", i)
			product := productdomain.Product{
				Name:        fmt.Sprintf("Seed Product %02d", i),
				Description: &description,
				Price:       int64(1000 + i*250),
				Status:      productdomain.StatusActive,
			}

			if err := tx.Create(&product).Error; err != nil {
				return fmt.Errorf("create product %d: %w", i, err)
			}

			inventory := productdomain.Inventory{
				ProductID:        product.ID,
				TotalQuantity:    100 + i*5,
				ReservedQuantity: rand.Intn(50),
				SoldQuantity:     rand.Intn(30),
			}

			if err := tx.Create(&inventory).Error; err != nil {
				return fmt.Errorf("create inventory for product %d: %w", product.ID, err)
			}

			products = append(products, seededProduct{
				product:   product,
				inventory: inventory,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return products, nil
}

func truncateTables(db *gorm.DB) error {
	tables := []string{
		"payment_attempts",
		"payments",
		"idempotency_keys",
		"inventory_movements",
		"inventory_jobs",
		"order_items",
		"orders",
		"inventories",
		"products",
		"users",
	}

	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		return fmt.Errorf("disable foreign key checks: %w", err)
	}
	defer func() {
		if err := db.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
			log.Printf("enable foreign key checks failed: %v", err)
		}
	}()

	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			continue
		}

		if err := db.Exec(fmt.Sprintf("TRUNCATE TABLE `%s`", table)).Error; err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}

	return nil
}

func seedRedis(ctx context.Context, rdb *redis.Client, products []seededProduct) error {
	if err := rdb.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("flush redis: %w", err)
	}

	pipe := rdb.Pipeline()
	for _, seeded := range products {
		product := seeded.product
		inventory := seeded.inventory

		description := ""
		if product.Description != nil {
			description = *product.Description
		}

		pipe.HSet(ctx, rediskey.ProductKey(product.ID), map[string]interface{}{
			"name":        product.Name,
			"description": description,
			"price":       product.Price,
			"status":      product.Status.String(),
		})
		pipe.HSet(ctx, rediskey.ProductInventoryKey(product.ID), map[string]interface{}{
			"total_quantity":    inventory.TotalQuantity,
			"reserved_quantity": inventory.ReservedQuantity,
			"sold_quantity":     inventory.SoldQuantity,
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("populate redis: %w", err)
	}

	return nil
}
