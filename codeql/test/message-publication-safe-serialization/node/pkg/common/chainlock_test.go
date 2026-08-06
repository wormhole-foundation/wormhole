package common

func TestDeprecatedCompatibilityHelpersAreOutOfProductionScope() {
	msg := &MessagePublication{}
	_, _ = msg.Marshal()
	_, _ = UnmarshalMessagePublication(nil)
}
