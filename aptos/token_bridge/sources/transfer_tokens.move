module token_bridge::transfer_tokens {
    use aptos_framework::aptos_coin::{AptosCoin};
    use aptos_framework::coin::{Self, Coin};

    use std::signer;

    use wormhole::u16::{Self, U16};
    use wormhole::external_address::{Self, ExternalAddress};
    use wormhole::emitter::{Self, EmitterCapability};

    use token_bridge::state;
    use token_bridge::transfer;
    use token_bridge::transfer_result::{Self, TransferResult};
    use token_bridge::transfer_with_payload;
    use token_bridge::normalized_amount;
    use token_bridge::wrapped;

    const E_TOO_MUCH_RELAYER_FEE: u64 = 0;

    public entry fun transfer_tokens_entry<CoinType>(
        sender: &signer,
        amount: u64,
        recipient_chain: u64,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u64
    ) {
        let coins = coin::withdraw<CoinType>(sender, amount);
        let wormhole_fee = wormhole::state::get_message_fee();
        let wormhole_fee_coins = coin::withdraw<AptosCoin>(sender, wormhole_fee);
        transfer_tokens<CoinType>(
            coins,
            wormhole_fee_coins,
            u16::from_u64(recipient_chain),
            external_address::from_bytes(recipient),
            relayer_fee,
            nonce
        );
    }

    public fun transfer_tokens<CoinType>(
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<AptosCoin>,
        recipient_chain: U16,
        recipient: ExternalAddress,
        relayer_fee: u64,
        nonce: u64
    ): u64 {
        let result = transfer_tokens_internal<CoinType>(coins, relayer_fee);
        let (token_chain, token_address, normalized_amount, normalized_relayer_fee)
            = transfer_result::destroy(result);
        let transfer = transfer::create(
            normalized_amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            normalized_relayer_fee,
        );
        state::publish_message(
            nonce,
            transfer::encode(transfer),
            wormhole_fee_coins,
        )
    }

    /// This struct stores the emitter capability for a given user account.
    struct EmitterCapabilityStore has key {
        emitter_cap: EmitterCapability,
    }

    #[view]
    public fun is_emitter_registered(account: address): bool {
        exists<EmitterCapabilityStore>(account)
    }

    public entry fun register_emitter(sender: &signer) {
        if (is_emitter_registered(signer::address_of(sender))) {
            return
        };
        let emitter_cap = wormhole::wormhole::register_emitter();
        move_to<EmitterCapabilityStore>(
            sender,
            EmitterCapabilityStore { emitter_cap }
        );
    }

    public entry fun transfer_tokens_with_payload_entry<CoinType>(
        sender: &signer,
        amount: u64,
        recipient_chain: u64,
        recipient: vector<u8>,
        nonce: u64,
        payload: vector<u8>
    ) acquires EmitterCapabilityStore {
        register_emitter(sender);

        let EmitterCapabilityStore { emitter_cap } =
            borrow_global<EmitterCapabilityStore>(signer::address_of(sender));

        let coins = coin::withdraw<CoinType>(sender, amount);
        let wormhole_fee = wormhole::state::get_message_fee();
        let wormhole_fee_coins = coin::withdraw<AptosCoin>(sender, wormhole_fee);
        transfer_tokens_with_payload<CoinType>(
            emitter_cap,
            coins,
            wormhole_fee_coins,
            u16::from_u64(recipient_chain),
            external_address::from_bytes(recipient),
            nonce,
            payload
        );
    }

    public fun transfer_tokens_with_payload<CoinType>(
        emitter_cap: &EmitterCapability,
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<AptosCoin>,
        recipient_chain: U16,
        recipient: ExternalAddress,
        nonce: u64,
        payload: vector<u8>
    ): u64 {
        let result = transfer_tokens_internal<CoinType>(coins, 0);
        let (token_chain, token_address, normalized_amount, _)
            = transfer_result::destroy(result);

        let transfer = transfer_with_payload::create(
            normalized_amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            emitter::get_external_address(emitter_cap),
            payload
        );
        let payload = transfer_with_payload::encode(transfer);
        state::publish_message(
            nonce,
            payload,
            wormhole_fee_coins,
        )
    }

    #[test_only]
    public fun transfer_tokens_test<CoinType>(
        coins: Coin<CoinType>,
        relayer_fee: u64,
    ): TransferResult {
        transfer_tokens_internal(coins, relayer_fee)
    }

    // transfer a native or wrapped token from sender to token_bridge
    fun transfer_tokens_internal<CoinType>(
        coins: Coin<CoinType>,
        relayer_fee: u64,
    ): TransferResult {

        // transfer coin to token_bridge
        if (!coin::is_account_registered<CoinType>(@token_bridge)) {
            coin::register<CoinType>(&state::token_bridge_signer());
        };
        if (!coin::is_account_registered<AptosCoin>(@token_bridge)) {
            coin::register<AptosCoin>(&state::token_bridge_signer());
        };

        let amount = coin::value<CoinType>(&coins);
        assert!(relayer_fee <= amount, E_TOO_MUCH_RELAYER_FEE);

        if (state::is_wrapped_asset<CoinType>()) {
            // now we burn the wrapped coins to remove them from circulation
            wrapped::burn<CoinType>(coins);
        } else {
            coin::deposit<CoinType>(@token_bridge, coins);
            // if we're seeing this native token for the first time, store its
            // type info
            if (!state::is_registered_native_asset<CoinType>()) {
                state::set_native_asset_type_info<CoinType>();
            };
        };

        let origin_info = state::origin_info<CoinType>();
        let token_chain = state::get_origin_info_token_chain(&origin_info);
        let token_address = state::get_origin_info_token_address(&origin_info);

        let decimals_token = coin::decimals<CoinType>();

        let normalized_amount = normalized_amount::normalize(amount, decimals_token);
        let normalized_relayer_fee = normalized_amount::normalize(relayer_fee, decimals_token);

        let transfer_result: TransferResult = transfer_result::create(
            token_chain,
            token_address,
            normalized_amount,
            normalized_relayer_fee,
        );
        transfer_result
    }


}

