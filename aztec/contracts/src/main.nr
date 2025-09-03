use dep::aztec::macros::aztec;
mod structs;

/*
Aztec Wormhole contract for the Wormhole bridge on Aztec testnet.

This contract is responsible for:
- Parsing and verifying Wormhole VAA messages
- Managing guardian sets and their expiration
- Publishing messages to the Wormhole network
- Receiving messages from the Wormhole network

To deploy this contract, follow the instructions in the deploy.md file, or use the deploy.sh script.

TODO: Fix hardcoded addresses in private functions.
*/

#[aztec]
pub contract Wormhole {
    use crate::structs::{Guardian, Provider, Signature, WormholeStorage};
    use dep::keccak256::keccak256;

    use dep::aztec::{
        macros::{functions::{initializer, internal, private, public, utility}, storage::storage},
        prelude::{AztecAddress, Map, PublicMutable},
        protocol_types::traits::ToField,
    };

    use dep::token::Token;

    #[utility]
    unconstrained fn parse_vaa_unconstrained(
        bytes: [u8; 2000],
        actual_length: u32,
    ) -> (u32, u8, [[u8; 66]; 13], [u8; 1860], u32) {
        let guardian_set_index: u32 = (bytes[1] as u32) << 24
            | (bytes[2] as u32) << 16
            | (bytes[3] as u32) << 8
            | (bytes[4] as u32);

        let signatures_len = bytes[5];

        // Parse signatures
        let mut signatures = [[0; 66]; 13];
        for i in 0..13 {
            if i < signatures_len {
                let sig_start: u32 = 6 + (i as u32 * 66);
                for j in 0..66 {
                    let byte_index: u32 = sig_start + j as u32;
                    if byte_index < actual_length {
                        signatures[i as u32][j] = bytes[byte_index];
                    }
                }
            }
        }

        // Calculate where the body starts
        let body_start: u32 = 6 + (signatures_len as u32 * 66);

        // Calculate actual body length
        let body_length: u32 = actual_length - body_start;

        let mut body_bytes = [0; 1860];
        for i in 0..1860 {
            if i < body_length {
                body_bytes[i] = bytes[body_start + i];
            }
        }

        (guardian_set_index, signatures_len, signatures, body_bytes, body_length)
    }

    #[utility]
    unconstrained fn compute_vaa_hash_unconstrained(
        body_bytes: [u8; 1860],
        body_length: u32,
    ) -> [u8; 32] {
        // Extract only the actual body content
        let mut actual_body = [0; 1860];
        for i in 0..body_length {
            actual_body[i] = body_bytes[i];
        }

        // Double hash as per Wormhole spec
        let hash: [u8; 32] = keccak256(actual_body, body_length);
        let double_hash: [u8; 32] = keccak256(hash, 32);

        double_hash
    }

    #[storage]
    struct Storage<Context> {
        state: PublicMutable<WormholeStorage, Context>,
        sequences: Map<AztecAddress, PublicMutable<u64, Context>, Context>,
        receiver_address: PublicMutable<AztecAddress, Context>,
        token_address: PublicMutable<AztecAddress, Context>,
        guardian_1: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_2: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_3: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_4: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_5: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_6: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_7: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_8: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_9: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_10: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_11: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_12: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_13: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_14: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_15: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_16: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_17: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_18: Map<u32, PublicMutable<Guardian, Context>, Context>,
        guardian_19: Map<u32, PublicMutable<Guardian, Context>, Context>,
        current_guardian_set_index: Map<u32, PublicMutable<u64, Context>, Context>,
    }

    #[public]
    #[initializer]
    fn init(
        chain_id: u16,
        evm_chain_id: u16,
        receiver_address: AztecAddress,
        token_address: AztecAddress,
    ) {
        let provider: Provider = Provider { chain_id, evm_chain_id };
        storage.state.write(WormholeStorage::init(provider));
        storage.receiver_address.write(receiver_address);
        storage.token_address.write(token_address);
        storage.current_guardian_set_index.at(0).write(18_446_744_073_709_551_615);
    }

    #[public]
    fn publish_message_in_public(
        nonce: u64,
        payloads: [[u8; 31]; 8],
        message_fee: u128,
        consistency: u8,
        from: AztecAddress,
        token_nonce: Field,
    ) -> u64 {
        assert(storage.state.read().message_fee <= message_fee, "insufficient fee");

        let receiver_address = storage.receiver_address.read();
        let token_address = storage.token_address.read();

        let _ = Token::at(token_address)
            .transfer_in_public(from, receiver_address, message_fee, token_nonce)
            .call(&mut context);

        let sequence = storage.sequences.at(context.msg_sender()).read();
        storage.sequences.at(context.msg_sender()).write(sequence + 1);

        let msg: [Field; 13] = [
            context.msg_sender().to_field(),
            sequence as Field,
            nonce as Field,
            consistency as Field,
            context.timestamp() as Field,
            Field::from_le_bytes(payloads[0]),
            Field::from_le_bytes(payloads[1]),
            Field::from_le_bytes(payloads[2]),
            Field::from_le_bytes(payloads[3]),
            Field::from_le_bytes(payloads[4]),
            Field::from_le_bytes(payloads[5]),
            Field::from_le_bytes(payloads[6]),
            Field::from_le_bytes(payloads[7]),
        ];

        context.emit_public_log(msg);
        sequence
    }

    #[private]
    fn publish_message_in_private(
        nonce: u64,
        payloads: [[u8; 31]; 8],
        message_fee: u128,
        consistency: u8,
        from: AztecAddress,
        token_nonce: Field,
    ) {
        assert(1 <= message_fee, "insufficient fee");

        // NOTE: These addresses are hardcoded because private functions cannot access PublicMutable storage.
        // They must be updated to match the deployed contract addresses before compilation.
        let receiver_address = AztecAddress {
            inner: 0x0d071eec273fa0c82825d9c5d2096965a40bcc33ae942714cf6c683af9632504,
        };
        let token_address = AztecAddress {
            inner: 0x037e5d19d6d27e2fb7c947cfe7c36459e27d35e46dd59f5f47373a64ff491d2c,
        };

        let _ = Token::at(token_address)
            .transfer_in_private(from, receiver_address, message_fee, token_nonce)
            .call(&mut context);

        Wormhole::at(context.this_address())._publish_message(nonce, payloads, consistency).enqueue(
            &mut context,
        );
    }

    #[public]
    #[internal]
    fn _publish_message(nonce: u64, payloads: [[u8; 31]; 8], consistency: u8) -> u64 {
        let sequence = storage.sequences.at(context.msg_sender()).read();
        storage.sequences.at(context.msg_sender()).write(sequence + 1);

        let msg: [Field; 13] = [
            context.msg_sender().to_field(),
            sequence as Field,
            nonce as Field,
            consistency as Field,
            context.timestamp() as Field,
            Field::from_le_bytes(payloads[0]),
            Field::from_le_bytes(payloads[1]),
            Field::from_le_bytes(payloads[2]),
            Field::from_le_bytes(payloads[3]),
            Field::from_le_bytes(payloads[4]),
            Field::from_le_bytes(payloads[5]),
            Field::from_le_bytes(payloads[6]),
            Field::from_le_bytes(payloads[7]),
        ];

        context.emit_public_log(msg);
        sequence
    }

    // Keep all your guardian setters exactly as they are
    #[public]
    fn set_guardian1(guardian: Guardian, index: u32) {
        storage.guardian_1.at(index).write(guardian);
    }

    #[public]
    fn set_guardian2(guardian: Guardian, index: u32) {
        storage.guardian_2.at(index).write(guardian);
    }

    #[public]
    fn set_guardian3(guardian: Guardian, index: u32) {
        storage.guardian_3.at(index).write(guardian);
    }

    #[public]
    fn set_guardian4(guardian: Guardian, index: u32) {
        storage.guardian_4.at(index).write(guardian);
    }

    #[public]
    fn set_guardian5(guardian: Guardian, index: u32) {
        storage.guardian_5.at(index).write(guardian);
    }

    #[public]
    fn set_guardian6(guardian: Guardian, index: u32) {
        storage.guardian_6.at(index).write(guardian);
    }

    #[public]
    fn set_guardian7(guardian: Guardian, index: u32) {
        storage.guardian_7.at(index).write(guardian);
    }

    #[public]
    fn set_guardian8(guardian: Guardian, index: u32) {
        storage.guardian_8.at(index).write(guardian);
    }

    #[public]
    fn set_guardian9(guardian: Guardian, index: u32) {
        storage.guardian_9.at(index).write(guardian);
    }

    #[public]
    fn set_guardian10(guardian: Guardian, index: u32) {
        storage.guardian_10.at(index).write(guardian);
    }

    #[public]
    fn set_guardian11(guardian: Guardian, index: u32) {
        storage.guardian_11.at(index).write(guardian);
    }

    #[public]
    fn set_guardian12(guardian: Guardian, index: u32) {
        storage.guardian_12.at(index).write(guardian);
    }

    #[public]
    fn set_guardian13(guardian: Guardian, index: u32) {
        storage.guardian_13.at(index).write(guardian);
    }

    #[public]
    fn set_guardian14(guardian: Guardian, index: u32) {
        storage.guardian_14.at(index).write(guardian);
    }

    #[public]
    fn set_guardian15(guardian: Guardian, index: u32) {
        storage.guardian_15.at(index).write(guardian);
    }

    #[public]
    fn set_guardian16(guardian: Guardian, index: u32) {
        storage.guardian_16.at(index).write(guardian);
    }

    #[public]
    fn set_guardian17(guardian: Guardian, index: u32) {
        storage.guardian_17.at(index).write(guardian);
    }

    #[public]
    fn set_guardian18(guardian: Guardian, index: u32) {
        storage.guardian_18.at(index).write(guardian);
    }

    #[public]
    fn set_guardian19(guardian: Guardian, index: u32) {
        storage.guardian_19.at(index).write(guardian);
    }

    #[public]
    fn expire_guardian_set(index: u32) {
        storage.current_guardian_set_index.at(index).write(context.timestamp() + 86400);
    }

    #[public]
    fn guardian_set_expired(index: u32) -> bool {
        let timestamp = storage.current_guardian_set_index.at(index).read();
        context.timestamp() > timestamp
    }

    #[public]
    fn set_provider(provider: Provider) {
        let updated_storage = WormholeStorage::set_provider(storage.state.read(), provider);
        storage.state.write(updated_storage);
    }

    #[public]
    fn get_provider() -> Provider {
        WormholeStorage::get_provider(storage.state.read())
    }

    #[contract_library_method]
    unconstrained fn verify_vaa_unconstrained(
        bytes: [u8; 2000],
        actual_length: u32,
    ) -> ([[u8; 32]; 13], [[u8; 32]; 13], [[u8; 64]; 13], [u8; 32], u8) {
        let version = bytes[0];
        assert(version == 1, "VM version incompatible");

        let (_, signatures_len, signatures, body_bytes, body_length) =
            parse_vaa_unconstrained(bytes, actual_length);

        assert(signatures_len >= 1, "Need at least 1 signature");
        assert(signatures_len <= 13, "Too many signatures");

        // Compute the VAA hash
        let hash = compute_vaa_hash_unconstrained(body_bytes, body_length);

        let guardian = crate::structs::Guardian::new(
            [
                0x13, 0x94, 0x7b, 0xd4, 0x8b, 0x18, 0xe5, 0x3f, 0xda, 0xee, 0xe7, 0x7f, 0x34, 0x73,
                0x39, 0x1a, 0xc7, 0x27, 0xc6, 0x38,
            ],
            [
                0xfa, 0x9d, 0x6b, 0x47, 0x04, 0x3b, 0x15, 0xb4, 0xb3, 0x3c, 0xf0, 0x5b, 0xb1, 0xde,
                0x0f, 0x13, 0xda, 0x70, 0x31, 0x36, 0xb2, 0xcd, 0x15, 0x73, 0x24, 0xeb, 0x61, 0x5e,
                0xc9, 0xee, 0x95, 0x12,
            ],
            [
                0x38, 0xc6, 0x7b, 0x54, 0xd0, 0x3e, 0x3c, 0x18, 0x2e, 0x0e, 0x77, 0xb8, 0x5f, 0x88,
                0x3c, 0xcd, 0x74, 0xdd, 0xdf, 0xc5, 0x90, 0x10, 0xfb, 0xf4, 0x75, 0xa0, 0xbe, 0xe6,
                0x59, 0x3c, 0xe2, 0xbd,
            ],
        );

        // Prepare arrays for constrained verification
        let mut pub_keys_x = [[0; 32]; 13];
        let mut pub_keys_y = [[0; 32]; 13];
        let mut signature_bytes_array = [[0; 64]; 13];

        for i in 0..13 {
            if i < signatures_len {
                let sig = Signature::from_bytes(signatures[i as u32]);
                let guardian_idx = sig.guardian_index;

                assert(guardian_idx < 19, "guardian index out of bounds");

                // Get public key components as byte arrays
                let pub_key_x = guardian.get_pub_key_x();
                let pub_key_y = guardian.get_pub_key_y();

                pub_keys_x[i as u32] = pub_key_x;
                pub_keys_y[i as u32] = pub_key_y;

                // Prepare signature bytes
                let mut signature_bytes = [0; 64];
                for j in 0..32 {
                    signature_bytes[j] = sig.r[j];
                    signature_bytes[32 + j] = sig.s[j];
                }
                signature_bytes_array[i as u32] = signature_bytes;
            }
        }
        
        (pub_keys_x, pub_keys_y, signature_bytes_array, hash, signatures_len)
    }

    #[private]
    fn verify_vaa(bytes: [u8; 2000], actual_length: u32) {
        // Safety: verify_vaa_unconstrained performs parsing and validation that doesn't need constraints
        let (pub_keys_x, pub_keys_y, signature_bytes_array, hash, signatures_len) =
            unsafe { verify_vaa_unconstrained(bytes, actual_length) };

        let required_signatures = 1;
        let mut verified_count = 0;

        // Testnet VAA verification only utilizes a single guardian.

        // TODO: Uncomment this when ECDSA verification in loop is implemented.
        // See issue: https://github.com/noir-lang/noir/pull/8993
        
        // for i in 0..13 {
        //     if i < signatures_len {
        //        let is_valid = std::ecdsa_secp256k1::verify_signature(
        //            pub_keys_x[i as u32],
        //            pub_keys_y[i as u32],
        //            signature_bytes_array[i as u32],
        //            hash,
        //        );

        //        if is_valid {
        //            verified_count += 1;
        //        }
        //    }
        //}

        let is_valid = std::ecdsa_secp256k1::verify_signature(
            pub_keys_x[0 as u32],
            pub_keys_y[0 as u32],
            signature_bytes_array[0 as u32],
            hash,
        );

        if is_valid {
            verified_count += 1;
        }

        assert(verified_count >= required_signatures, "Insufficient valid signatures");
    }
}

