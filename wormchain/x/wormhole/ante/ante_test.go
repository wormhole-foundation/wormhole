package ante_test

import sdk "github.com/cosmos/cosmos-sdk/types"

// Define an empty ante handle
var (
	EmptyAnte = func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	}
)

type MockTx struct {
	msgs []sdk.Msg
}

func NewMockTx(msgs ...sdk.Msg) MockTx {
	return MockTx{
		msgs: msgs,
	}
}

func (tx MockTx) GetMsgs() []sdk.Msg {
	return tx.msgs
}

func (tx MockTx) ValidateBasic() error {
	return nil
}
