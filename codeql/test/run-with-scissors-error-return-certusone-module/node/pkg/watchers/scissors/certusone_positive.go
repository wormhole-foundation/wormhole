package scissors

import (
	"context"
	"errors"

	"github.com/certusone/wormhole/node/pkg/common"
)

func legacyModulePathInlineDirectSend(ctx context.Context, errC chan error) {
	common.RunWithScissors(ctx, errC, "legacy-inline", func() error {
		err := errors.New("legacy inline failed")
		errC <- err
		return nil
	})
}