mod tests {
    use crate::{structs::Signature, Wormhole};
    use dep::aztec::prelude::AztecAddress;
    use dep::aztec::protocol_types::traits::FromField;
    use dep::aztec::test::helpers::cheatcodes;
    use dep::aztec::test::helpers::test_environment::TestEnvironment;

    #[test]
    unconstrained fn test_signature_extraction() {
        let signature_bytes: [u8; 66] = [
            0x00, // Guardian index
            // R component (32 bytes)
            0x46, 0x82, 0xbc, 0x4d, 0x5f, 0xf2, 0xe5, 0x4d, 0xc2, 0xee, 0x5e, 0x0e, 0xb6, 0x4f,
            0x5c, 0x6c, 0x07, 0xaa, 0x44, 0x9a, 0xc5, 0x39, 0xab, 0xc6, 0x3c, 0x2b, 0xe5, 0xc3,
            0x06, 0xa4, 0x8f, 0x23,
            // S component (32 bytes)
            0x3e, 0x93, 0x00, 0x17, 0x0a, 0x82, 0xad, 0xf3, 0xc3, 0xb7, 0xf4, 0x3f, 0x23, 0x17,
            0x6f, 0xb0, 0x79, 0x17, 0x4a, 0x58, 0xd6, 0x7d, 0x14, 0x24, 0x77, 0xf6, 0x46, 0x67,
            0x5d, 0x86, 0xeb, 0x63,
            // Recovery ID
            0x01,
        ];

        let sig = Signature::from_bytes(signature_bytes);

        assert(sig.guardian_index == 0, "Guardian index should be 0");
        assert(sig.r[0] == 0x46, "R component should start with 0x46");
        assert(sig.s[0] == 0x3e, "S component should start with 0x3e");
    }

