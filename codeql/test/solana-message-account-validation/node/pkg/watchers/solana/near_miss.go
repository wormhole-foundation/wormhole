package solana

func mixedFactory(raw []byte, useManual bool) (MessageAccountData, error) {
	if useManual {
		return MessageAccountData{Data: raw}, nil
	}
	data, err := NewMessageAccountData(raw)
	if err != nil {
		return MessageAccountData{}, err
	}
	return data, nil
}

func checkedMixedFactoryParse(raw []byte, useManual bool) error {
	data, err := mixedFactory(raw, useManual)
	if err != nil {
		return err
	}
	ParseMessagePublicationAccount(data)
	return nil
}

func localPrefixCheckButCtor(raw []byte) error {
	if len(raw) < 3 || string(raw[:3]) != "msg" {
		return nil
	}
	data, err := NewMessageAccountData(raw)
	if err != nil {
		return err
	}
	ParseMessagePublicationAccount(data)
	return nil
}

func nestedMixedFactory(raw []byte, useManual bool) (MessageAccountData, error) {
	data, err := mixedFactory(raw, useManual)
	if err != nil {
		return MessageAccountData{}, err
	}
	return data, nil
}

func checkedNestedMixedFactoryParse(raw []byte, useManual bool) error {
	data, err := nestedMixedFactory(raw, useManual)
	if err != nil {
		return err
	}
	ParseMessagePublicationAccount(data)
	return nil
}
