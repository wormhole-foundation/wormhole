module token_bridge::complete_transfer {
    use aptos_std::from_bcs;
    use aptos_framework::coin::{Self, Coin};

    use token_bridge::vaa;
    use token_bridge::transfer::{Self, Transfer};
    use token_bridge::bridge_state as state;
    use token_bridge::wrapped;

    use wormhole::u256;

    const E_INVALID_TARGET: u64 = 0;

    public entry fun submit_vaa<CoinType>(vaa: vector<u8>, fee_recipient: address): Transfer {
        let vaa = vaa::parse_verify_and_replay_protect(vaa);
        let transfer = transfer::parse(wormhole::vaa::destroy(vaa));
        complete_transfer<CoinType>(&transfer, fee_recipient);
        transfer
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
        let origin_info = state::create_origin_info(token_address, token_chain);

        state::assert_coin_origin_info<CoinType>(origin_info);

        // Convert to u64. Aborts in case of overflow
        let amount = u256::as_u64(transfer::get_amount(transfer));
        let fee_amount = u256::as_u64(transfer::get_fee(transfer));

        let recipient = from_bcs::to_address(transfer::get_to(transfer));

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
        coin::deposit(recipient, recipient_coins);
        coin::deposit(fee_recipient, fee_coins);
    }

}

#[test_only]
module token_bridge::complete_transfer_test {
    use std::bcs;
    use std::signer;
    use aptos_framework::coin;
    use aptos_framework::account;
    use aptos_framework::string::{utf8};

    use token_bridge::transfer::{Self, Transfer};
    use token_bridge::token_hash;
    use token_bridge::complete_transfer;
    use token_bridge::token_bridge;
    use token_bridge::wrapped;
    use token_bridge::asset_meta;
    use token_bridge::string32;

    use wormhole::u256;
    use wormhole::state;
    use wormhole::wormhole_test;

    struct MyCoin {}

    struct OtherCoin {}

    fun init_my_token(admin: &signer, amount: u64) {
        let name = utf8(b"mycoindd");
        let symbol = utf8(b"MCdd");
        let decimals = 10;
        let monitor_supply = true;
        let (burn_cap, freeze_cap, mint_cap) = coin::initialize<MyCoin>(admin, name, symbol, decimals, monitor_supply);
        coin::destroy_freeze_cap(freeze_cap);
        coin::destroy_burn_cap(burn_cap);
        coin::register<MyCoin>(admin);
        coin::deposit(signer::address_of(admin), coin::mint(amount, &mint_cap));
        coin::destroy_mint_cap(mint_cap);
    }

    fun init_wrapped_token() {
        let token_address = x"00000000000000000000000000000000000000000000000000000000deadbeef";
        let asset_meta = asset_meta::create(
            0, // ARGH
            token_address,
            wormhole::u16::from_u64(2),
            9,
            string32::from_bytes(b"foo"),
            string32::from_bytes(b"Foo bar token")
        );
        let wrapped_coin = account::create_account_for_test(@wrapped_coin);
        wrapped::init_wrapped_coin<wrapped_coin::coin::T>(&wrapped_coin, &asset_meta);
    }

    public fun setup(
        deployer: &signer,
        token_bridge: &signer,
        to: address,
        fee_recipient: address,
        amount: u64,
    ) {
        // initialise wormhole and token bridge
        wormhole_test::setup(0);
        token_bridge::init_test(deployer);

        // initialise MyToken
        init_my_token(token_bridge, amount);

        // initialise 'to' and 'fee_recipient' and register them to accept MyCoins
        let to = &account::create_account_for_test(to);
        let fee_recipient = &account::create_account_for_test(fee_recipient);
        coin::register<MyCoin>(to);
        coin::register<MyCoin>(fee_recipient);

        // initialise wrapped token
        init_wrapped_token();
        coin::register<wrapped_coin::coin::T>(to);
        coin::register<wrapped_coin::coin::T>(fee_recipient);

    }


