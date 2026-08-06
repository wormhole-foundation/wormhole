package scissors

import (
	"context"
	"errors"

	"github.com/wormhole-foundation/wormhole/node/pkg/common"
)

type watcher struct {
	errC        chan error
	metricsErrC chan error
}

func inlineDirectSend(ctx context.Context, errC chan error) {
	common.RunWithScissors(ctx, errC, "inline", func() error {
		err := errors.New("inline failed")
		errC <- err
		return nil
	})
}

func inlineReceiverFieldSend(ctx context.Context, w *watcher) {
	common.RunWithScissors(ctx, w.errC, "receiver-field", func() error {
		err := errors.New("receiver failed")
		w.errC <- err
		return err
	})
}

func localFunctionValueSend(ctx context.Context, errC chan error) {
	run := func() error {
		err := errors.New("local function failed")
		errC <- err
		return nil
	}
	common.RunWithScissors(ctx, errC, "local", run)
}

func methodValueSend(ctx context.Context, w *watcher) {
	common.RunWithScissors(ctx, w.errC, "method", w.run)
}

func (w *watcher) run() error {
	err := errors.New("method failed")
	w.errC <- err
	return nil
}

func helperChannelSend(ctx context.Context, errC chan error) {
	common.RunWithScissors(ctx, errC, "helper-channel", func() error {
		return reportOnChannel(errC, errors.New("helper failed"))
	})
}

func reportOnChannel(ch chan error, err error) error {
	ch <- err
	return nil
}

func helperReceiverSend(ctx context.Context, w *watcher) {
	common.RunWithScissors(ctx, w.errC, "helper-receiver", func() error {
		return w.reportOnReceiver(errors.New("receiver helper failed"))
	})
}

func (w *watcher) reportOnReceiver(err error) error {
	w.errC <- err
	return nil
}
