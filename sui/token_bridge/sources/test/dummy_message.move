#[test_only]
module token_bridge::dummy_message {
    public fun encoded_transfer(): vector<u8> {
        // let decimals = 8;
        // let expected_amount = normalized_amount::from_raw(234567890, decimals);
        // let expected_token_address = external_address::from_any_bytes(x"beef");
        // let expected_token_chain = 1;
        // let expected_recipient = external_address::from_any_bytes(x"cafe");
        // let expected_recipient_chain = 7;
        // let expected_relayer_fee =
        //     normalized_amount::from_raw(123456789, decimals);
        x"01000000000000000000000000000000000000000000000000000000000dfb38d2000000000000000000000000000000000000000000000000000000000000beef0001000000000000000000000000000000000000000000000000000000000000cafe000700000000000000000000000000000000000000000000000000000000075bcd15"
    }

    public fun encoded_transfer_with_payload(): vector<u8> {
        // let expected_amount = normalized_amount::from_raw(234567890, 8);
        // let expected_token_address = external_address::from_any_bytes(x"beef");
        // let expected_token_chain = 1;
        // let expected_recipient = external_address::from_any_bytes(x"cafe");
        // let expected_recipient_chain = 7;
        // let expected_sender = external_address::from_any_bytes(x"deadbeef");
        // let expected_payload = b"All your base are belong to us.";
        x"03000000000000000000000000000000000000000000000000000000000dfb38d2000000000000000000000000000000000000000000000000000000000000beef0001000000000000000000000000000000000000000000000000000000000000cafe000700000000000000000000000000000000000000000000000000000000deadbeef416c6c20796f75722062617365206172652062656c6f6e6720746f2075732e"
    }

    public fun encoded_register_chain_2(): vector<u8> {
        x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef"
    }
}