#[test_only]
module token_bridge::transfer_tokens_test {
    use aptos_framework::coin::{Self, Coin};
    use aptos_framework::string::{utf8};
    use aptos_framework::aptos_coin::{Self, AptosCoin};
    use aptos_framework::aptos_account;

    use token_bridge::token_bridge::{Self as bridge};
    use token_bridge::transfer_tokens;
    use token_bridge::wrapped;
    use token_bridge::transfer_result;
    use token_bridge::token_hash;
    use token_bridge::register_chain;
    use token_bridge::normalized_amount;

    use wormhole::external_address::{Self};

    use wrapped_coin::coin::T;

    /// Registration VAA for the ethereum token bridge 0xdeadbeef
    /// +------------------------------------------------------------------------------+
    /// | Wormhole VAA v1         | nonce: 1                | time: 1                  |
    /// | guardian set #0         | #23663022               | consistency: 0           |
    /// |------------------------------------------------------------------------------|
    /// | Signature:                                                                   |
    /// |   #0: 15d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e...        |
    /// |------------------------------------------------------------------------------|
    /// | Emitter: 11111111111111111111111111111115 (Solana)                           |
    /// |==============================================================================|
    /// | Chain registration (TokenBridge)                                             |
    /// | Emitter chain: Ethereum                                                      |
    /// | Emitter address: 0x00000000000000000000000000000000deadbeef (Ethereum)       |
    /// +------------------------------------------------------------------------------+
    const ETHEREUM_TOKEN_REG: vector<u8> = x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    /// Attestation VAA sent from the ethereum token bridge 0xdeadbeef
    /// +------------------------------------------------------------------------------+
    /// | Wormhole VAA v1         | nonce: 1                | time: 1                  |
    /// | guardian set #0         | #22080291               | consistency: 0           |
    /// |------------------------------------------------------------------------------|
    /// | Signature:                                                                   |
    /// |   #0: 80366065746148420220f25a6275097370e8db40984529a6676b7a5fc9fe...        |
    /// |------------------------------------------------------------------------------|
    /// | Emitter: 0x00000000000000000000000000000000deadbeef (Ethereum)               |
    /// |==============================================================================|
    /// | Token attestation                                                            |
    /// | decimals: 12                                                                 |
    /// | Token: 0x00000000000000000000000000000000beefface (Ethereum)                 |
    /// | Symbol: BEEF                                                                 |
    /// | Name: Beef face Token                                                        |
    /// +------------------------------------------------------------------------------+
    const ATTESTATION_VAA: vector<u8> = x"0100000000010080366065746148420220f25a6275097370e8db40984529a6676b7a5fc9feb11755ec49ca626b858ddfde88d15601f85ab7683c5f161413b0412143241c700aff010000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef000000000150eb23000200000000000000000000000000000000000000000000000000000000beefface00020c424545460000000000000000000000000000000000000000000000000000000042656566206661636520546f6b656e0000000000000000000000000000000000";

    struct MyCoin has key {}

    fun init_my_token(admin: &signer, amount: u64): Coin<MyCoin> {
        let name = utf8(b"mycoindd");
        let symbol = utf8(b"MCdd");
        let decimals = 6;
        let monitor_supply = true;
        let (burn_cap, freeze_cap, mint_cap) = coin::initialize<MyCoin>(admin, name, symbol, decimals, monitor_supply);
        let coins = coin::mint<MyCoin>(amount, &mint_cap);
        coin::destroy_burn_cap(burn_cap);
        coin::destroy_mint_cap(mint_cap);
        coin::destroy_freeze_cap(freeze_cap);
        coins
    }

