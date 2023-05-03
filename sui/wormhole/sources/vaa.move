// SPDX-License-Identifier: Apache 2

/// This module implements a mechanism to parse and verify VAAs, which are
/// verified Wormhole messages (messages with Guardian signatures attesting to
/// its observation). Signatures on VAA are checked against an existing Guardian
/// set that exists in the `State` (see `wormhole::state`).
///
/// A Wormhole integrator is discouraged from integrating `parse_and_verify` in
/// his contract. If there is a breaking change to the `vaa` module, Wormhole
/// will be upgraded to prevent previous build versions of this module to work.
/// If an integrator happened to use `parse_and_verify` in his contract, he will
/// need to be prepared to upgrade his contract to take the change (by building
/// with the latest package implementation).
///
/// Instead, an integrator is encouraged to execute a transaction block, which
/// executes `parse_and_verify` from the latest Wormhole package ID and to
/// implement his methods that require redeeming a VAA to take `VAA` as an
/// argument.
///
/// A good example of how this methodology is implemented is how the Token
/// Bridge contract redeems its VAAs.
module wormhole::vaa {
    use std::option::{Self};
    use std::vector::{Self};
    use sui::clock::{Clock};
    use sui::hash::{keccak256};

    use wormhole::bytes::{Self};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::consumed_vaas::{Self, ConsumedVAAs};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::guardian::{Self};
    use wormhole::guardian_set::{Self, GuardianSet};
    use wormhole::guardian_signature::{Self, GuardianSignature};
    use wormhole::state::{Self, State};

    /// Incorrect VAA version.
    const E_WRONG_VERSION: u64 = 0;
    /// Not enough guardians attested to this Wormhole observation.
    const E_NO_QUORUM: u64 = 1;
    /// Signature does not match expected Guardian public key.
    const E_INVALID_SIGNATURE: u64 = 2;
    /// Prior guardian set is no longer valid.
    const E_GUARDIAN_SET_EXPIRED: u64 = 3;
    /// Guardian signature is encoded out of sequence.
    const E_NON_INCREASING_SIGNERS: u64 = 4;

    const VERSION_VAA: u8 = 1;

    /// Container storing verified Wormhole message info. This struct also
    /// caches the digest, which is a double Keccak256 hash of the message body.
    struct VAA {
        /// Guardian set index of Guardians that attested to observing the
        /// Wormhole message.
        guardian_set_index: u32,
        /// Time when Wormhole message was emitted or observed.
        timestamp: u32,
        /// A.K.A. Batch ID.
        nonce: u32,
        /// Wormhole chain ID from which network the message originated from.
        emitter_chain: u16,
        /// Address of contract (standardized to 32 bytes) that produced the
        /// message.
        emitter_address: ExternalAddress,
        /// Sequence number of emitter's Wormhole message.
        sequence: u64,
        /// A.K.A. Finality.
        consistency_level: u8,
        /// Arbitrary payload encoding data relevant to receiver.
        payload: vector<u8>,

        /// Double Keccak256 hash of message body.
        digest: Bytes32
    }

    public fun guardian_set_index(self: &VAA): u32 {
        self.guardian_set_index
    }

    public fun timestamp(self: &VAA): u32 {
         self.timestamp
    }

    public fun nonce(self: &VAA): u32 {
        self.nonce
    }

    public fun batch_id(self: &VAA): u32 {
        nonce(self)
    }

    public fun payload(self: &VAA): vector<u8> {
         self.payload
    }

    public fun digest(self: &VAA): Bytes32 {
         self.digest
    }

    public fun emitter_chain(self: &VAA): u16 {
         self.emitter_chain
    }

    public fun emitter_address(self: &VAA): ExternalAddress {
         self.emitter_address
    }

    public fun emitter_info(self: &VAA): (u16, ExternalAddress, u64) {
        (self.emitter_chain, self.emitter_address, self.sequence)
    }

    public fun sequence(self: &VAA): u64 {
         self.sequence
    }

    public fun consistency_level(self: &VAA): u8 {
        self.consistency_level
    }

