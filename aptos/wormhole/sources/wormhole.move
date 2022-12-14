module wormhole::wormhole {
    use std::vector;
    use aptos_framework::account;
    use aptos_framework::coin::{Self, Coin};
    use aptos_framework::aptos_coin::{AptosCoin};
    use wormhole::structs::{create_guardian, create_guardian_set, Guardian};
    use wormhole::state;
    use deployer::deployer;
    use wormhole::u16;
    use wormhole::u32::{Self, U32};
    use wormhole::emitter;
    use wormhole::external_address::{Self};
    use std::signer;

    const E_INSUFFICIENT_FEE: u64 = 0;

// -----------------------------------------------------------------------------
// Sending messages

    public fun publish_message(
        emitter_cap: &mut emitter::EmitterCapability,
        nonce: u64,
        payload: vector<u8>,
        message_fee: Coin<AptosCoin>
    ): u64 {
        // ensure that provided fee is sufficient to cover message fees
        let expected_fee = state::get_message_fee();
        assert!(expected_fee <= coin::value(&message_fee), E_INSUFFICIENT_FEE);
        // deposit the fees into the wormhole account
        coin::deposit(@wormhole, message_fee);
        let sequence = emitter::use_sequence(emitter_cap);
        state::publish_event(
            emitter::get_emitter(emitter_cap),
            sequence,
            nonce,
            payload,
        );
        sequence
    }

// -----------------------------------------------------------------------------
// Emitter registration

    public fun register_emitter(): emitter::EmitterCapability {
        state::new_emitter()
    }

// -----------------------------------------------------------------------------
// Contract initialization

    /// Initializes the contract. Note that this function takes additional
    /// arguments, so the native `init_module` function (which takes no
    /// arguments) cannot be used.
    /// Can only be called by the deployer (checked by the
    /// `deployer::claim_signer_capability` function).
    public entry fun init(
        deployer: &signer,
        chain_id: u64,
        governance_chain_id: u64,
        governance_contract: vector<u8>,
        initial_guardians: vector<vector<u8>>
    ) {
        // account::SignerCapability can't be copied, so once it's stored into
        // state, the init function can no longer be called (since
        // the deployer signer capability must have been unlocked).
        let signer_cap = deployer::claim_signer_capability(deployer, @wormhole);
        let message_fee = 0;
        let guardians: vector<Guardian> = vector[];

        vector::reverse(&mut initial_guardians);
        while (!vector::is_empty(&initial_guardians)) {
           vector::push_back(&mut guardians, create_guardian(vector::pop_back(&mut initial_guardians)));
        };

        init_internal(
            signer_cap,
            chain_id,
            governance_chain_id,
            governance_contract,
            guardians,
            u32::from_u64(86400),
            message_fee
        )
    }

    fun init_internal(
        signer_cap: account::SignerCapability,
        chain_id: u64,
        governance_chain_id: u64,
        governance_contract: vector<u8>,
        initial_guardians: vector<Guardian>,
        guardian_set_expiry: U32,
        message_fee: u64,
    ) {
        let wormhole = account::create_signer_with_capability(&signer_cap);
        state::init_wormhole_state(
            &wormhole,
            u16::from_u64(chain_id),
            u16::from_u64(governance_chain_id),
            external_address::from_bytes(governance_contract),
            guardian_set_expiry,
            message_fee,
            signer_cap
        );
        state::init_message_handles(&wormhole);
        state::store_guardian_set(
            create_guardian_set(
                u32::from_u64(0),
                initial_guardians
            )
        );
        // register wormhole to be able to receive fees
        // `aptos_account::create_account` is a permissionless operation that
        // performs aptos coin registration too, so it's possible that this is
        // already performed, in which case we just skip
        // (thanks ottersec for point this out)
        if (!coin::is_account_registered<AptosCoin>(signer::address_of(&wormhole))) {
            coin::register<AptosCoin>(&wormhole);
        };
    }

    #[test_only]
    /// Initialise a dummy contract for testing. Returns the wormhole signer.
    public fun init_test(
        chain_id: u64,
        governance_chain_id: u64,
        governance_contract: vector<u8>,
        initial_guardian: vector<u8>,
        message_fee: u64,
    ): signer {
        let deployer = account::create_account_for_test(@deployer);
        let (wormhole, signer_cap) = account::create_resource_account(&deployer, b"wormhole");
        init_internal(
            signer_cap,
            chain_id,
            governance_chain_id,
            governance_contract,
            vector[create_guardian(initial_guardian)],
            u32::from_u64(86400),
            message_fee
        );
        wormhole
    }
}

#[test_only]
module wormhole::wormhole_test {
    use std::hash;
    use wormhole::wormhole;
    use wormhole::keccak256::keccak256;
    use aptos_framework::aptos_coin::{Self};
    use aptos_framework::coin;

    // public so we an re-use this in token_bridge test
    public fun setup(message_fee: u64) {
        let aptos_framework = std::account::create_account_for_test(@aptos_framework);
        std::timestamp::set_time_has_started_for_testing(&aptos_framework);
        wormhole::init_test(
            22,
            1,
            x"0000000000000000000000000000000000000000000000000000000000000004",
            x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
            message_fee
        );
    }

    #[test]
    public fun test_hash() {
        assert!(hash::sha3_256(vector[0]) == x"5d53469f20fef4f8eab52b88044ede69c77a6a68a60728609fc4a65ff531e7d0", 0);
        assert!(keccak256(vector[0]) == x"bc36789e7a1e281436464229828f817d6612f7b477d66591ff96a9e064bcc98a", 0);
    }

    #[test(aptos_framework = @aptos_framework)]
    public fun test_publish_message(aptos_framework: &signer) {
        setup(100);

        let (burn_cap, mint_cap) = aptos_coin::initialize_for_test(aptos_framework);
        let fees = coin::mint(100, &mint_cap);

        let emitter_cap = wormhole::register_emitter();

        wormhole::publish_message(
            &mut emitter_cap,
            0,
            b"hi mom",
            fees
        );

        //TODO - check if event is actually emitted?

        wormhole::emitter::destroy_emitter_cap(emitter_cap);
        coin::destroy_mint_cap(mint_cap);
        coin::destroy_burn_cap(burn_cap);
    }

    #[test]
    #[expected_failure(abort_code = 0x0, location = wormhole::wormhole)]
    public fun test_publish_message_insufficient_fee() {
        setup(100);
        let emitter_cap = wormhole::register_emitter();

        wormhole::publish_message(
            &mut emitter_cap,
            0,
            b"hi mom",
            coin::zero()
        );
        wormhole::emitter::destroy_emitter_cap(emitter_cap);
    }
}
