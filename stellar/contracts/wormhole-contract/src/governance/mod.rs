mod action;
mod contract_upgrade;
pub mod guardian_set;
mod set_message_fee;
mod transfer_fees;

pub use action::GovernanceAction;
pub use contract_upgrade::ContractUpgradeAction;
pub use guardian_set::GuardianSetUpgradeAction;
pub use set_message_fee::{SetMessageFeeAction, get_message_fee};
pub use transfer_fees::{TransferFeesAction, get_last_fee_transfer};

use crate::initialize;
use soroban_sdk::{Bytes, Env};
use wormhole_soroban_client::WormholeError;

pub fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
    initialize::require_initialized(&env)?;
    if action::is_governance_vaa_consumed_from_bytes(&env, &vaa_bytes)? {
        Err(WormholeError::GovernanceVAAAlreadyConsumed)
    } else {
        Ok(())
    }
}
