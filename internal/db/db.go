package db

import (
	"context"
	"huahua-service/internal/config"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	GORM  *gorm.DB
	Redis *redis.Client
)

func InitGorm(cfg config.DatabaseConfig) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("gorm连接失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("postGresql连接失败: %v", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	GORM = db
	log.Println("数据库已连接")

	AutoMigrate()
}

func InitRedis(cfg config.RedisConfig) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis连接失败：%v", err)
	}
	Redis = client
	log.Printf("Redis已连接")
}

func Close() {
	if GORM != nil {
		if sqlDB, err := GORM.DB(); err == nil {
			sqlDB.Close()
		}
	}
	if Redis != nil {
		Redis.Close()
	}
}
