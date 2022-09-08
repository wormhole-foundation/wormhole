// Implementations for wrapped asset creation and token transfers
// (TokenTransferWithPayload, TokenTransfer, CreateWrapped)
//
// TODO: we shouldn't follow the ethereum module layout, it's not great (+ it's
// EVM specific with the "implementation" setup)
module token_bridge::bridge_implementation {
    use aptos_framework::type_info::{TypeInfo};
    use aptos_framework::account::{create_resource_account};
    use aptos_framework::signer::{address_of};

    use token_bridge::bridge_state as state;
    use token_bridge::asset_meta::{Self, AssetMeta};
    use token_bridge::deploy_coin::{deploy_coin};

    use wormhole::u256::{Self, U256};
    use wormhole::u32::{U32};
    use wormhole::u16::{U16};
    use wormhole::vaa::{Self, VAA, parse_and_verify};

    // this function is called before create_wrapped_coin
    public entry fun create_wrapped_coin_type(vaa: vector<u8>): address {
        // TODO: verify VAA is from a known emitter + replay protection
        let vaa = parse_and_verify(vaa);
        let asset_meta:AssetMeta = asset_meta::parse(vaa::get_payload(&vaa));
        let seed = asset_meta::create_seed(&asset_meta);

        //create resource account
        let token_bridge_signer = state::token_bridge_signer();
        let (new_signer, new_cap) = create_resource_account(&token_bridge_signer, seed);

        let token_address = asset_meta::get_token_address(&asset_meta);
        let token_chain = asset_meta::get_token_chain(&asset_meta);
        let origin_info = state::create_origin_info(token_address, token_chain);

        deploy_coin(&new_signer);
        state::set_wrapped_asset_signer_capability(origin_info, new_cap);
        vaa::destroy(vaa);

        // return address of the new signer
        address_of(&new_signer)
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
        state::set_outstanding_bridged(token, u256::sub(state::outstanding_bridged(token), normalized_amount));
    }

    fun verify_bridge_vm(vm: &VAA): bool{
        if (state::get_registered_emitter(vaa::get_emitter_chain(vm)) == vaa::get_emitter_address(vm)) {
            return true
        };
        return false
    }

}
