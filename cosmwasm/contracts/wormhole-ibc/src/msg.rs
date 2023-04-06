use cosmwasm_schema::cw_serde;
use cosmwasm_std::Response;

/// This is the message we send over the IBC channel
#[cw_serde]
pub enum WormholeIbcPacketMsg {
    Publish { msg: Response },
}
