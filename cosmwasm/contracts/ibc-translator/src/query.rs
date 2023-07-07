use cosmwasm_std::{Deps, StdResult};

use crate::{msg::ChannelResponse, state::CHAIN_TO_CHANNEL_MAP};

pub fn query_ibc_channel(deps: Deps, chain_id: u16) -> StdResult<ChannelResponse> {
    let channel = CHAIN_TO_CHANNEL_MAP.load(deps.storage, chain_id)?;

    Ok(ChannelResponse { channel })
}
