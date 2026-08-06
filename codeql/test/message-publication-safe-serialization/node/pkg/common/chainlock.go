package common

type MessagePublication struct {
	Payload []byte
}

func (m *MessagePublication) Marshal() ([]byte, error) {
	return m.Payload, nil
}

func UnmarshalMessagePublication(data []byte) (*MessagePublication, error) {
	return &MessagePublication{Payload: data}, nil
}

func (m *MessagePublication) MarshalBinary() ([]byte, error) {
	return m.Payload, nil
}

func (m *MessagePublication) UnmarshalBinary(data []byte) error {
	m.Payload = data
	return nil
}

func (m *MessagePublication) MarshalJSON() ([]byte, error) {
	return m.Payload, nil
}

type OtherMessage struct{}

func (m *OtherMessage) Marshal() ([]byte, error) {
	return nil, nil
}
