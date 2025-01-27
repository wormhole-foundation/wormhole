use crate::{
    msg::GatewayIbcTokenBridgePayload,
    state::{CHAIN_TO_CHANNEL_MAP, CURRENT_TRANSFER, CW_DENOMS},
};
use anybuf::Anybuf;
use anyhow::{ensure, Context};
use cosmwasm_std::{
    from_json, to_json_binary, Binary, CosmosMsg::Stargate, Deps, DepsMut, Env, QueryRequest,
    Reply, Response, SubMsg, WasmQuery,
};
use cw20::TokenInfoResponse;
use cw20_base::msg::QueryMsg as TokenQuery;
use cw_token_bridge::msg::CompleteTransferResponse;
use wormhole_bindings::tokenfactory::{DenomUnit, Metadata, TokenFactoryMsg, TokenMsg};

pub fn handle_complete_transfer_reply(
    deps: DepsMut,
    env: Env,
    msg: Reply,
) -> Result<Response<TokenFactoryMsg>, anyhow::Error> {
    // we should only be replying on success
    ensure!(
        msg.result.is_ok(),
        "msg result is not okay, we should never get here"
    );

    let res_data_raw = cw_utils::parse_reply_execute_data(msg)
        .context("failed to parse protobuf reply response_data")?
        .data
        .context("no data in the response, we should never get here")?;
    let res_data: CompleteTransferResponse =
        from_json(&res_data_raw).context("failed to deserialize response data")?;
    let contract_addr = res_data
        .contract
        .context("no contract in response, we should never get here")?;

    // load interim state
    let transfer_info = CURRENT_TRANSFER
        .load(deps.storage)
        .context("failed to load current transfer from storage")?;

    // delete interim state
    CURRENT_TRANSFER.remove(deps.storage);

    // deserialize payload into the type we expect
    let payload: GatewayIbcTokenBridgePayload = serde_json_wasm::from_slice(&transfer_info.payload)
        .context("failed to deserialize transfer payload")?;

    match payload {
        GatewayIbcTokenBridgePayload::GatewayTransfer {
            chain,
            recipient,
            fee: _,
            nonce: _,
        } => {
            let recipient_decoded = String::from_utf8(recipient.to_vec())
                .context(format!("failed to convert {recipient} to utf8 string"))?;
            convert_cw20_to_bank_and_send(
                deps,
                env,
                recipient_decoded,
                res_data.amount.into(),
                contract_addr,
                chain,
                None,
            )
        }
        GatewayIbcTokenBridgePayload::GatewayTransferWithPayload {
            chain,
            contract,
            payload,
            nonce: _,
        } => {
            let contract_decoded = String::from_utf8(contract.to_vec())
                .context(format!("failed to convert {contract} to utf8 string"))?;
            convert_cw20_to_bank_and_send(
                deps,
                env,
                contract_decoded,
                res_data.amount.into(),
                contract_addr,
                chain,
                Some(payload),
            )
        }
    }
}
pub fn convert_cw20_to_bank_and_send(
    deps: DepsMut,
    env: Env,
    recipient: String,
    amount: u128,
    cw20_contract_addr: String,
    chain_id: u16,
    payload: Option<Binary>,
) -> Result<Response<TokenFactoryMsg>, anyhow::Error> {
    deps.api
        .addr_validate(&cw20_contract_addr)
        .context(format!("invalid contract address {cw20_contract_addr}"))?;

    // convert contract address into base64
    let subdenom = contract_addr_to_base58(deps.as_ref(), cw20_contract_addr.clone())?;
    // format the token factory denom
    let tokenfactory_denom = "factory/".to_string()
        + env.contract.address.to_string().as_ref()
        + "/"
        + subdenom.as_ref();

    let mut response: Response<TokenFactoryMsg> = Response::new();

    // check contract storage see if we've created a denom for this cw20 token yet
    // if we haven't created the denom, then create the denom
    if !CW_DENOMS.has(deps.storage, cw20_contract_addr.clone()) {
        // call into the cw20 contract to get the token's metadata
        let request = QueryRequest::Wasm(WasmQuery::Smart {
            contract_addr: cw20_contract_addr.clone(),
            msg: to_json_binary(&TokenQuery::TokenInfo {})?,
        });
        let token_info: TokenInfoResponse = deps.querier.query(&request)?;

        // Populate token factory token's metadata from cw20 token's metadata
        let tf_denom_unit_base = DenomUnit {
            denom: tokenfactory_denom.clone(),
            exponent: 0,
            aliases: vec![],
        };
        let tf_scaled_denom = "wormhole/".to_string()
            + subdenom.as_str()
            + "/"
            + token_info.decimals.to_string().as_str();
        let tf_denom_unit_scaled = DenomUnit {
            denom: tf_scaled_denom.clone(),
            exponent: u32::from(token_info.decimals),
            aliases: vec![],
        };

        let mut symbol = token_info.symbol;
        if symbol.is_empty() {
            symbol.clone_from(&tf_scaled_denom);
        }
        let tf_description =
            token_info.name.clone() + ", " + symbol.as_str() + ", " + tokenfactory_denom.as_str();
        let tf_metadata = Metadata {
            description: Some(tf_description),
            base: Some(tokenfactory_denom.clone()),
            denom_units: vec![tf_denom_unit_base, tf_denom_unit_scaled],
            display: Some(tf_scaled_denom),
            name: Some(token_info.name),
            symbol: Some(symbol),
        };

        // call into token factory to create the denom
        let create_denom = SubMsg::new(TokenMsg::CreateDenom {
            subdenom,
            metadata: Some(tf_metadata),
        });
        response = response.add_submessage(create_denom);

        // add the contract_addr => tokenfactory denom to storage
        CW_DENOMS
            .save(deps.storage, cw20_contract_addr, &tokenfactory_denom)
            .context("failed to save contract_addr => tokenfactory denom to storage")?;
    }

    // add calls to mint and send bank tokens
    response = response.add_message(TokenMsg::MintTokens {
        denom: tokenfactory_denom.clone(),
        amount,
        mint_to_address: env.contract.address.to_string(),
    });

    let channel = CHAIN_TO_CHANNEL_MAP
        .load(deps.storage, chain_id)
        .context("chain id does not have an allowed channel")?;

    let payload_decoded = match payload {
        Some(payload) => String::from_utf8(payload.to_vec())
            .context(format!("failed to convert {payload} to utf8 string"))?,
        None => "".to_string(),
    };

    // Create MsgTransfer protobuf message for Stargate
    // https://github.com/cosmos/ibc-go/blob/main/proto/ibc/applications/transfer/v1/tx.proto#L27
    // TimeoutTimestamp is 1 year from now, which should effectively protect against packets timing out when sending from Gateway
    let ibc_msg_transfer = Anybuf::new()
        .append_string(1, "transfer") // source port
        .append_string(2, channel) // source channel
        .append_message(
            3,
            &Anybuf::new()
                .append_string(1, tokenfactory_denom)
                .append_string(2, amount.to_string()),
        ) // Token
        .append_string(4, env.contract.address) // sender
        .append_string(5, recipient) // receiver
        .append_message(6, &Anybuf::new().append_uint64(1, 0).append_uint64(2, 0)) // TimeoutHeight
        .append_uint64(7, env.block.time.plus_days(365).nanos()) // TimeoutTimestamp
        .append_string(8, payload_decoded); // Memo

    response = response.add_message(Stargate {
        type_url: "/ibc.applications.transfer.v1.MsgTransfer".to_string(),
        value: ibc_msg_transfer.into_vec().into(),
    });
    Ok(response)
}

// Base58 allows the subdenom to be a maximum of 44 bytes (max subdenom length) for up to a 32 byte address
pub fn contract_addr_to_base58(deps: Deps, contract_addr: String) -> Result<String, anyhow::Error> {
    // convert the contract address into bytes
    let contract_addr_bytes = deps.api.addr_canonicalize(&contract_addr).context(format!(
        "could not canonicalize contract address {contract_addr}"
    ))?;
    let base_58_addr = bs58::encode(contract_addr_bytes.as_slice()).into_string();
    Ok(base_58_addr)
}