    #[test(
        deployer = @deployer,
        token_bridge = @token_bridge,
    )]
    public fun test_native_transfer(
        deployer: &signer,
        token_bridge: &signer
    ) {
        let to = @0x12;
        let fee_recipient = @0x32;
        let amount = 100;
        let fee_amount = 40;

        setup(deployer, token_bridge, to, fee_recipient, amount);

        let token_address = token_hash::get_bytes(&token_hash::derive<MyCoin>());
        let token_chain = state::get_chain_id();
        let to_chain = state::get_chain_id();
        let fee = u256::from_u64(fee_amount);
        let transfer: Transfer = transfer::create(
            1, // ARGH
            u256::from_u64(amount),
            token_address,
            token_chain,
            bcs::to_bytes(&to),
            to_chain,
            fee,
        );

        assert!(coin::balance<MyCoin>(to) == 0, 0);
        assert!(coin::balance<MyCoin>(fee_recipient) == 0, 0);
        complete_transfer::test<MyCoin>(&transfer, fee_recipient);
        assert!(coin::balance<MyCoin>(to) == 60, 0);
        assert!(coin::balance<MyCoin>(fee_recipient) == 40, 0);
    }

    #[test(
        deployer = @deployer,
        token_bridge = @token_bridge,
    )]
    #[expected_failure(abort_code = 65542)] // EINSUFFICIENT_BALANCE
    public fun test_native_too_much_fee(
        deployer: &signer,
        token_bridge: &signer
    ) {
        let to = @0x12;
        let fee_recipient = @0x32;
        let amount = 100;
        let fee_amount = 101; // FAIL: too much fee

        setup(deployer, token_bridge, to, fee_recipient, amount);

        let token_address = token_hash::get_bytes(&token_hash::derive<MyCoin>());
        let token_chain = state::get_chain_id();
        let to_chain = state::get_chain_id();
        let fee = u256::from_u64(fee_amount);
        let transfer: Transfer = transfer::create(
            1, // ARGH
            u256::from_u64(amount),
            token_address,
            token_chain,
            bcs::to_bytes(&to),
            to_chain,
            fee,
        );
        complete_transfer::test<MyCoin>(&transfer, fee_recipient);
    }

    #[test(
        deployer = @deployer,
        token_bridge = @token_bridge,
    )]
    #[expected_failure(abort_code = 1)] // E_ORIGIN_ADDRESS_MISMATCH
    public fun test_native_wrong_coin(
        deployer: &signer,
        token_bridge: &signer
    ) {
        let to = @0x12;
        let fee_recipient = @0x32;
        let amount = 100;
        let fee_amount = 40;

        setup(deployer, token_bridge, to, fee_recipient, amount);

        let token_address = token_hash::get_bytes(&token_hash::derive<MyCoin>());
        let token_chain = state::get_chain_id();
        let to_chain = state::get_chain_id();
        let fee = u256::from_u64(fee_amount);
        let transfer: Transfer = transfer::create(
            1, // ARGH
            u256::from_u64(amount),
            token_address,
            token_chain,
            bcs::to_bytes(&to),
            to_chain,
            fee,
        );
        // FAIL: wrong type argument
        complete_transfer::test<OtherCoin>(&transfer, fee_recipient);
    }

    #[test(
        deployer = @deployer,
        token_bridge = @token_bridge,
    )]
    #[expected_failure(abort_code = 0)] // E_ORIGIN_CHAIN_MISMATCH
    public fun test_native_wrong_origin_address(
        deployer: &signer,
        token_bridge: &signer
    ) {
        let to = @0x12;
        let fee_recipient = @0x32;
        let amount = 100;
        let fee_amount = 40;

        setup(deployer, token_bridge, to, fee_recipient, amount);

        let token_address = token_hash::get_bytes(&token_hash::derive<MyCoin>());
        let token_chain = wormhole::u16::from_u64(10); // FAIL: wrong origin chain (MyCoin is native)
        let to_chain = state::get_chain_id();
        let fee = u256::from_u64(fee_amount);
        let transfer: Transfer = transfer::create(
            1, // ARGH
            u256::from_u64(amount),
            token_address,
            token_chain,
            bcs::to_bytes(&to),
            to_chain,
            fee,
        );
        complete_transfer::test<MyCoin>(&transfer, fee_recipient);
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

        setup(deployer, token_bridge, to, fee_recipient, amount);

        let token_address = x"00000000000000000000000000000000000000000000000000000000deadbeef";
        let token_chain = wormhole::u16::from_u64(2);
        let to_chain = state::get_chain_id();
        let fee = u256::from_u64(fee_amount);
        let transfer: Transfer = transfer::create(
            1, // ARGH
            u256::from_u64(amount),
            token_address,
            token_chain,
            bcs::to_bytes(&to),
            to_chain,
            fee,
        );

        assert!(coin::balance<wrapped_coin::coin::T>(to) == 0, 0);
        assert!(coin::balance<wrapped_coin::coin::T>(fee_recipient) == 0, 0);
        complete_transfer::test<wrapped_coin::coin::T>(&transfer, fee_recipient);
        assert!(coin::balance<wrapped_coin::coin::T>(to) == 60, 0);
        assert!(coin::balance<wrapped_coin::coin::T>(fee_recipient) == 40, 0);
    }
}
