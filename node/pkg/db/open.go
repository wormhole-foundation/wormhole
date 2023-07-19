package db

import (
	"os"
	"path"

	"github.com/dgraph-io/badger/v3"
	"go.uber.org/zap"
)

func OpenDb(logger *zap.Logger, dataDir *string) *Database {
	var options badger.Options

	if dataDir != nil {
		dbPath := path.Join(*dataDir, "db")
		if err := os.MkdirAll(dbPath, 0700); err != nil {
			logger.Fatal("failed to create database directory", zap.Error(err))
		}

		options = badger.DefaultOptions(dbPath)
	} else {
		options = badger.DefaultOptions("").WithInMemory(true)
	}

	db, err := badger.Open(options)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}

	return &Database{
		db: db,
	}
}
