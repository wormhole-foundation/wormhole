package scissors

import (
	"context"
	"errors"

	"github.com/wormhole-foundation/wormhole/node/pkg/common"
)

func nonRunWithScissorsGoroutine(errC chan error) {
	go func() {
		errC <- errors.New("plain goroutine")
	}()
}

func unsupportedTwoHopHelper(ctx context.Context, errC chan error) {
	common.RunWithScissors(ctx, errC, "two-hop", func() error {
		return outerHelper(errC, errors.New("two hop"))
	})
}

func outerHelper(errC chan error, err error) error {
	return reportOnChannel(errC, err)
}

func unsupportedSameScopeLocalAlias(ctx context.Context, errC chan error) {
	common.RunWithScissors(ctx, errC, "local-alias", func() error {
		ch := errC
		ch <- errors.New("local alias currently unsupported")
		return nil
	})
}

func unsupportedSameScopeReceiverFieldAlias(ctx context.Context, w *watcher) {
	common.RunWithScissors(ctx, w.errC, "receiver-alias", func() error {
		ch := w.errC
		ch <- errors.New("receiver alias currently unsupported")
		return nil
	})
}