    public fun finality(self: &VAA): u8 {
        consistency_level(self)
    }

    /// Destroy the `VAA` and take the Wormhole message payload.
    public fun take_payload(vaa: VAA): vector<u8> {
        let (_, _, payload) = take_emitter_info_and_payload(vaa);

        payload
    }

    /// Destroy the `VAA` and take emitter info (chain and address) and Wormhole
    /// message payload.
    public fun take_emitter_info_and_payload(
        vaa: VAA
    ): (u16, ExternalAddress, vector<u8>) {
        let VAA {
            guardian_set_index: _,
            timestamp: _,
            nonce: _,
            emitter_chain,
            emitter_address,
            sequence: _,
            consistency_level: _,
            digest: _,
            payload,
        } = vaa;
        (emitter_chain, emitter_address, payload)
    }

    /// Parses and verifies the signatures of a VAA.
    ///
    /// NOTE: This is the only public function that returns a VAA, and it should
    /// be kept that way. This ensures that if an external module receives a
    /// `VAA`, it has been verified.
    public fun parse_and_verify(
        wormhole_state: &State,
        buf: vector<u8>,
        the_clock: &Clock
    ): VAA {
        state::assert_latest_only(wormhole_state);

        // Deserialize VAA buffer (and return `VAA` after verifying signatures).
        let (signatures, vaa) = parse(buf);

        // Fetch the guardian set which this VAA was supposedly signed with and
        // verify signatures using guardian set.
        verify_signatures(
            state::guardian_set_at(
                wormhole_state,
                vaa.guardian_set_index
            ),
            signatures,
            bytes32::to_bytes(compute_message_hash(&vaa)),
            the_clock
        );

        // Done.
        vaa
    }

    public fun consume(consumed: &mut ConsumedVAAs, parsed: &VAA) {
        consumed_vaas::consume(consumed, digest(parsed))
    }

    public fun compute_message_hash(parsed: &VAA): Bytes32 {
        let buf = vector::empty();

        bytes::push_u32_be(&mut buf, parsed.timestamp);
        bytes::push_u32_be(&mut buf, parsed.nonce);
        bytes::push_u16_be(&mut buf, parsed.emitter_chain);
        vector::append(
            &mut buf,
            external_address::to_bytes(parsed.emitter_address)
        );
        bytes::push_u64_be(&mut buf, parsed.sequence);
        bytes::push_u8(&mut buf, parsed.consistency_level);
        vector::append(&mut buf, parsed.payload);

        // Return hash.
        bytes32::new(keccak256(&buf))
    }

    /// Parses a VAA.
    ///
    /// NOTE: This method does NOT perform any verification. This ensures the
    /// invariant that if an external module receives a `VAA` object, its
    /// signatures must have been verified, because the only public function
    /// that returns a `VAA` is `parse_and_verify`.
    fun parse(buf: vector<u8>): (vector<GuardianSignature>, VAA) {
        let cur = cursor::new(buf);

        // Check VAA version.
        assert!(
            bytes::take_u8(&mut cur) == VERSION_VAA,
            E_WRONG_VERSION
        );

        let guardian_set_index = bytes::take_u32_be(&mut cur);

        // Deserialize guardian signatures.
        let num_signatures = bytes::take_u8(&mut cur);
        let signatures = vector::empty();
        let i = 0;
        while (i < num_signatures) {
            let guardian_index = bytes::take_u8(&mut cur);
            let r = bytes32::take_bytes(&mut cur);
            let s = bytes32::take_bytes(&mut cur);
            let recovery_id = bytes::take_u8(&mut cur);
            vector::push_back(
                &mut signatures,
                guardian_signature::new(r, s, recovery_id, guardian_index)
            );
            i = i + 1;
        };

        // Deserialize message body.
        let body_buf = cursor::take_rest(cur);

        let cur = cursor::new(body_buf);
        let timestamp = bytes::take_u32_be(&mut cur);
        let nonce = bytes::take_u32_be(&mut cur);
        let emitter_chain = bytes::take_u16_be(&mut cur);
        let emitter_address = external_address::take_bytes(&mut cur);
        let sequence = bytes::take_u64_be(&mut cur);
        let consistency_level = bytes::take_u8(&mut cur);
        let payload = cursor::take_rest(cur);

        let parsed = VAA {
            guardian_set_index,
            timestamp,
            nonce,
            emitter_chain,
            emitter_address,
            sequence,
            consistency_level,
            digest: double_keccak256(body_buf),
            payload,
        };

        (signatures, parsed)
    }

