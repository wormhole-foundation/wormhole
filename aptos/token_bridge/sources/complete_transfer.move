module token_bridge::complete_transfer {
    use std::signer;
    use aptos_std::from_bcs;
    use aptos_framework::coin::{Self, Coin};

    use aptos_framework::aptos_account;

    use token_bridge::vaa;
    use token_bridge::transfer::{Self, Transfer};
    use token_bridge::state;
    use token_bridge::wrapped;
    use token_bridge::normalized_amount;

    use wormhole::external_address::get_bytes;

    const E_INVALID_TARGET: u64 = 0;

    public fun submit_vaa<CoinType>(vaa: vector<u8>, fee_recipient: address): Transfer {
        let vaa = vaa::parse_verify_and_replay_protect(vaa);
        let transfer = transfer::parse(wormhole::vaa::destroy(vaa));
        complete_transfer<CoinType>(&transfer, fee_recipient);
        transfer
    }

    public entry fun submit_vaa_entry<CoinType>(vaa: vector<u8>, fee_recipient: address) {
        submit_vaa<CoinType>(vaa, fee_recipient);
    }

    /// Submits the complete transfer VAA and registers the coin for the fee
    /// recipient if not already registered.
    public entry fun submit_vaa_and_register_entry<CoinType>(fee_recipient: &signer, vaa: vector<u8>) {
        if (!coin::is_account_registered<CoinType>(signer::address_of(fee_recipient))) {
            coin::register<CoinType>(fee_recipient);
        };
        submit_vaa<CoinType>(vaa, signer::address_of(fee_recipient));
    }


    #[test_only]
    public fun test<CoinType>(transfer: &Transfer, fee_recipient: address) {
        complete_transfer<CoinType>(transfer, fee_recipient)
    }

    fun complete_transfer<CoinType>(transfer: &Transfer, fee_recipient: address) {
        let to_chain = transfer::get_to_chain(transfer);
        assert!(to_chain == wormhole::state::get_chain_id(), E_INVALID_TARGET);

        let token_chain = transfer::get_token_chain(transfer);
        let token_address = transfer::get_token_address(transfer);
        let origin_info = state::create_origin_info(token_chain, token_address);

        state::assert_coin_origin_info<CoinType>(origin_info);

        let decimals = coin::decimals<CoinType>();

        let amount = normalized_amount::denormalize(transfer::get_amount(transfer), decimals);
        let fee_amount = normalized_amount::denormalize(transfer::get_fee(transfer), decimals);

        let recipient = from_bcs::to_address(get_bytes(&transfer::get_to(transfer)));

        let recipient_coins: Coin<CoinType>;

        if (state::is_wrapped_asset<CoinType>()) {
            recipient_coins = wrapped::mint<CoinType>(amount);
        } else {
            let token_bridge = state::token_bridge_signer();
            recipient_coins = coin::withdraw<CoinType>(&token_bridge, amount);
        };

        // take out fee from the recipient's coins. `extract` will revert
        // if fee > amount
        let fee_coins = coin::extract(&mut recipient_coins, fee_amount);
        aptos_account::deposit_coins(recipient, recipient_coins);
        aptos_account::deposit_coins(fee_recipient, fee_coins);
    }

}

#[test_only]
module token_bridge::complete_transfer_test {
    use std::bcs;
    use std::signer;
    use aptos_framework::coin;
    use aptos_framework::string::{utf8};

    use aptos_framework::aptos_account;

    use token_bridge::transfer::{Self, Transfer};
    use token_bridge::transfer_tokens;
    use token_bridge::token_hash;
    use token_bridge::complete_transfer;
    use token_bridge::token_bridge;
    use token_bridge::wrapped;
    use token_bridge::transfer_result;
    use token_bridge::normalized_amount;

    use token_bridge::wrapped_test;

    use wormhole::state;
    use wormhole::wormhole_test;
    use wormhole::external_address;

    struct MyCoin {}

    struct OtherCoin {}

    fun init_my_token(admin: &signer, decimals: u8, amount: u64) {
        let name = utf8(b"mycoindd");
        let symbol = utf8(b"MCdd");
        let monitor_supply = true;
        let (burn_cap, freeze_cap, mint_cap) = coin::initialize<MyCoin>(admin, name, symbol, decimals, monitor_supply);
        coin::destroy_freeze_cap(freeze_cap);
        coin::destroy_burn_cap(burn_cap);
        aptos_account::deposit_coins(signer::address_of(admin), coin::mint(amount, &mint_cap));
        coin::destroy_mint_cap(mint_cap);
    }

