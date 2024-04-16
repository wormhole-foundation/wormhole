package db

import (
	"fmt"
	"os"
	"path"

	"github.com/dgraph-io/badger/v3"
	"go.uber.org/zap"
)

type badgerZapLogger struct {
	*zap.Logger
}

func (l badgerZapLogger) Errorf(f string, v ...interface{}) {
	l.Error(fmt.Sprintf(f, v...))
}

func (l badgerZapLogger) Warningf(f string, v ...interface{}) {
	l.Warn(fmt.Sprintf(f, v...))
}

func (l badgerZapLogger) Infof(f string, v ...interface{}) {
	l.Info(fmt.Sprintf(f, v...))
}

func (l badgerZapLogger) Debugf(f string, v ...interface{}) {
	l.Debug(fmt.Sprintf(f, v...))
}

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

	if logger != nil {
		options = options.WithLogger(badgerZapLogger{logger})
	}

	db, err := badger.Open(options)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}

	return &Database{
		db: db,
	}
}