    fun double_keccak256(buf: vector<u8>): Bytes32 {
        use sui::hash::{keccak256};

        bytes32::new(keccak256(&keccak256(&buf)))
    }

    /// Using the Guardian signatures deserialized from VAA, verify that all of
    /// the Guardian public keys are recovered using these signatures and the
    /// VAA message body as the message used to produce these signatures.
    ///
    /// We are careful to only allow `wormhole:vaa` to control the hash that
    /// gets used in the `ecdsa_k1` module by computing the hash after
    /// deserializing the VAA message body. Even though `ecdsa_k1` hashes a
    /// raw message (as of version 0.28), the "raw message" in this case is a
    /// single keccak256 hash of the VAA message body.
    fun verify_signatures(
        set: &GuardianSet,
        signatures: vector<GuardianSignature>,
        message_hash: vector<u8>,
        the_clock: &Clock
    ) {
        // Guardian set must be active (not expired).
        assert!(
            guardian_set::is_active(set, the_clock),
            E_GUARDIAN_SET_EXPIRED
        );

        // Number of signatures must be at least quorum.
        assert!(
            vector::length(&signatures) >= guardian_set::quorum(set),
            E_NO_QUORUM
        );

        // Drain `Cursor` by checking each signature.
        let cur = cursor::new(signatures);
        let last_guardian_index = option::none();
        while (!cursor::is_empty(&cur)) {
            let signature = cursor::poke(&mut cur);
            let guardian_index = guardian_signature::index_as_u64(&signature);

            // Ensure that the provided signatures are strictly increasing.
            // This check makes sure that no duplicate signers occur. The
            // increasing order is guaranteed by the guardians, or can always be
            // reordered by the client.
            assert!(
                (
                    option::is_none(&last_guardian_index) ||
                    guardian_index > *option::borrow(&last_guardian_index)
                ),
                E_NON_INCREASING_SIGNERS
            );

            // If the guardian pubkey cannot be recovered using the signature
            // and message hash, revert.
            assert!(
                guardian::verify(
                    guardian_set::guardian_at(set, guardian_index),
                    signature,
                    message_hash
                ),
                E_INVALID_SIGNATURE
            );

            // Continue.
            option::swap_or_fill(&mut last_guardian_index, guardian_index);
        };

        // Done.
        cursor::destroy_empty(cur);
    }

    #[test_only]
    public fun parse_test_only(
        buf: vector<u8>
    ): (vector<GuardianSignature>, VAA) {
        parse(buf)
    }

    #[test_only]
    public fun destroy(vaa: VAA) {
        take_payload(vaa);
    }

    #[test_only]
    public fun peel_payload_from_vaa(buf: &vector<u8>): vector<u8> {
        // Just make sure that we are passing version 1 VAAs to this method.
        assert!(*vector::borrow(buf, 0) == VERSION_VAA, E_WRONG_VERSION);

        // Find the location of the payload.
        let num_signatures = (*vector::borrow(buf, 5) as u64);
        let i = 57 + num_signatures * 66;

        // Push the payload bytes to `out` and return.
        let out = vector::empty();
        let len = vector::length(buf);
        while (i < len) {
            vector::push_back(&mut out, *vector::borrow(buf, i));
            i = i + 1;
        };

        // Return the payload.
        out
    }
}

#[test_only]
module wormhole::vaa_tests {
    use std::vector::{Self};
    use sui::test_scenario::{Self};

