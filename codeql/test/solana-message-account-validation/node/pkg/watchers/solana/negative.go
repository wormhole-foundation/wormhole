package solana

import "context"

func directConstructorParse(raw []byte) error {
	data, err := NewMessageAccountData(raw)
	if err != nil {
		return err
	}
	alias := data
	ParseMessagePublicationAccount(alias)
	return nil
}

func safeFactory(raw []byte) (MessageAccountData, error) {
	data, err := NewMessageAccountData(raw)
	if err != nil {
		return MessageAccountData{}, err
	}
	return data, nil
}

func checkedSafeFactoryProcess(ctx context.Context, watcher *SolanaWatcher, raw []byte) error {
	data, err := safeFactory(raw)
	if err != nil {
		return err
	}
	watcher.processMessageAccount(ctx, data)
	return nil
}

func pointerValueRoundTrip(raw []byte) error {
	data, err := NewMessageAccountData(raw)
	if err != nil {
		return err
	}
	ptr := &data
	ParseMessagePublicationAccount(*ptr)
	return nil
}

func nestedSafeFactory(raw []byte) (MessageAccountData, error) {
	data, err := safeFactory(raw)
	if err != nil {
		return MessageAccountData{}, err
	}
	return data, nil
}

func checkedNestedSafeFactoryParse(raw []byte) error {
	data, err := nestedSafeFactory(raw)
	if err != nil {
		return err
	}
	ParseMessagePublicationAccount(data)
	return nil
}

func constructorErrorOverwrittenAfterGuard(raw []byte) error {
	data, err := NewMessageAccountData(raw)
	if err != nil {
		return err
	}
	err = nil
	ParseMessagePublicationAccount(data)
	return err
}
