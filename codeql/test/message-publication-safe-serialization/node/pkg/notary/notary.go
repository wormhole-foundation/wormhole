package notary

import wormcommon "github.com/wormhole-foundation/wormhole/codeqltest/messagepublication/node/pkg/common"

type MP = wormcommon.MessagePublication

type EmbeddedPublication struct {
	wormcommon.MessagePublication
}

var marshalPublication = (*wormcommon.MessagePublication).Marshal
var unmarshalPublication = wormcommon.UnmarshalMessagePublication

func ImportAliasDeprecatedRead(buf []byte) (*wormcommon.MessagePublication, error) {
	return wormcommon.UnmarshalMessagePublication(buf)
}

func TypeAliasDeprecatedWrite(msg *MP) ([]byte, error) {
	return msg.Marshal()
}

func EmbeddedExplicitDeprecatedWrite(wrapper *EmbeddedPublication) ([]byte, error) {
	return wrapper.MessagePublication.Marshal()
}

func EmbeddedPromotedDeprecatedWrite(wrapper *EmbeddedPublication) ([]byte, error) {
	return wrapper.Marshal()
}

func CurrentBinaryAPIs(msg *wormcommon.MessagePublication, buf []byte) ([]byte, error) {
	bz, err := msg.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if err := msg.UnmarshalBinary(buf); err != nil {
		return nil, err
	}
	return bz, nil
}

func OutOfScopeMarshalJSON(msg *wormcommon.MessagePublication) ([]byte, error) {
	return msg.MarshalJSON()
}

func OutOfScopeOtherMarshal(msg *wormcommon.OtherMessage) ([]byte, error) {
	return msg.Marshal()
}

func UseCapturedHelpers(msg *wormcommon.MessagePublication, buf []byte) {
	_, _ = marshalPublication(msg)
	_, _ = unmarshalPublication(buf)
}

func BoundMethodValue(msg *wormcommon.MessagePublication) {
	marshal := msg.Marshal
	_, _ = marshal()
}

func ParenthesizedDeprecatedRead(buf []byte) (*wormcommon.MessagePublication, error) {
	return (wormcommon.UnmarshalMessagePublication)(buf)
}