    use wormhole::bytes32::{Self};
    use wormhole::cursor::{Self};
    use wormhole::external_address::{Self};
    use wormhole::guardian_signature::{Self};
    use wormhole::state::{Self};
    use wormhole::vaa::{Self};
    use wormhole::version_control::{Self};
    use wormhole::wormhole_scenario::{
        guardians,
        person,
        return_clock,
        return_state,
        set_up_wormhole_with_guardians,
        take_clock,
        take_state
        //upgrade_wormhole
    };

    const VAA_1: vector<u8> =
        x"01000000000d009bafff633087a9587d9afb6d29bd74a3483b7a8d5619323a416fe9ca43b482cd5526fabe953157cfd42eea9ffa544babc0f3a025a8a6159217b96fc9ff586d560002c9367884940a43a1a4d86531ea33ccb41e52bd7d1679c106fdff756f3da2ca743f7c181fcf40d19151d0a8397335c1b71709279b6e4fa97b6e3de90824e841c801035a493b65bf12ab9b98aa4db3bfcb73df20ab854d8e5998a1552f3b3e57ea7cd3546187c62cd450d12d430cae0fb48124ae68034dae602fa3e2232b55257961f90104758e265101353923661f6df67cec3c38528ed1b68825099b5bb2ce3fb2e735c5073d90223bebd00cc10406a60413a6089b5fb9acee0a1b04a63a8d7db24c0bbc000587777306dd174e266c313f711e881086355b6ce66cf2bf1f5da58273a10be77813b5ffcafc1ba6b83645e326a7c1a3751496f279ba307a6cd554f2709c2f1eda0108ed23ba8264146c3e3cc0601c93260c25058bcdd25213a7834e51679afdc4b50104e3f3a3079ba45115e703096c7e0700354cd48348bbf686dcbc58be896c35a20009c2352cb46ef1d2ef9e185764650403aee87a1be071555b31cdcee0c346991da858defb8d5e164a293ce4377b54fc74b65e3acbdedcbb53c2bcc2688a0b5bd1c9010ae470b1573989f387f7c54a86325cc05978bbcbc13267e90e2fa2efb0e18bccb772252bd6d13ebf908f7f3f2caf20a45c17dec7168122a2535ea93d300fae7063000ba0e8770298d4e3567488f455455a33f1e723e1e629ba4f87928016aeaa5875561ec38bde5d934389dc657d80a927cd9d06a9d9c7ce910c98d77a576e3f31735c000eeeedc956cff4489ac55b52ca38233cdc11e88767e5cc82f664bd1d7c28dfb5a12d7d17620725aae08e499b021200919f42c50c05916cf425dcd6e59f24b4b233000f18d447c9608a076c066b30ee770910e3c133087d33e329ad0101f08f88d88e142623df87aa3842edcf34e10fd36271b49f7af73ff2a7bcf4a65a4306d59586f20111905fc99dc650d9b1b33c313e9b31dfdbc85ce57e9f31abc4841d5791a239f20e5f28e4e612db96aee2f49ae712f724466007aaf27309d0385005fe0264d33dd100127b46f2fbbbf12efb10c2e662b4449de404f6a408ad7f38c7ea40a46300930e9a3b1e02ce00b97e33fa8a87221c1fd9064ce966dc4772658b98f2ec1e28d13e7400000023280000000c002adeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef000000000000092a20416c6c20796f75722062617365206172652062656c6f6e6720746f207573";
    const VAA_DOUBLE_SIGNED: vector<u8> =
        x"01000000000d009bafff633087a9587d9afb6d29bd74a3483b7a8d5619323a416fe9ca43b482cd5526fabe953157cfd42eea9ffa544babc0f3a025a8a6159217b96fc9ff586d560002c9367884940a43a1a4d86531ea33ccb41e52bd7d1679c106fdff756f3da2ca743f7c181fcf40d19151d0a8397335c1b71709279b6e4fa97b6e3de90824e841c80102c9367884940a43a1a4d86531ea33ccb41e52bd7d1679c106fdff756f3da2ca743f7c181fcf40d19151d0a8397335c1b71709279b6e4fa97b6e3de90824e841c801035a493b65bf12ab9b98aa4db3bfcb73df20ab854d8e5998a1552f3b3e57ea7cd3546187c62cd450d12d430cae0fb48124ae68034dae602fa3e2232b55257961f90104758e265101353923661f6df67cec3c38528ed1b68825099b5bb2ce3fb2e735c5073d90223bebd00cc10406a60413a6089b5fb9acee0a1b04a63a8d7db24c0bbc000587777306dd174e266c313f711e881086355b6ce66cf2bf1f5da58273a10be77813b5ffcafc1ba6b83645e326a7c1a3751496f279ba307a6cd554f2709c2f1eda0108ed23ba8264146c3e3cc0601c93260c25058bcdd25213a7834e51679afdc4b50104e3f3a3079ba45115e703096c7e0700354cd48348bbf686dcbc58be896c35a20009c2352cb46ef1d2ef9e185764650403aee87a1be071555b31cdcee0c346991da858defb8d5e164a293ce4377b54fc74b65e3acbdedcbb53c2bcc2688a0b5bd1c9010ae470b1573989f387f7c54a86325cc05978bbcbc13267e90e2fa2efb0e18bccb772252bd6d13ebf908f7f3f2caf20a45c17dec7168122a2535ea93d300fae7063000ba0e8770298d4e3567488f455455a33f1e723e1e629ba4f87928016aeaa5875561ec38bde5d934389dc657d80a927cd9d06a9d9c7ce910c98d77a576e3f31735c000f18d447c9608a076c066b30ee770910e3c133087d33e329ad0101f08f88d88e142623df87aa3842edcf34e10fd36271b49f7af73ff2a7bcf4a65a4306d59586f20111905fc99dc650d9b1b33c313e9b31dfdbc85ce57e9f31abc4841d5791a239f20e5f28e4e612db96aee2f49ae712f724466007aaf27309d0385005fe0264d33dd100127b46f2fbbbf12efb10c2e662b4449de404f6a408ad7f38c7ea40a46300930e9a3b1e02ce00b97e33fa8a87221c1fd9064ce966dc4772658b98f2ec1e28d13e7400000023280000000c002adeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef000000000000092a20416c6c20796f75722062617365206172652062656c6f6e6720746f207573";
    const VAA_NO_QUORUM: vector<u8> =
        x"01000000000c009bafff633087a9587d9afb6d29bd74a3483b7a8d5619323a416fe9ca43b482cd5526fabe953157cfd42eea9ffa544babc0f3a025a8a6159217b96fc9ff586d560002c9367884940a43a1a4d86531ea33ccb41e52bd7d1679c106fdff756f3da2ca743f7c181fcf40d19151d0a8397335c1b71709279b6e4fa97b6e3de90824e841c801035a493b65bf12ab9b98aa4db3bfcb73df20ab854d8e5998a1552f3b3e57ea7cd3546187c62cd450d12d430cae0fb48124ae68034dae602fa3e2232b55257961f90104758e265101353923661f6df67cec3c38528ed1b68825099b5bb2ce3fb2e735c5073d90223bebd00cc10406a60413a6089b5fb9acee0a1b04a63a8d7db24c0bbc000587777306dd174e266c313f711e881086355b6ce66cf2bf1f5da58273a10be77813b5ffcafc1ba6b83645e326a7c1a3751496f279ba307a6cd554f2709c2f1eda0108ed23ba8264146c3e3cc0601c93260c25058bcdd25213a7834e51679afdc4b50104e3f3a3079ba45115e703096c7e0700354cd48348bbf686dcbc58be896c35a20009c2352cb46ef1d2ef9e185764650403aee87a1be071555b31cdcee0c346991da858defb8d5e164a293ce4377b54fc74b65e3acbdedcbb53c2bcc2688a0b5bd1c9010ae470b1573989f387f7c54a86325cc05978bbcbc13267e90e2fa2efb0e18bccb772252bd6d13ebf908f7f3f2caf20a45c17dec7168122a2535ea93d300fae7063000ba0e8770298d4e3567488f455455a33f1e723e1e629ba4f87928016aeaa5875561ec38bde5d934389dc657d80a927cd9d06a9d9c7ce910c98d77a576e3f31735c000f18d447c9608a076c066b30ee770910e3c133087d33e329ad0101f08f88d88e142623df87aa3842edcf34e10fd36271b49f7af73ff2a7bcf4a65a4306d59586f20111905fc99dc650d9b1b33c313e9b31dfdbc85ce57e9f31abc4841d5791a239f20e5f28e4e612db96aee2f49ae712f724466007aaf27309d0385005fe0264d33dd100127b46f2fbbbf12efb10c2e662b4449de404f6a408ad7f38c7ea40a46300930e9a3b1e02ce00b97e33fa8a87221c1fd9064ce966dc4772658b98f2ec1e28d13e7400000023280000000c002adeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef000000000000092a20416c6c20796f75722062617365206172652062656c6f6e6720746f207573";

