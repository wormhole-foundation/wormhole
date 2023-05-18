use cosmwasm_std::{
    entry_point, DepsMut, Env, Ibc3ChannelOpenResponse, IbcBasicResponse, IbcChannelCloseMsg,
    IbcChannelConnectMsg, IbcChannelOpenMsg, IbcChannelOpenResponse, IbcPacketAckMsg,
    IbcPacketReceiveMsg, IbcPacketTimeoutMsg, IbcReceiveResponse, StdError, StdResult,
};

use std::str;

// Implementation of IBC protocol
// Implements 6 entry points that are required for the x/wasm runtime to bind a port for this contract
// https://github.com/CosmWasm/cosmwasm/blob/main/IBC.md#writing-new-protocols

pub const IBC_APP_VERSION: &str = "ibc-wormhole-v1";

/// packets live one year
pub const PACKET_LIFETIME: u64 = 31_536_000;

/// 1. Opening a channel. Step 1 of handshake. Combines ChanOpenInit and ChanOpenTry from the spec.
/// The only valid action of the contract is to accept the channel or reject it.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn ibc_channel_open(
    _deps: DepsMut,
    _env: Env,
    msg: IbcChannelOpenMsg,
) -> StdResult<IbcChannelOpenResponse> {
    let channel = msg.channel();

    if channel.version.as_str() != IBC_APP_VERSION {
        return Err(StdError::generic_err(format!(
            "Must set version to `{}`",
            IBC_APP_VERSION
        )));
    }

    if let Some(counter_version) = msg.counterparty_version() {
        if counter_version != IBC_APP_VERSION {
            return Err(StdError::generic_err(format!(
                "Counterparty version must be `{}`",
                IBC_APP_VERSION
            )));
        }
    }

    // We return the version we need (which could be different than the counterparty version)
    Ok(Some(Ibc3ChannelOpenResponse {
        version: IBC_APP_VERSION.to_string(),
    }))
}

/// 2. Step 2 of handshake. Combines ChanOpenAck and ChanOpenConfirm from the spec.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn ibc_channel_connect(
    _deps: DepsMut,
    _env: Env,
    msg: IbcChannelConnectMsg,
) -> StdResult<IbcBasicResponse> {
    let channel = msg.channel();
    let channel_id = &channel.endpoint.channel_id;

    Ok(IbcBasicResponse::new()
        .add_attribute("action", "ibc_connect")
        .add_attribute("channel_id", channel_id))
}

/// 3. Closing a channel - whether due to an IBC error, at our request, or at the request of the other side.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn ibc_channel_close(
    _deps: DepsMut,
    _env: Env,
    _msg: IbcChannelCloseMsg,
) -> StdResult<IbcBasicResponse> {
    Err(StdError::generic_err("user cannot close channel"))
}

/// 4. Receiving a packet.
/// Never should be called as the other side never sends packets
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn ibc_packet_receive(
    _deps: DepsMut,
    _env: Env,
    _msg: IbcPacketReceiveMsg,
) -> StdResult<IbcReceiveResponse> {
    Err(StdError::generic_err(
        "receive should never be called as this contract should never receive packets",
    ))
}

/// 5. Acknowledging a packet. Called when the other chain successfully receives a packet from us.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn ibc_packet_ack(
    _deps: DepsMut,
    _env: Env,
    _msg: IbcPacketAckMsg,
) -> StdResult<IbcBasicResponse> {
    Ok(IbcBasicResponse::new().add_attribute("action", "ibc_packet_ack"))
}

/// 6. Timing out a packet. Called when the packet was not recieved on the other chain before the timeout.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn ibc_packet_timeout(
    _deps: DepsMut,
    _env: Env,
    _msg: IbcPacketTimeoutMsg,
) -> StdResult<IbcBasicResponse> {
    Ok(IbcBasicResponse::new().add_attribute("action", "ibc_packet_timeout"))
}