    public fun setup(
        deployer: &signer,
        token_bridge: &signer,
        decimals: u8,
        amount: u64,
    ) {
        // initialise wormhole and token bridge
        wormhole_test::setup(0);
        token_bridge::init_test(deployer);

        // initialise MyToken
        init_my_token(token_bridge, decimals, amount);

        // initialise wrapped token
        wrapped_test::init_wrapped_token();
    }


    #[test(
        deployer = @deployer,
        token_bridge = @token_bridge,
    )]
    public fun test_native_transfer_10_decimals(
        deployer: &signer,
        token_bridge: &signer
    ) {
        let to = @0x12;
        let fee_recipient = @0x32;
        // the dust at the end will be removed during normalisation/denormalisation
        let amount = 10010;
        let fee_amount = 4000;
        let decimals = 10;

        setup(deployer, token_bridge, decimals, amount);

        let token_address = token_hash::get_external_address(&token_hash::derive<MyCoin>());
        let token_chain = state::get_chain_id();
        let to_chain = state::get_chain_id();
        let transfer: Transfer = transfer::create(
            normalized_amount::normalize(amount, decimals),
            token_address,
            token_chain,
            external_address::from_bytes(bcs::to_bytes(&to)),
            to_chain,
            normalized_amount::normalize(fee_amount, decimals),
        );

        assert!(!coin::is_account_registered<MyCoin>(to), 0);
        assert!(!coin::is_account_registered<MyCoin>(fee_recipient), 0);
        complete_transfer::test<MyCoin>(&transfer, fee_recipient);
        assert!(coin::balance<MyCoin>(to) == 6000, 0);
        assert!(coin::balance<MyCoin>(fee_recipient) == 4000, 0);
    }

    #[test(
        deployer = @deployer,
        token_bridge = @token_bridge,
    )]
    public fun test_native_transfer_4_decimals(
        deployer: &signer,
        token_bridge: &signer
    ) {
        let to = @0x12;
        let fee_recipient = @0x32;
        let amount = 100;
        let fee_amount = 40;
        let decimals = 4;

        // the token has 4 decimals, so no scaling is expected
        setup(deployer, token_bridge, decimals, amount);

        let token_address = token_hash::get_external_address(&token_hash::derive<MyCoin>());
        let token_chain = state::get_chain_id();
        let to_chain = state::get_chain_id();
        let transfer: Transfer = transfer::create(
            normalized_amount::normalize(amount, decimals),
            token_address,
            token_chain,
            external_address::from_bytes(bcs::to_bytes(&to)),
            to_chain,
            normalized_amount::normalize(fee_amount, decimals),
        );

        assert!(!coin::is_account_registered<MyCoin>(to), 0);
        assert!(!coin::is_account_registered<MyCoin>(fee_recipient), 0);
        complete_transfer::test<MyCoin>(&transfer, fee_recipient);
        assert!(coin::balance<MyCoin>(to) == 60, 0);
        assert!(coin::balance<MyCoin>(fee_recipient) == 40, 0);
    }

    #[test(
        deployer = @deployer,
        token_bridge = @token_bridge,
    )]
    #[expected_failure(abort_code = 65542, location = aptos_framework::coin)] // EINSUFFICIENT_BALANCE
    public fun test_native_too_much_fee(
        deployer: &signer,
        token_bridge: &signer
    ) {
        let to = @0x12;
        let fee_recipient = @0x32;
        let amount = 100;
        let fee_amount = 101; // FAIL: too much fee
        let decimals = 8;

        setup(deployer, token_bridge, decimals, amount);

        let token_address = token_hash::get_external_address(&token_hash::derive<MyCoin>());
        let token_chain = state::get_chain_id();
        let to_chain = state::get_chain_id();
        let transfer: Transfer = transfer::create(
            normalized_amount::normalize(amount, decimals),
            token_address,
            token_chain,
            external_address::from_bytes(bcs::to_bytes(&to)),
            to_chain,
            normalized_amount::normalize(fee_amount, decimals),
        );
        complete_transfer::test<MyCoin>(&transfer, fee_recipient);
    }

    #[test(
        deployer = @deployer,
        token_bridge = @token_bridge,
    )]
    #[expected_failure(abort_code = 1, location = token_bridge::state)] // E_ORIGIN_ADDRESS_MISMATCH
    public fun test_native_wrong_coin(
        deployer: &signer,
        token_bridge: &signer
    ) {
        let to = @0x12;
        let fee_recipient = @0x32;
        let amount = 100;
        let fee_amount = 40;
        let decimals = 8;

        setup(deployer, token_bridge, decimals, amount);

        let token_address = token_hash::get_external_address(&token_hash::derive<MyCoin>());
        let token_chain = state::get_chain_id();
        let to_chain = state::get_chain_id();
        let transfer: Transfer = transfer::create(
            normalized_amount::normalize(amount, decimals),
            token_address,
            token_chain,
            external_address::from_bytes(bcs::to_bytes(&to)),
            to_chain,
            normalized_amount::normalize(fee_amount, decimals),
        );
        // FAIL: wrong type argument
        complete_transfer::test<OtherCoin>(&transfer, fee_recipient);
    }

    #[test(
        deployer = @deployer,
        token_bridge = @token_bridge,
    )]
    #[expected_failure(abort_code = 0, location = token_bridge::state)] // E_ORIGIN_CHAIN_MISMATCH
    public fun test_native_wrong_origin_address(
        deployer: &signer,
        token_bridge: &signer
    ) {
        let to = @0x12;
        let fee_recipient = @0x32;
        let amount = 100;
        let fee_amount = 40;
        let decimals = 8;

        setup(deployer, token_bridge, decimals, amount);

        let token_address = token_hash::get_external_address(&token_hash::derive<MyCoin>());
        let token_chain = wormhole::u16::from_u64(10); // FAIL: wrong origin chain (MyCoin is native)
        let to_chain = state::get_chain_id();
        let transfer: Transfer = transfer::create(
            normalized_amount::normalize(amount, decimals),
            token_address,
            token_chain,
            external_address::from_bytes(bcs::to_bytes(&to)),
            to_chain,
            normalized_amount::normalize(fee_amount, decimals),
        );
        complete_transfer::test<MyCoin>(&transfer, fee_recipient);
    }

    #[test(
        deployer = @deployer,
        token_bridge = @token_bridge,
    )]
    public fun test_wrapped_transfer_roundtrip(
        deployer: &signer,
        token_bridge: &signer
    ) {
        let to = @0x12;
        let fee_recipient = @0x32;

        setup(deployer, token_bridge, 8, 0);

        let beef_coins = wrapped::mint<wrapped_coin::coin::T>(100000);

        let result = transfer_tokens::transfer_tokens_test<wrapped_coin::coin::T>(
            beef_coins,
            5000
        );

        let (token_chain, token_address, normalized_amount, normalized_relayer_fee)
            = transfer_result::destroy(result);


        let to_chain = state::get_chain_id();
        let transfer: Transfer = transfer::create(
            normalized_amount,
            token_address,
            token_chain,
            external_address::from_bytes(bcs::to_bytes(&to)),
            to_chain,
            normalized_relayer_fee,
        );

        assert!(!coin::is_account_registered<wrapped_coin::coin::T>(to), 0);
        assert!(!coin::is_account_registered<wrapped_coin::coin::T>(fee_recipient), 0);

        complete_transfer::test<wrapped_coin::coin::T>(&transfer, fee_recipient);

        assert!(coin::balance<wrapped_coin::coin::T>(to) == 95000, 0);
        assert!(coin::balance<wrapped_coin::coin::T>(fee_recipient) == 5000, 0);
    }

    #[test(
        deployer = @deployer,
        token_bridge = @token_bridge,
    )]
    public fun test_wrapped_transfer(
        deployer: &signer,
        token_bridge: &signer
    ) {
        let to = @0x12;
        let fee_recipient = @0x32;
        let amount = 100;
        let fee_amount = 40;
        let decimals = 9;

        setup(deployer, token_bridge, decimals, 0);

        let token_address = external_address::from_bytes(x"deadbeef");
        let token_chain = wormhole::u16::from_u64(2);
        let to_chain = state::get_chain_id();
        let transfer: Transfer = transfer::create(
            normalized_amount::normalize(amount, decimals),
            token_address,
            token_chain,
            external_address::from_bytes(bcs::to_bytes(&to)),
            to_chain,
            normalized_amount::normalize(fee_amount, decimals),
        );

        assert!(!coin::is_account_registered<wrapped_coin::coin::T>(to), 0);
        assert!(!coin::is_account_registered<wrapped_coin::coin::T>(fee_recipient), 0);

        complete_transfer::test<wrapped_coin::coin::T>(&transfer, fee_recipient);

        // the wrapped asset has 8 decimals (see wrapped_test::init_wrapped_token)
        assert!(coin::balance<wrapped_coin::coin::T>(to) == 6, 0);
        assert!(coin::balance<wrapped_coin::coin::T>(fee_recipient) == 4, 0);
    }
}
