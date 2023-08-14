package types

import (
	"encoding/json"
	"time"
)

type PacketMetadata struct {
	Forward *ForwardMetadata `json:"forward"`
}

type ForwardMetadata struct {
	Receiver string        `json:"receiver"`
	Port     string        `json:"port"`
	Channel  string        `json:"channel"`
	Timeout  time.Duration `json:"timeout"`
	Retries  *uint8        `json:"retries,omitempty"`
	Next     *string       `json:"next,omitempty"`
}

func FormatPfmMemo(parsedPayload ParsedPayload, resp []byte, timeout time.Duration, retries uint8) (string, error) {
	var queryRsp IbcTranslatorQueryRsp
	err := json.Unmarshal(resp, &queryRsp)
	if err != nil {
		return "", err
	}

	forwardMetadata := ForwardMetadata{
		Receiver: string(parsedPayload.Recipient),
		Port:     "transfer",
		Channel:  queryRsp.Channel,
		Timeout:  timeout,
		Retries:  &retries,
	}
	if !parsedPayload.NoPayload {
		next := string(parsedPayload.Payload)
		forwardMetadata.Next = &next
	}

	packet := PacketMetadata{Forward: &forwardMetadata}
	packetBz, err := json.Marshal(&packet)
	if err != nil {
		return "", err
	}

	return string(packetBz), nil
}
