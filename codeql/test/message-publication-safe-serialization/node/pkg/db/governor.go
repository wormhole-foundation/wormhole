package db

import "github.com/wormhole-foundation/wormhole/codeqltest/messagepublication/node/pkg/common"

type PendingTransfer struct {
	Msg *common.MessagePublication
}

func DeprecatedCurrentWrite(p *PendingTransfer) ([]byte, error) {
	return p.Msg.Marshal()
}

func CurrentWrite(p *PendingTransfer) ([]byte, error) {
	bz, err := p.Msg.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return bz, nil
}

func DeprecatedCurrentRead(buf []byte) (*common.MessagePublication, error) {
	return common.UnmarshalMessagePublication(buf)
}

func UnmarshalPendingTransfer(buf []byte, isOld bool) (*PendingTransfer, error) {
	if isOld {
		oldMsg, err := common.UnmarshalMessagePublication(buf)
		if err != nil {
			return nil, err
		}
		if len(buf) == 0 {
			_, err := oldMsg.Marshal()
			if err != nil {
				return nil, err
			}
		}
		return &PendingTransfer{Msg: oldMsg}, nil
	}
	if isOld == true {
		oldMsg, err := common.UnmarshalMessagePublication(buf)
		if err != nil {
			return nil, err
		}
		return &PendingTransfer{Msg: oldMsg}, nil
	}
	if len(buf) > 0 {
		isOld := true
		if isOld {
			_, err := common.UnmarshalMessagePublication(buf)
			return nil, err
		}
	}

	msg := &common.MessagePublication{}
	if err := msg.UnmarshalBinary(buf); err != nil {
		return nil, err
	}
	return &PendingTransfer{Msg: msg}, nil
}
