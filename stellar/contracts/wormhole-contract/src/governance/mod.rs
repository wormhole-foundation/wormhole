mod action;
pub mod guardian_set;

pub use action::GovernanceAction;
pub use guardian_set::GuardianSetUpgradeAction;

use crate::initialize;
use soroban_sdk::{Bytes, Env};
use wormhole_soroban_client::WormholeError;

pub fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<bool, WormholeError> {
    initialize::require_initialized(&env)?;
    action::is_governance_vaa_consumed_from_bytes(&env, &vaa_bytes)
}
