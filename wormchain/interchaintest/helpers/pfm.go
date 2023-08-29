package helpers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type PacketMetadata struct {
	Forward *ForwardMetadata `json:"forward"`
}

type ForwardMetadata struct {
	Receiver       string        `json:"receiver"`
	Port           string        `json:"port"`
	Channel        string        `json:"channel"`
	Timeout        time.Duration `json:"timeout"`
	Retries        *uint8        `json:"retries,omitempty"`
	Next           *string       `json:"next,omitempty"`
	RefundSequence *uint64       `json:"refund_sequence,omitempty"`
}

func CreatePfmSimpleMsg(t *testing.T, recipient string, channel string) string {
	retries := uint8(0)
	msg := &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: recipient,
			Port:     "transfer",
			Channel:  channel,
			Timeout:  time.Minute * 10,
			Retries:  &retries,
		},
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
}

func CreatePfmContractControlledMsg(t *testing.T, contract string, channel string, recipient string) string {
	ibchooks := &IbcHooks{
		Payload: IbcHooksPayload{
			Contract: contract,
			Msg: IbcHooksExecute{
				Forward: IbcHooksForward{
					Recipient: recipient,
				},
			},
		},
	}

	nextBz, err := json.Marshal(ibchooks)
	require.NoError(t, err)

	next := string(nextBz)

	retries := uint8(0)
	msg := &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: contract,
			Port:     "transfer",
			Channel:  channel,
			Timeout:  1 * time.Minute,
			Retries:  &retries,
			Next:     &next,
		},
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
}
