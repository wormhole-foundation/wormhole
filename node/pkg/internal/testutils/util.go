package testutils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// MustGetMockGuardianTssStorage returns the path to a mock guardian storage file.
func MustGetMockGuardianTssStorage() string {
	str, err := GetMockGuardianTssStorage(0)
	if err != nil {
		panic(err)
	}
	return str
}

func GetMockGuardianTssStorage(guardianIndex int, guardianTssStorageSet ...string) (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("could not get runtime.Caller(0)")
	}

	setFolder := "tss5"
	if len(guardianTssStorageSet) > 0 {
		setFolder = guardianTssStorageSet[0]
	}
	guardianStorageFname := path.Join(path.Dir(file), "testdata", setFolder, fmt.Sprintf("guardian%d.json", guardianIndex))
	return guardianStorageFname, nil
}

func MakeSupervisorContext(ctx context.Context) context.Context {
	var supervisedCtx context.Context

	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.AddSync(zapcore.Lock(os.Stderr)),
			zap.NewAtomicLevelAt(zapcore.Level(zapcore.DebugLevel)),
		),
	)

	// used to block this function until the supervisor sets the supervisedCtx
	barrier := make(chan struct{})

	supervisor.New(ctx, logger, func(ctx context.Context) error {
		supervisedCtx = ctx

		close(barrier)

		<-ctx.Done()
		return ctx.Err()
	})

	<-barrier
	return supervisedCtx
}
