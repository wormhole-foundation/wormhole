use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Binary, Uint128};

pub const COMPLETE_TRANSFER_REPLY_ID: u64 = 1;

#[cw_serde]
pub struct InstantiateMsg {
    pub token_bridge_contract: String,
}

#[cw_serde]
pub enum ExecuteMsg {
    /// Submit a VAA to complete a wormhole payload3 token bridge transfer.
    /// This function will:
    /// 1. complete the wormhole token bridge transfer.
    /// 2. Lock the newly minted cw20 tokens.
    /// 3. CreateDenom (if it doesn't already exist)
    /// 4. Mint an equivalent amount of bank tokens using the token factory.
    /// 5. Send the minted bank tokens to the destination address with contract payload if applicable.
    CompleteTransferAndConvert {
        /// VAA to submit. The VAA should be encoded in the standard wormhole
        /// wire format.
        vaa: Binary,
    },

    /// Convert bank tokens into the equivalent (locked) cw20 tokens and trigger a wormhole token bridge transfer.
    /// This function will:
    /// 1. Validate that the bank tokens originated from cw20 tokens that are locked in this contract.
    /// 2. Burn the bank tokens using the token factory.
    /// 3. Unlock the equivalent cw20 tokens.
    /// 4. Cross-call into the wormhole token bridge to initiate a cross-chain transfer with a gateway transfer payload.
    GatewayConvertAndTransfer {
        recipient: Binary,
        chain: u16,
        fee: Uint128,
        nonce: u32,
    },

    /// Convert bank tokens into the equivalent (locked) cw20 tokens and trigger a wormhole token bridge transfer.
    /// This function will:
    /// 1. Validate that the bank tokens originated from cw20 tokens that are locked in this contract.
    /// 2. Burn the bank tokens using the token factory.
    /// 3. Unlock the equivalent cw20 tokens.
    /// 4. Cross-call into the wormhole token bridge to initiate a cross-chain transfer with a gateway transfer-with-payload payload.
    GatewayConvertAndTransferWithPayload {
        contract: Binary,
        chain: u16,
        payload: Binary,
        nonce: u32,
    },

    /// Submit a signed VAA to update the on-chain state.
    SubmitUpdateChainToChannelMap {
        /// VAA to submit. The VAA should be encoded in the standard wormhole
        /// wire format.
        vaa: Binary,
    },
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(ChannelResponse)]
    IbcChannel { chain_id: u16 },
}

#[cw_serde]
pub struct ChannelResponse {
    pub channel: String,
}

#[cw_serde]
pub enum GatewayIbcTokenBridgePayload {
    GatewayTransfer {
        chain: u16,
        recipient: Binary,
        fee: u128,
        nonce: u32,
    },
    GatewayTransferWithPayload {
        chain: u16,
        contract: Binary,
        payload: Binary,
        nonce: u32,
    },
}
