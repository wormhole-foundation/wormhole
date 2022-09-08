// Implementations for wrapped asset creation and token transfers
// (TokenTransferWithPayload, TokenTransfer, CreateWrapped)
//
// TODO: we shouldn't follow the ethereum module layout, it's not great (+ it's
// EVM specific with the "implementation" setup)
module token_bridge::bridge_implementation {
    use aptos_framework::type_info::{type_of, TypeInfo};
    use aptos_framework::coin::{Self, Coin, name, symbol, decimals, withdraw};
    use aptos_framework::aptos_coin::{AptosCoin};
    use aptos_framework::account::{create_resource_account};
    use aptos_framework::signer::{address_of};
    use aptos_framework::bcs::{to_bytes};
    use aptos_framework::vector::{Self};

    use std::string;
    use token_bridge::bridge_state::{Self as state, token_bridge_signer, set_outstanding_bridged, outstanding_bridged, set_native_asset};
    //use Wormhole::bridge_structs::{AssetMeta, Transfer, TransferWithPayload};
    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::utils::{hash_type_info};

    use wormhole::u256::{Self, U256};
    //use wormhole::u128::{U128};
    use wormhole::u32::{U32};
    use wormhole::u16::{U16};
    use wormhole::vaa::{Self, VAA, parse_and_verify};
    //use wormhole::serialize::{serialize_vector, serialize_u16};

    use token_bridge::deploy_coin::{deploy_coin};

    const E_COIN_IS_NOT_INITIALIZED: u64 = 0;

    public fun attest_token_with_signer<CoinType>(user: &signer): u64 {
        let message_fee = wormhole::state::get_message_fee();
        let fee_coins = withdraw<AptosCoin>(user, message_fee);
        attest_token<CoinType>(fee_coins)
    }

    public fun attest_token<CoinType>(fee_coins: Coin<AptosCoin>): u64 {
        // you've can't attest an uninitialized token
        // TODO - throw error if attempt to attest wrapped token?
        assert!(coin::is_coin_initialized<CoinType>(), E_COIN_IS_NOT_INITIALIZED);
        let payload_id = 0;
        let token_address = hash_type_info<CoinType>();
        assert!(vector::length<u8>(&token_address)==32, 0);
        if (!state::is_registered_native_asset(token_address) && !state::is_wrapped_asset(token_address)) {
            // if native asset is not registered, register it in the reverse look-up map
            set_native_asset(token_address, type_of<CoinType>());
        };
        let token_chain = wormhole::state::get_chain_id();
        let decimals = decimals<CoinType>();
        let symbol = *string::bytes(&symbol<CoinType>());
        // TODO - left pad to be 32 bytes?
        let name = *string::bytes(&name<CoinType>());
        let asset_meta: AssetMeta = asset_meta::create(
            payload_id,
            token_address,
            token_chain,
            decimals,
            symbol,
            name
        );
        let payload:vector<u8> = asset_meta::encode(asset_meta);
        let nonce = 0;
        state::publish_message(
            nonce,
            payload,
            fee_coins
        )
    }

    // this function is called before create_wrapped_coin
    public entry fun create_wrapped_coin_type(vaa: vector<u8>): address {
        let vaa = parse_and_verify(vaa);
        let _asset_meta:AssetMeta = asset_meta::parse(vaa::get_payload(&vaa));
        let seed = asset_meta::create_seed(&_asset_meta);

        //create resource account
        let _token_bridge_signer = token_bridge_signer();
        let (new_signer, new_cap) = create_resource_account(&_token_bridge_signer, seed);
        let token_address = address_of(&new_signer);
        deploy_coin(&new_signer);
        state::set_wrapped_asset_signer_capability(to_bytes(&token_address), new_cap);
        vaa::destroy(vaa);
        token_address
    }

    public entry fun complete_transfer(vaa: vector<u8>) {
        let vaa = parse_and_verify(vaa);
        vaa::destroy(vaa);
        //TODO
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
     * assume we can transfer both wrapped tokens and
     */
    fun transfer_tokens_(
        _token: TypeInfo,
        _amount: u128,
        _arbiterFee: u128
    ) {//returns TransferResult
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
        if (state::get_registered_emitter(vaa::get_emitter_chain(vm)) == vaa::get_emitter_address(vm)) {
            return true
        };
        return false
    }

}
