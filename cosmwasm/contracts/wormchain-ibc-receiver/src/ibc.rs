use anyhow::{bail, ensure};
use cosmwasm_std::{
    entry_point, from_json, to_json_binary, Attribute, Binary, ContractResult, DepsMut, Env,
    Ibc3ChannelOpenResponse, IbcBasicResponse, IbcChannelCloseMsg, IbcChannelConnectMsg,
    IbcChannelOpenMsg, IbcChannelOpenResponse, IbcPacketAckMsg, IbcPacketReceiveMsg,
    IbcPacketTimeoutMsg, IbcReceiveResponse, StdError, StdResult,
};

use crate::msg::WormholeIbcPacketMsg;

// Implementation of IBC protocol
// Implements 6 entry points that are required for the x/wasm runtime to bind a port for this contract
// https://github.com/CosmWasm/cosmwasm/blob/main/IBC.md#writing-new-protocols

pub const IBC_APP_VERSION: &str = "ibc-wormhole-v1";

/// 1. Opening a channel. Step 1 of handshake. Combines ChanOpenInit and ChanOpenTry from the spec.
///    The only valid action of the contract is to accept the channel or reject it.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn ibc_channel_open(
    _deps: DepsMut,
    _env: Env,
    msg: IbcChannelOpenMsg,
) -> StdResult<IbcChannelOpenResponse> {
    let channel = msg.channel();

    if channel.version.as_str() != IBC_APP_VERSION {
        return Err(StdError::generic_err(format!(
            "Must set version to `{IBC_APP_VERSION}`"
        )));
    }

    if let Some(counter_version) = msg.counterparty_version() {
        if counter_version != IBC_APP_VERSION {
            return Err(StdError::generic_err(format!(
                "Counterparty version must be `{IBC_APP_VERSION}`"
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
    let connection_id = &channel.connection_id;

    Ok(IbcBasicResponse::new()
        .add_attribute("action", "ibc_connect")
        .add_attribute("connection_id", connection_id))
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
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn ibc_packet_receive(
    _deps: DepsMut,
    _env: Env,
    msg: IbcPacketReceiveMsg,
) -> StdResult<IbcReceiveResponse> {
    handle_packet_receive(msg).or_else(|e| {
        // we try to capture all app-level errors and convert them into
        // acknowledgement packets that contain an error code.
        let acknowledgement = encode_ibc_error(format!("invalid packet: {e}"));
        Ok(IbcReceiveResponse::new()
            .set_ack(acknowledgement)
            .add_attribute("action", "ibc_packet_ack"))
    })
}

/// Decode the IBC packet as WormholeIbcPacketMsg::Publish and take appropriate action
fn handle_packet_receive(msg: IbcPacketReceiveMsg) -> Result<IbcReceiveResponse, anyhow::Error> {
    let packet = msg.packet;
    // which local channel did this packet come on
    let channel_id = packet.dest.channel_id;
    let wormhole_msg: WormholeIbcPacketMsg = from_json(&packet.data)?;
    match wormhole_msg {
        WormholeIbcPacketMsg::Publish { msg: publish_attrs } => {
            receive_publish(channel_id, publish_attrs)
        }
    }
}

const EXPECTED_WORMHOLE_IBC_EVENT_ATTRS: [&str; 6] = [
    "message.message",
    "message.sender",
    "message.chain_id",
    "message.nonce",
    "message.sequence",
    "message.block_time",
];

fn receive_publish(
    channel_id: String,
    publish_attrs: Vec<Attribute>,
) -> Result<IbcReceiveResponse, anyhow::Error> {
    // check the attributes are what we expect from wormhole
    ensure!(
        publish_attrs.len() == EXPECTED_WORMHOLE_IBC_EVENT_ATTRS.len(),
        "number of received attributes does not match number of expected"
    );

    for key in EXPECTED_WORMHOLE_IBC_EVENT_ATTRS {
        let mut matched = false;
        for attr in &publish_attrs {
            if key == attr.key {
                matched = true;
                break;
            }
        }
        if !matched {
            bail!(
                "expected attribute unmmatched in received attributes: {}",
                key
            );
        }
    }

    // send the ack and emit the message with the attributes from the wormhole message
    let acknowledgement = to_json_binary(&ContractResult::<()>::Ok(()))?;
    Ok(IbcReceiveResponse::new()
        .set_ack(acknowledgement)
        .add_attribute("action", "receive_publish")
        .add_attribute("channel_id", channel_id)
        .add_attributes(publish_attrs))
}

// this encode an error or error message into a proper acknowledgement to the recevier
fn encode_ibc_error(msg: impl Into<String>) -> Binary {
    // this cannot error, unwrap to keep the interface simple
    to_json_binary(&ContractResult::<()>::Err(msg.into())).unwrap()
}

/// 5. Acknowledging a packet. Called when the other chain successfully receives a packet from us.
///    Never should be called as this contract never sends packets
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn ibc_packet_ack(
    _deps: DepsMut,
    _env: Env,
    _msg: IbcPacketAckMsg,
) -> StdResult<IbcBasicResponse> {
    Err(StdError::generic_err(
        "ack should never be called as this contract never sends packets",
    ))
}

/// 6. Timing out a packet. Called when the packet was not recieved on the other chain before the timeout.
///    Never should be called as this contract never sends packets
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn ibc_packet_timeout(
    _deps: DepsMut,
    _env: Env,
    _msg: IbcPacketTimeoutMsg,
) -> StdResult<IbcBasicResponse> {
    Err(StdError::generic_err(
        "timeout should never be called as this contract never sends packets",
    ))
}
