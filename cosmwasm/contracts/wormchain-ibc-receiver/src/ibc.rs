use cosmwasm_std::{
    entry_point, from_slice, to_binary, Binary, ContractResult, DepsMut, Env,
    Ibc3ChannelOpenResponse, IbcBasicResponse, IbcChannelCloseMsg, IbcChannelConnectMsg,
    IbcChannelOpenMsg, IbcChannelOpenResponse, IbcPacketAckMsg, IbcPacketReceiveMsg,
    IbcPacketTimeoutMsg, IbcReceiveResponse, Response, StdError, StdResult,
};

use crate::msg::WormholeIbcPacketMsg;

// Implementation of IBC protocol
// Implements 6 entry points that are required for the x/wasm runtime to bind a port for this contract
// https://github.com/CosmWasm/cosmwasm/blob/main/IBC.md#writing-new-protocols

pub const IBC_APP_VERSION: &str = "ibc-wormhole-v1";

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
    deps: DepsMut,
    _env: Env,
    msg: IbcPacketReceiveMsg,
) -> StdResult<IbcReceiveResponse> {
    // decode the packet as WormholeIbcPacketMsg::Publish and add the response attributes to the IbcReceiveResponse
    // put this in a closure so we can convert all error responses into acknowledgements
    (|| {
        let packet = msg.packet;
        // which local channel did this packet come on
        let channel_id = packet.dest.channel_id;
        let wormhole_msg: WormholeIbcPacketMsg = from_slice(&packet.data)?;
        match wormhole_msg {
            WormholeIbcPacketMsg::Publish { msg: publish_msg } => {
                receive_publish(deps, channel_id, publish_msg)
            }
        }
    })()
    .or_else(|e| {
        // we try to capture all app-level errors and convert them into
        // acknowledgement packets that contain an error code.
        let acknowledgement = encode_ibc_error(format!("invalid packet: {}", e));
        Ok(IbcReceiveResponse::new()
            .set_ack(acknowledgement)
            .add_attribute("action", "ibc_packet_ack"))
    })
}

fn receive_publish(
    _deps: DepsMut,
    channel_id: String,
    publish_msg: Response,
) -> StdResult<IbcReceiveResponse> {
    // send the ack and emit the message
    let acknowledgement = to_binary(&ContractResult::<()>::Ok(()))?;
    Ok(IbcReceiveResponse::new()
        .set_ack(acknowledgement)
        .add_attribute("action", "receive_publish")
        .add_attribute("channel_id", channel_id)
        .add_attributes(publish_msg.attributes))
}

// this encode an error or error message into a proper acknowledgement to the recevier
fn encode_ibc_error(msg: impl Into<String>) -> Binary {
    // this cannot error, unwrap to keep the interface simple
    to_binary(&ContractResult::<()>::Err(msg.into())).unwrap()
}

/// 5. Acknowledging a packet. Called when the other chain successfully receives a packet from us.
/// Never should be called as this contract never sends packets
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
/// Never should be called as this contract never sends packets
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

#[cfg(test)]
mod tests {
    // test opening channel
    // 1. succeed - happy path
    // 2. fail - channel version is not correct
    // 3. fail - counterparty channel version is not correct

    // test connecting
    // 1. success - it logs the right attributes and values

    // test closing
    // 1. success - it logs the right attributes and values

    // test packet receive
    // 1. failure - deserializing packet failure
    // 2. failure - receive_publish failure (use a mock for this?)
    // 3. success - happy path

    // test receive_publish
    // 1. success - it logs the right attributes and values
}
