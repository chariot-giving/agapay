package adb

import (
	"os"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var AgapayDatabase *AgapayDB

type AgapayDB struct {
	*gorm.DB
	logger *zap.Logger
}

func NewAgapayDatabase() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		panic("DATABASE_URL environment variable is not set")
	}
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "public.",
			SingularTable: true,
		},
		PrepareStmt: true, // Creates a prepared statement when executing any SQL and caches them to speed up future calls
	})
	if err != nil {
		panic(err)
	}
	sqlDb, err := db.DB()
	if err != nil {
		panic(err)
	}
	// configure connection pool settings
	sqlDb.SetMaxIdleConns(3)
	sqlDb.SetMaxOpenConns(10)
	sqlDb.SetConnMaxLifetime(time.Hour)

	AgapayDatabase = &AgapayDB{
		DB:     db,
		logger: zap.L(),
	}
}

type IdempotentRequest struct {
	UserId         uint64
	IdempotencyKey string
	Method         string
	Path           string
	Params         map[string]string
	Body           any
}
