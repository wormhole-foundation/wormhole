module token_bridge::transfer_tokens {
    use aptos_framework::aptos_coin::{AptosCoin};
    use aptos_framework::bcs::to_bytes;
    use aptos_framework::coin::{Self, Coin};

    use wormhole::u16::{Self, U16};
    use wormhole::u256;

    use token_bridge::bridge_state as state;
    use token_bridge::transfer;
    use token_bridge::transfer_result::{Self, TransferResult};
    use token_bridge::transfer_with_payload;
    use token_bridge::utils;

    public entry fun transfer_tokens_with_signer<CoinType>(
        sender: &signer,
        amount: u64,
        recipient_chain: u64,
        recipient: vector<u8>,
        relayer_fee: u64,
        wormhole_fee: u64,
        nonce: u64
        ): u64 {
        let coins = coin::withdraw<CoinType>(sender, amount);
        //let relayer_fee_coins = coin::withdraw<AptosCoin>(sender, relayer_fee);
        let wormhole_fee_coins = coin::withdraw<AptosCoin>(sender, wormhole_fee);
        transfer_tokens<CoinType>(coins, wormhole_fee_coins, u16::from_u64(recipient_chain), recipient, relayer_fee, nonce)
    }

    public fun transfer_tokens<CoinType>(
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<AptosCoin>,
        recipient_chain: U16,
        recipient: vector<u8>,
        relayer_fee: u64,
        nonce: u64
        ): u64 {
        let wormhole_fee = coin::value<AptosCoin>(&wormhole_fee_coins);
        let result = transfer_tokens_internal<CoinType>(coins, relayer_fee, wormhole_fee);
        let transfer = transfer::create(
            transfer_result::get_normalized_amount(&result),
            transfer_result::get_token_address(&result),
            transfer_result::get_token_chain(&result),
            recipient,
            recipient_chain,
            transfer_result::get_normalized_relayer_fee(&result),
        );
        state::publish_message(
            nonce,
            transfer::encode(transfer),
            wormhole_fee_coins,
        )
    }

    public fun transfer_tokens_with_payload_with_signer<CoinType>(
        sender: &signer,
        amount: u64,
        wormhole_fee: u64,
        recipient_chain: U16,
        recipient: vector<u8>,
        nonce: u64,
        payload: vector<u8>
        ): u64 {
        let coins = coin::withdraw<CoinType>(sender, amount);
        let wormhole_fee_coins = coin::withdraw<AptosCoin>(sender, wormhole_fee);
        transfer_tokens_with_payload(coins, wormhole_fee_coins, recipient_chain, recipient, nonce, payload)
    }

    public fun transfer_tokens_with_payload<CoinType>(
        coins: Coin<CoinType>,
        wormhole_fee_coins: Coin<AptosCoin>,
        recipient_chain: U16,
        recipient: vector<u8>,
        nonce: u64,
        payload: vector<u8>
        ): u64 {
        let result = transfer_tokens_internal<CoinType>(coins, 0, 0); // TODO: the wormhole fee 0 is sus
        let transfer = transfer_with_payload::create(
            transfer_result::get_normalized_amount(&result),
            transfer_result::get_token_address(&result),
            transfer_result::get_token_chain(&result),
            recipient,
            recipient_chain,
            to_bytes<address>(&@token_bridge), //TODO - is token bridge the only one who will ever call log_transfer_with_payload?
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
        wormhole_fee: u64
    ): TransferResult {
        transfer_tokens_internal(coins, relayer_fee, wormhole_fee)
    }

    // transfer a native or wraped token from sender to token_bridge
    fun transfer_tokens_internal<CoinType>(
        coins: Coin<CoinType>,
        relayer_fee: u64,
        wormhole_fee: u64,
        ): TransferResult {

        // transfer coin to token_bridge
        if (!coin::is_account_registered<CoinType>(@token_bridge)){
            coin::register<CoinType>(&state::token_bridge_signer());
        };
        if (!coin::is_account_registered<AptosCoin>(@token_bridge)){
            coin::register<AptosCoin>(&state::token_bridge_signer());
        };
        // TODO: check that fee <= amount
        let amount = coin::value<CoinType>(&coins);
        coin::deposit<CoinType>(@token_bridge, coins);

        if (state::is_wrapped_asset<CoinType>()) {
            // now we burn the wrapped coins to remove them from circulation
            // TODO - wrapped::burn<CoinType>(amount);
            // wrapped::burn<CoinType>(amount);
            // problem here is that wrapped imports state, so state can't import wrapped...
        } else {
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

        let normalized_amount = utils::normalize_amount(u256::from_u64(amount), decimals_token);
        let normalized_relayer_fee = utils::normalize_amount(u256::from_u64(relayer_fee), decimals_token);

        let transfer_result: TransferResult = transfer_result::create(
            token_chain,
            token_address,
            normalized_amount,
            normalized_relayer_fee,
            u256::from_u64(wormhole_fee)
        );
        transfer_result
    }


}

#[test_only]
module token_bridge::transfer_tokens_test {
    use aptos_framework::coin::{Self, MintCapability, FreezeCapability, BurnCapability};
    use aptos_framework::string::{utf8};
    use aptos_framework::aptos_coin::{Self, AptosCoin};
    use aptos_framework::signer;

    use token_bridge::token_bridge::{Self as bridge};
    use token_bridge::transfer_tokens;
    use token_bridge::wrapped;

    use token_bridge::register_chain;

    use wormhole::u16;

    use wrapped_coin::coin::T;

    /// Registration VAA for the etheruem token bridge 0xdeadbeef
    const ETHEREUM_TOKEN_REG: vector<u8> = x"0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef";

    /// Attestation VAA sent from the ethereum token bridge 0xdeadbeef
    const ATTESTATION_VAA: vector<u8> = x"01000000000100102d399190fa61daccb11c2ea4f7a3db3a9365e5936bcda4cded87c1b9eeb095173514f226256d5579af71d4089eb89496befb998075ba94cd1d4460c5c57b84000000000100000001000200000000000000000000000000000000000000000000000000000000deadbeef0000000002634973000200000000000000000000000000000000000000000000000000000000beefface00020c0000000000000000000000000000000000000000000000000000000042454546000000000000000000000000000000000042656566206661636520546f6b656e";

    struct MyCoin has key {}

    struct MyCoinCaps<phantom CoinType> has key, store {
        burn_cap: BurnCapability<CoinType>,
        freeze_cap: FreezeCapability<CoinType>,
        mint_cap: MintCapability<CoinType>,
    }

    fun init_my_token(admin: &signer) {
        let name = utf8(b"mycoindd");
        let symbol = utf8(b"MCdd");
        let decimals = 10;
        let monitor_supply = true;
        let (burn_cap, freeze_cap, mint_cap) = coin::initialize<MyCoin>(admin, name, symbol, decimals, monitor_supply);
        move_to(admin, MyCoinCaps {burn_cap, freeze_cap, mint_cap});
    }

    fun setup(
        aptos_framework: &signer,
        token_bridge: &signer,
        deployer: &signer,
    ) {
        // we initialise the bridge with zero fees to avoid having to mint fee
        // tokens in these tests. The wormolhe fee handling is already tested
        // in wormhole.move, so it's unnecessary here.
        let (burn_cap, mint_cap) = aptos_coin::initialize_for_test(aptos_framework);
        wormhole::wormhole_test::setup(0);
        bridge::init_test(deployer);

        coin::register<AptosCoin>(deployer);
        coin::register<AptosCoin>(token_bridge); //how important is this registration step and where to check it?
        coin::destroy_burn_cap(burn_cap);
        coin::destroy_mint_cap(mint_cap);
    }

    // test transfer wrapped coin (with and without payload)
    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_transfer_wrapped_token(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) {
        setup(aptos_framework, token_bridge, deployer);
        register_chain::submit_vaa(ETHEREUM_TOKEN_REG);
        // TODO(csongor): create a better error message when attestation is missing
        let _addr = wrapped::create_wrapped_coin_type(ATTESTATION_VAA);
        // TODO(csongor): write a blurb about why this test works (something
        // something static linking)
        // initialize coin using type T, move caps to token_bridge, sets bridge state variables
        wrapped::create_wrapped_coin<T>(ATTESTATION_VAA);

        // test transfer wrapped tokens
        let beef_coins = wrapped::mint<T>(10000);
        let _sequence = transfer_tokens::transfer_tokens<T>(
            beef_coins,
            coin::zero(),
            u16::from_u64(2),
            x"C973E38e87A0571446dC6Ad17C28217F079583C2",
            0,
            0
        );

        //test transfer wrapped tokens with payload
        let beef_coins = wrapped::mint<T>(10000);
        let _sequence = transfer_tokens::transfer_tokens_with_payload<T>(
            beef_coins,
            coin::zero(),
            u16::from_u64(2),
            x"C973E38e87A0571446dC6Ad17C28217F079583C2",
            0,
            x"beeeff",
        );
    }

    // test transfer native coin (with and without payload)
    #[test(aptos_framework = @aptos_framework, token_bridge=@token_bridge, deployer=@deployer)]
    fun test_transfer_native_token(aptos_framework: &signer, token_bridge: &signer, deployer: &signer) acquires MyCoinCaps{
        setup(aptos_framework, token_bridge, deployer);
        init_my_token(token_bridge);
        let MyCoinCaps {burn_cap, freeze_cap, mint_cap} = move_from<MyCoinCaps<MyCoin>>(signer::address_of(token_bridge));

        // test transfer native coins
        let my_coins = coin::mint<MyCoin>(10000, &mint_cap);
        let _sequence = transfer_tokens::transfer_tokens<MyCoin>(
            my_coins,
            coin::zero(),
            u16::from_u64(2),
            x"C973E38e87A0571446dC6Ad17C28217F079583C2",
            0,
            0
        );

         // test transfer native coins with payload
        let my_coins = coin::mint<MyCoin>(10000, &mint_cap);
        let _sequence = transfer_tokens::transfer_tokens_with_payload<MyCoin>(
            my_coins,
            coin::zero(),
            u16::from_u64(2),
            x"C973E38e87A0571446dC6Ad17C28217F079583C2",
            0,
            x"beeeff",
        );

        // destroy coin caps
        coin::destroy_mint_cap<MyCoin>(mint_cap);
        coin::destroy_burn_cap<MyCoin>(burn_cap);
        coin::destroy_freeze_cap<MyCoin>(freeze_cap);
    }
}
