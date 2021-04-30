use anchor_lang::{prelude::*, solana_program};

use crate::{
    accounts,
    anchor_bridge::Bridge,
    types::{BridgeConfig, Index},
    PublishMessage,
};

pub fn publish_message(
    bridge: &mut Bridge,
    ctx: Context<PublishMessage>,
) -> ProgramResult {
    Ok(())
}
