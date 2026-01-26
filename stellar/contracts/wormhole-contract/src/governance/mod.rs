//! Governance action processing for the Wormhole Core contract.
//!
//! Implements the four governance actions defined by Wormhole.
//!
//! Action IDs are encoded in the VAA governance header `action` byte and
//! validated by `validate_governance_header` in each action module.
//! Mappings are:
//! - Contract upgrade (`ACTION_CONTRACT_UPGRADE`, 1)
//! - Guardian set upgrade (`ACTION_GUARDIAN_SET_UPGRADE`, 2)
//! - Set message fee (`ACTION_SET_MESSAGE_FEE`, 3)
//! - Transfer fees (`ACTION_TRANSFER_FEES`, 4)
//!
//! All actions require VAAs signed by a quorum of guardians and originating
//! from the governance chain (Solana) and emitter address.

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

/// Checks if a governance VAA has been consumed (replay protection).
///
/// Returns `Ok(())` if not consumed, `Err(GovernanceVAAAlreadyConsumed)` if already used.
pub fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
    initialize::require_initialized(&env)?;
    if action::is_governance_vaa_consumed_from_bytes(&env, &vaa_bytes)? {
        Err(WormholeError::GovernanceVAAAlreadyConsumed)
    } else {
        Ok(())
    }
}
