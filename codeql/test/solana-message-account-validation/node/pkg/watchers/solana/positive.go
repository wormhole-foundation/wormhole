package solana

import (
	"context"
	"errors"
)

type MessageAccountData struct {
	Data []byte
}

type SolanaWatcher struct{}

func NewMessageAccountData(data []byte) (MessageAccountData, error) {
	if len(data) == 0 {
		return MessageAccountData{}, errors.New("empty")
	}
	return MessageAccountData{Data: data}, nil
}

func ParseMessagePublicationAccount(data MessageAccountData) {}

func (w *SolanaWatcher) processMessageAccount(ctx context.Context, data MessageAccountData) {}

func uncheckedLiteralParse(raw []byte) {
	ParseMessagePublicationAccount(MessageAccountData{Data: raw})
}

func manualFactory(raw []byte) (MessageAccountData, error) {
	if len(raw) == 0 {
		return MessageAccountData{}, errors.New("empty")
	}
	return MessageAccountData{Data: raw}, nil
}

func checkedManualFactoryParse(raw []byte) error {
	data, err := manualFactory(raw)
	if err != nil {
		return err
	}
	ParseMessagePublicationAccount(data)
	return nil
}

func rawPrefixCheckThenProcess(ctx context.Context, watcher *SolanaWatcher, raw []byte) {
	if len(raw) < 3 || string(raw[:3]) != "msg" {
		return
	}
	data := MessageAccountData{Data: raw}
	watcher.processMessageAccount(ctx, data)
}

func constructorErrorNotRejected(raw []byte) {
	data, _ := NewMessageAccountData(raw)
	ParseMessagePublicationAccount(data)
}

func constructorErrorOverwrittenBeforeGuard(raw []byte) error {
	data, err := NewMessageAccountData(raw)
	err = errors.New("replacement")
	if err != nil {
		return err
	}
	ParseMessagePublicationAccount(data)
	return nil
}

func priorErrorGuardDoesNotValidateConstructor(raw []byte) error {
	_, err := manualFactory(raw)
	if err != nil {
		return err
	}
	data, err := NewMessageAccountData(raw)
	ParseMessagePublicationAccount(data)
	return err
}

func sameLineConstructorErrorOverwrite(raw []byte) error {
	data, err := NewMessageAccountData(raw); err = errors.New("replacement"); if err != nil {
		return err
	}
	ParseMessagePublicationAccount(data)
	return nil
}
