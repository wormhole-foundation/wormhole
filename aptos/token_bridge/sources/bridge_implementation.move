//
// Implementations for wrapped asset creation and token transfers
// (TokenTransferWithPayload, TokenTransfer, CreateWrapped)
//
// TODO: we shouldn't follow the ethereum module layout, it's not great (+ it's
// EVM specific with the "implementation" setup)
module token_bridge::bridge_implementation {
    use 0x1::type_info::{type_of, TypeInfo};
    use 0x1::coin::{Self, initialize, name, symbol, decimals, withdraw};
    use 0x1::account::{create_resource_account};
    use 0x1::signer::{address_of};
    use 0x1::bcs::{to_bytes};
    use 0x1::string::{utf8};

    use std::string;
    use token_bridge::bridge_state::{Self, token_bridge_signer, set_outstanding_bridged, outstanding_bridged, bridge_contracts, set_native_asset};
    //use Wormhole::bridge_structs::{AssetMeta, Transfer, TransferWithPayload};
    use token_bridge::bridge_structs::{Self, create_asset_meta, encode_asset_meta, AssetMeta};
    use token_bridge::utils::{hash_type_info};

    use wormhole::u256::{Self, U256};
    //use wormhole::u128::{U128};
    use wormhole::u32::{U32};
    use wormhole::u16::{U16};
    use wormhole::vaa::{Self, VAA, parse_and_verify};

    use token_bridge::deploy_coin::{deploy_coin};


    // TODO: for functions that do take a signer, we should have an equivalent
    // function that does *not* take a signer, and instead takes explicitly
    // whatever's needed from the signer. In this case, it would be a version of
    // `attest_token` that takes a sufficient amount of coins. Then the signer
    // version could do the withdrawal first and call the other version (which
    // itself should also be public).
    // There are multiple benefits:
    // 1. we have a version of the function that doesn't require a signer
    // 2. structuring the code in this way makes it very clear what the signer
    //    is needed for (since the signer stuff is forced to be written separately)
    //    -- something not that clear in the current version
    public fun attest_token<CoinType>(user: &signer) {
        let payload_id = 0;
        let token_address = hash_type_info<CoinType>();
        if (!bridge_state::is_registered_native_asset(token_address) && !bridge_state::is_wrapped_asset(token_address)) {
            // if native asset is not registered, register it in the reverse look-up map
            set_native_asset(token_address, type_of<CoinType>());
        };
        let token_chain = wormhole::state::get_chain_id();
        let decimals = decimals<CoinType>();
        let symbol = *string::bytes(&symbol<CoinType>());
        let name = *string::bytes(&name<CoinType>());

        let _asset_meta: AssetMeta = create_asset_meta(
            payload_id,
            token_address,
            token_chain,
            decimals,
            symbol,
            name
        );

        let payload:vector<u8> = encode_asset_meta(_asset_meta);
        let nonce = 0;
        let message_fee = wormhole::state::get_message_fee();
        let fee_coins = withdraw(user, message_fee);
        bridge_state::publish_message(
            nonce,
            payload,
            fee_coins
        )
    }

    public entry fun create_wrapped<CoinType>(vaa: vector<u8>): address {
        let vaa = parse_and_verify(vaa);
        let _token_bridge_signer = token_bridge_signer();
        let seed = b"blah"; //TODO - how to set seed?
        let (new_signer, new_cap) = create_resource_account(&_token_bridge_signer, seed);
        let token_address = address_of(&new_signer);
        deploy_coin(&new_signer);
        bridge_state::set_wrapped_asset_signer_capability(to_bytes(&token_address), new_cap);

        let _asset_meta: AssetMeta = bridge_structs::parse_asset_meta(vaa::get_payload(&vaa));
        // TODO - how to initialize a new Coin without the type as input? e.g. create_wrapped<CoinType>...
        // pre-deploy, or figure out what the asset address will be somehow?
        let name = bridge_structs::get_name(&_asset_meta);
        let symbol = bridge_structs::get_symbol(&_asset_meta);
        let decimals = bridge_structs::get_decimals(&_asset_meta);
        let monitor_supply = false;
        let (burn_cap, freeze_cap, mint_cap) = initialize<CoinType>(&new_signer, utf8(name), utf8(symbol), decimals, monitor_supply);
        vaa::destroy(vaa);

        //TODO - store these caps inside of state. For now we just destroy them
        // so the code compiles
        coin::destroy_burn_cap(burn_cap);
        coin::destroy_freeze_cap(freeze_cap);
        coin::destroy_mint_cap(mint_cap);

        token_address
    }

    public entry fun transfer_tokens_with_payload (
        _token: vector<u8>,
        _amount: U256,
        _recipientChain: U16,
        _recipient: vector<u8>,
        _nonce: U32,
        _payload: vector<u8>
    ) {
        //TODO
    }

    /*
     *  @notice Initiate a transfer
     */
    fun transfer_tokens_(_token: TypeInfo, _amount: u128, _arbiterFee: u128) {//returns TransferResult
        // TODO
    }

    fun bridge_out(_token: vector<u8>, _normalized_amount: U256) {
        // TODO
        //let outstanding = outstanding_bridged(token);
        //let lhs = u256::add(outstanding, normalized_amount);
        //assert!(u256::compare(lhs, &(2<<128-1))==1, 0); //LHS is less than RHS
        //setOutstandingBridged(token, u256::add(outstanding, normalized_amount));
    }

    fun bridged_in(token: vector<u8>, normalized_amount: U256) {
        set_outstanding_bridged(token, u256::sub(outstanding_bridged(token), normalized_amount));
    }

    fun verify_bridge_vm(vm: &VAA): bool{
        if (bridge_contracts(vaa::get_emitter_chain(vm)) == vaa::get_emitter_address(vm)) {
            return true
        };
        return false
    }

}