    #[test]
    fun test_parse() {
        let (signatures, parsed) = vaa::parse_test_only(VAA_1);

        let expected_signatures =
            vector[
                guardian_signature::new(
                    bytes32::new(
                        x"9bafff633087a9587d9afb6d29bd74a3483b7a8d5619323a416fe9ca43b482cd"
                    ), // r
                    bytes32::new(
                        x"5526fabe953157cfd42eea9ffa544babc0f3a025a8a6159217b96fc9ff586d56"
                    ), // s
                    0, // recovery_id
                    0 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"c9367884940a43a1a4d86531ea33ccb41e52bd7d1679c106fdff756f3da2ca74"
                    ), // r
                    bytes32::new(
                        x"3f7c181fcf40d19151d0a8397335c1b71709279b6e4fa97b6e3de90824e841c8"
                    ), // s
                    1, // recovery_id
                    2 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"5a493b65bf12ab9b98aa4db3bfcb73df20ab854d8e5998a1552f3b3e57ea7cd3"
                    ), // r
                    bytes32::new(
                        x"546187c62cd450d12d430cae0fb48124ae68034dae602fa3e2232b55257961f9"
                    ), // s
                    1, // recovery_id
                    3 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"758e265101353923661f6df67cec3c38528ed1b68825099b5bb2ce3fb2e735c5"
                    ), // r
                    bytes32::new(
                        x"073d90223bebd00cc10406a60413a6089b5fb9acee0a1b04a63a8d7db24c0bbc"
                    ), // s
                    0, // recovery_id
                    4 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"87777306dd174e266c313f711e881086355b6ce66cf2bf1f5da58273a10be778"
                    ), // r
                    bytes32::new(
                        x"13b5ffcafc1ba6b83645e326a7c1a3751496f279ba307a6cd554f2709c2f1eda"
                    ), // s
                    1, // recovery_id
                    5 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"ed23ba8264146c3e3cc0601c93260c25058bcdd25213a7834e51679afdc4b501"
                    ), // r
                    bytes32::new(
                        x"04e3f3a3079ba45115e703096c7e0700354cd48348bbf686dcbc58be896c35a2"
                    ), // s
                    0, // recovery_id
                    8 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"c2352cb46ef1d2ef9e185764650403aee87a1be071555b31cdcee0c346991da8"
                    ), // r
                    bytes32::new(
                        x"58defb8d5e164a293ce4377b54fc74b65e3acbdedcbb53c2bcc2688a0b5bd1c9"
                    ), // s
                    1, // recovery_id
                    9 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"e470b1573989f387f7c54a86325cc05978bbcbc13267e90e2fa2efb0e18bccb7"
                    ), // r
                    bytes32::new(
                        x"72252bd6d13ebf908f7f3f2caf20a45c17dec7168122a2535ea93d300fae7063"
                    ), // s
                    0, // recovery_id
                    10 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"a0e8770298d4e3567488f455455a33f1e723e1e629ba4f87928016aeaa587556"
                    ), // r
                    bytes32::new(
                        x"1ec38bde5d934389dc657d80a927cd9d06a9d9c7ce910c98d77a576e3f31735c"
                    ), // s
                    0, // recovery_id
                    11 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"eeedc956cff4489ac55b52ca38233cdc11e88767e5cc82f664bd1d7c28dfb5a1"
                    ), // r
                    bytes32::new(
                        x"2d7d17620725aae08e499b021200919f42c50c05916cf425dcd6e59f24b4b233"
                    ), // s
                    0, // recovery_id
                    14 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"18d447c9608a076c066b30ee770910e3c133087d33e329ad0101f08f88d88e14"
                    ), // r
                    bytes32::new(
                        x"2623df87aa3842edcf34e10fd36271b49f7af73ff2a7bcf4a65a4306d59586f2"
                    ), // s
                    1, // recovery_id
                    15 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"905fc99dc650d9b1b33c313e9b31dfdbc85ce57e9f31abc4841d5791a239f20e"
                    ), // r
                    bytes32::new(
                        x"5f28e4e612db96aee2f49ae712f724466007aaf27309d0385005fe0264d33dd1"
                    ), // s
                    0, // recovery_id
                    17 // index
                ),
                guardian_signature::new(
                    bytes32::new(
                        x"7b46f2fbbbf12efb10c2e662b4449de404f6a408ad7f38c7ea40a46300930e9a"
                    ), // r
                    bytes32::new(
                        x"3b1e02ce00b97e33fa8a87221c1fd9064ce966dc4772658b98f2ec1e28d13e74"
                    ), // s
                    0, // recovery_id
                    18 // index
                )
            ];
        assert!(
            vector::length(&signatures) == vector::length(&expected_signatures),
            0
        );
        let left = cursor::new(signatures);
        let right = cursor::new(expected_signatures);
        while (!cursor::is_empty(&left)) {
            assert!(cursor::poke(&mut left) == cursor::poke(&mut right), 0);
        };
        cursor::destroy_empty(left);
        cursor::destroy_empty(right);

        assert!(vaa::guardian_set_index(&parsed) == 0, 0);
        assert!(vaa::timestamp(&parsed) == 9000, 0);

        let expected_batch_id = 12;
        assert!(vaa::batch_id(&parsed) == expected_batch_id, 0);
        assert!(vaa::nonce(&parsed) == expected_batch_id, 0);

        assert!(vaa::emitter_chain(&parsed) == 42, 0);

        let expected_emitter_address =
            external_address::from_address(
                @0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef
            );
        assert!(vaa::emitter_address(&parsed) == expected_emitter_address, 0);
        assert!(vaa::sequence(&parsed) == 2346, 0);

        let expected_finality = 32;
        assert!(vaa::finality(&parsed) == expected_finality, 0);
        assert!(vaa::consistency_level(&parsed) == expected_finality, 0);

        // The message Wormhole guardians sign is a hash of the actual message
        // body. So the hash we need to check against is keccak256 of this
        // message.
        let body_buf = {
            use wormhole::bytes::{Self};

            let buf = vector::empty();
            bytes::push_u32_be(&mut buf, vaa::timestamp(&parsed));
            bytes::push_u32_be(&mut buf, vaa::nonce(&parsed));
            bytes::push_u16_be(&mut buf, vaa::emitter_chain(&parsed));
            vector::append(
                &mut buf,
                external_address::to_bytes(vaa::emitter_address(&parsed))
            );
            bytes::push_u64_be(&mut buf, vaa::sequence(&parsed));
            bytes::push_u8(&mut buf, vaa::consistency_level(&parsed));
            vector::append(&mut buf, vaa::payload(&parsed));

            buf
        };

        let expected_message_hash =
            bytes32::new(sui::hash::keccak256(&body_buf));
        assert!(vaa::compute_message_hash(&parsed) == expected_message_hash, 0);

        let expected_digest =
            bytes32::new(
                sui::hash::keccak256(&sui::hash::keccak256(&body_buf))
            );
        assert!(vaa::digest(&parsed) == expected_digest, 0);

        assert!(
            vaa::take_payload(parsed) == b"All your base are belong to us",
            0
        );
    }

    #[test]
    fun test_parse_and_verify() {
        // Testing this method.
        use wormhole::vaa::{parse_and_verify};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Initialize Wormhole with 19 guardians.
        let wormhole_fee = 350;
        set_up_wormhole_with_guardians(scenario, wormhole_fee, guardians());

        // Prepare test to execute `parse_and_verify`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        let verified_vaa = parse_and_verify(&worm_state, VAA_1, &the_clock);

        // We verified all parsed output in `test_parse`. But in destroying the
        // parsed VAA, we will check the payload for the heck of it.
        assert!(
            vaa::take_payload(verified_vaa) == b"All your base are belong to us",
            0
        );

        // Clean up.
        return_state(worm_state);
        return_clock(the_clock);

        // Done.
        test_scenario::end(my_scenario);
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_NO_QUORUM)]
    fun test_cannot_parse_and_verify_without_quorum() {
        // Testing this method.
        use wormhole::vaa::{parse_and_verify};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Initialize Wormhole with 19 guardians.
        let wormhole_fee = 350;
        set_up_wormhole_with_guardians(scenario, wormhole_fee, guardians());

        // Prepare test to execute `parse_and_verify`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // You shall not pass!
        let verified_vaa = parse_and_verify(&worm_state, VAA_NO_QUORUM, &the_clock);

        // Clean up.
        vaa::destroy(verified_vaa);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_NON_INCREASING_SIGNERS)]
    fun test_cannot_parse_and_verify_non_increasing() {
        // Testing this method.
        use wormhole::vaa::{parse_and_verify};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Initialize Wormhole with 19 guardians.
        let wormhole_fee = 350;
        set_up_wormhole_with_guardians(scenario, wormhole_fee, guardians());

        // Prepare test to execute `parse_and_verify`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // You shall not pass!
        let verified_vaa =
            parse_and_verify(&worm_state, VAA_DOUBLE_SIGNED, &the_clock);

        // Clean up.
        vaa::destroy(verified_vaa);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = vaa::E_INVALID_SIGNATURE)]
    fun test_cannot_parse_and_verify_invalid_signature() {
        // Testing this method.
        use wormhole::vaa::{parse_and_verify};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Initialize Wormhole with 19 guardians. But reverse the order so the
        // signatures will not match.
        let initial_guardians = guardians();
        std::vector::reverse(&mut initial_guardians);

        let wormhole_fee = 350;
        set_up_wormhole_with_guardians(
            scenario,
            wormhole_fee,
            initial_guardians
        );

        // Prepare test to execute `parse_and_verify`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // You shall not pass!
        let verified_vaa = parse_and_verify(&worm_state, VAA_1, &the_clock);

        // Clean up.
        vaa::destroy(verified_vaa);

        abort 42
    }

    #[test]
    #[expected_failure(abort_code = wormhole::package_utils::E_NOT_CURRENT_VERSION)]
    fun test_cannot_parse_and_verify_outdated_version() {
        // Testing this method.
        use wormhole::vaa::{parse_and_verify};

        // Set up.
        let caller = person();
        let my_scenario = test_scenario::begin(caller);
        let scenario = &mut my_scenario;

        // Initialize Wormhole with 19 guardians.
        let wormhole_fee = 350;
        set_up_wormhole_with_guardians(scenario, wormhole_fee, guardians());

        // Prepare test to execute `parse_and_verify`.
        test_scenario::next_tx(scenario, caller);

        let worm_state = take_state(scenario);
        let the_clock = take_clock(scenario);

        // Conveniently roll version back.
        state::reverse_migrate_version(&mut worm_state);

        // Simulate executing with an outdated build by upticking the minimum
        // required version for `publish_message` to something greater than
        // this build.
        state::migrate_version_test_only(
            &mut worm_state,
            version_control::previous_version_test_only(),
            version_control::next_version()
        );

        // You shall not pass!
        let verified_vaa = parse_and_verify(&worm_state, VAA_1, &the_clock);

        // Clean up.
        vaa::destroy(verified_vaa);

        abort 42
    }
}