    fun setup(
        aptos_framework: &signer,
        token_bridge: &signer,
        deployer: &signer,
    ) {
        // we initialise the bridge with zero fees to avoid having to mint fee
        // tokens in these tests. The wormhole fee handling is already tested
        // in wormhole.move, so it's unnecessary here.
        let (burn_cap, mint_cap) = aptos_coin::initialize_for_test(aptos_framework);
        wormhole::wormhole_test::setup(0);
        bridge::init_test(deployer);

        coin::register<AptosCoin>(deployer);
        coin::register<AptosCoin>(token_bridge); //how important is this registration step and where to check it?
        coin::destroy_burn_cap(burn_cap);
        coin::destroy_mint_cap(mint_cap);
    }

    // test transfer wrapped coin
    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_transfer_wrapped_token(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, token_bridge, deployer);
        register_chain::submit_vaa(ETHEREUM_TOKEN_REG);
        // TODO(csongor): create a better error message when attestation is missing
        wrapped::create_wrapped_coin_type(ATTESTATION_VAA);
        // TODO(csongor): write a blurb about why this test works (something
        // something static linking)
        // initialize coin using type T, move caps to token_bridge, sets bridge state variables
        wrapped::create_wrapped_coin<T>(ATTESTATION_VAA);

        // test transfer wrapped tokens
        let beef_coins = wrapped::mint<T>(100000);
        assert!(coin::supply<T>() == std::option::some(100000), 0);
        let result = transfer_tokens::transfer_tokens_test<T>(
            beef_coins,
            2,
        );
        let (token_chain, token_address, normalized_amount, normalized_relayer_fee)
            = transfer_result::destroy(result);

        // make sure the wrapped assets have been burned
        assert!(coin::supply<T>() == std::option::some(0), 0);

        assert!(token_chain == wormhole::u16::from_u64(2), 0);
        assert!(external_address::get_bytes(&token_address) == x"00000000000000000000000000000000000000000000000000000000beefface", 0);
        // the original coin has 12 decimals, but wrapped assets are capped at 8
        // decimals, so the normalized amount matches the transferred amount.
        assert!(normalized_amount::get_amount(normalized_amount) == 100000, 0);
        assert!(normalized_amount::get_amount(normalized_relayer_fee) == 2, 0);
    }

    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    #[expected_failure(abort_code = 0, location = token_bridge::transfer_tokens)]
    fun test_transfer_wrapped_token_too_much_relayer_fee(
        aptos_framework: &signer,
        token_bridge: &signer,
        deployer: &signer
    ) {
        setup(aptos_framework, token_bridge, deployer);
        register_chain::submit_vaa(ETHEREUM_TOKEN_REG);
        wrapped::create_wrapped_coin_type(ATTESTATION_VAA);
        wrapped::create_wrapped_coin<T>(ATTESTATION_VAA);

        // this will fail because the relayer fee exceeds the amount
        let beef_coins = wrapped::mint<T>(100000);
        assert!(coin::supply<T>() == std::option::some(100000), 0);
        let result = transfer_tokens::transfer_tokens_test<T>(beef_coins, 200000);
        let (_, _, _, _) = transfer_result::destroy(result);
    }

    // test transfer native coin
    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_transfer_native_token(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, token_bridge, deployer);

        let my_coins = init_my_token(token_bridge, 10000);

        // make sure the token bridge is not registered yet for this coin
        assert!(!coin::is_account_registered<MyCoin>(@token_bridge), 0);

        let result = transfer_tokens::transfer_tokens_test<MyCoin>(my_coins, 500);

        // the token bridge should now be registered and hold the balance
        assert!(coin::balance<MyCoin>(@token_bridge) == 10000, 0);

        let (token_chain, token_address, normalized_amount, normalized_relayer_fee)
            = transfer_result::destroy(result);

        assert!(token_chain == wormhole::state::get_chain_id(), 0);
        assert!(token_address == token_hash::get_external_address(&token_hash::derive<MyCoin>()), 0);
        // the coin has 6 decimals, so the amount doesn't get scaled
        assert!(normalized_amount::get_amount(normalized_amount) == 10000, 0);
        assert!(normalized_amount::get_amount(normalized_relayer_fee) == 500, 0);
    }

    // test transfer with payload entry
    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer, user=@0xBEEF)]
    fun test_transfer_with_payload(aptos_framework: &signer, token_bridge: &signer, deployer: &signer, user: &signer) {
        setup(aptos_framework, token_bridge, deployer);

        let my_coins = init_my_token(token_bridge, 10000);
        aptos_account::deposit_coins(std::signer::address_of(user), my_coins);

        transfer_tokens::transfer_tokens_with_payload_entry<MyCoin>(
            user,
            500,
            2,
            x"01",
            10,
            x"BEEFFACE"
        );

        assert!(coin::balance<MyCoin>(std::signer::address_of(user)) == 9500, 0);

        // let's send again to make sure we can
        transfer_tokens::transfer_tokens_with_payload_entry<MyCoin>(
            user,
            500,
            2,
            x"01",
            10,
            x"BEEFFACE"
        );

        assert!(coin::balance<MyCoin>(std::signer::address_of(user)) == 9000, 0);
    }
}
