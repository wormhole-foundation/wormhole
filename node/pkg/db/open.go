package db

import (
	"os"
	"path"

	"go.uber.org/zap"
)

func OpenDb(logger *zap.Logger, dataDir *string) *Database {
	dbPath := path.Join(*dataDir, "db")
	if err := os.MkdirAll(dbPath, 0700); err != nil {
		logger.Fatal("failed to create database directory", zap.Error(err))
	}
	db, err := Open(dbPath)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}

	return db
}