    #[test]
    unconstrained fn test_verify_vaa_logic() {
        // Setup env, generate keys
        let mut env = TestEnvironment::new();

        cheatcodes::reset(); // Reset TXE state

        // Deploy contract - need to specify all 3 parameters for deploy
        let dummy_owner = AztecAddress::from_field(1);
        let receiver_address = AztecAddress::from_field(
            0x0d071eec273fa0c82825d9c5d2096965a40bcc33ae942714cf6c683af9632504,
        );
        let token_address = AztecAddress::from_field(
            0x037e5d19d6d27e2fb7c947cfe7c36459e27d35e46dd59f5f47373a64ff491d2c,
        );

        let contract_address = env
            .deploy_self("Wormhole")
            .with_public_void_initializer(
                dummy_owner,
                Wormhole::interface().init(56, 56, receiver_address, token_address),
            )
            .to_address();

        let mut vaa_bytes: [u8; 2000] = [0; 2000];
        let actual_length = 221;

        // NOTE: This is a real Wormhole VAA from Arbitrum Sepolia testnet.
        // VAA bytes, fetched from https://wormholescan.io/#/tx/0xf93fd41efeb09ff28174824d4abf6dbc06ac408953a9975aa4a403d434051efc
        // This VAA contains the message "Hello Wormhole!" and is used to test VAA verification logic.
        let wh_vaa: [u8; 221] = [
            0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x46, 0x82, 0xbc, 0x4d, 0x5f, 0xf2, 0xe5,
            0x4d, 0xc2, 0xee, 0x5e, 0x0e, 0xb6, 0x4f, 0x5c, 0x6c, 0x07, 0xaa, 0x44, 0x9a, 0xc5,
            0x39, 0xab, 0xc6, 0x3c, 0x2b, 0xe5, 0xc3, 0x06, 0xa4, 0x8f, 0x23, 0x3e, 0x93, 0x00,
            0x17, 0x0a, 0x82, 0xad, 0xf3, 0xc3, 0xb7, 0xf4, 0x3f, 0x23, 0x17, 0x6f, 0xb0, 0x79,
            0x17, 0x4a, 0x58, 0xd6, 0x7d, 0x14, 0x24, 0x77, 0xf6, 0x46, 0x67, 0x5d, 0x86, 0xeb,
            0x63, 0x01, 0x68, 0x4b, 0xfa, 0xd4, 0x49, 0x96, 0x02, 0xd2, 0x27, 0x13, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x69, 0x7f, 0x31, 0xe0,
            0x74, 0xbf, 0x2c, 0x81, 0x93, 0x91, 0xd5, 0x27, 0x29, 0xf9, 0x55, 0x06, 0xe0, 0xa7,
            0x2f, 0xfb, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xc8, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x0e, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x57, 0x6f,
            0x72, 0x6d, 0x68, 0x6f, 0x6c, 0x65, 0x21, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        ];

        for i in 0..221 {
            vaa_bytes[i] = wh_vaa[i];
        }

        let _ = env.call_private_void(
            dummy_owner,
            Wormhole::at(contract_address).verify_vaa(vaa_bytes, actual_length),
        );
    }
}
