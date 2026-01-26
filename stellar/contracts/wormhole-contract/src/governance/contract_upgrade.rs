//! Contract upgrade governance action (`ACTION_CONTRACT_UPGRADE`, value 1).
//!
//! Allows guardians to upgrade the contract WASM by providing the hash of a
//! new contract version. The contract is its own admin, so upgrades require
//! guardian consensus rather than a separate admin key.

use crate::governance::action::{
    GovernanceAction, parse_governance_header, validate_governance_header,
};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env, contractevent};
use wormhole_soroban_client::{
    ACTION_CONTRACT_UPGRADE, BytesReader, CONTRACT_UPGRADE_PAYLOAD_MIN_LENGTH, WormholeError, VAA
};

/// Emitted when a contract upgrade is executed successfully.
///
/// Topics: `["wormhole_core", "upgrade"]`
#[contractevent(topics = ["wormhole_core", "upgrade"], data_format = "single-value")]
struct ContractUpgradeEvent {
    /// WASM hash of the newly deployed contract version.
    new_contract_hash: BytesN<32>,
}

/// Parsed payload for contract upgrade governance action.
#[derive(Debug, PartialEq)]
pub struct ContractUpgradePayload {
    /// Module identifier (must be "Core").
    pub module: BytesN<32>,
    /// Governance action ID from the VAA header.
    ///
    /// Must equal `ACTION_CONTRACT_UPGRADE` (1); any other value is rejected by
    /// `validate_governance_header`.
    pub action: u8,
    /// Target chain (0 for all, 61 for Stellar).
    pub chain: u16,
    /// WASM hash of the new contract to deploy.
    pub new_contract_hash: BytesN<32>,
}

impl<'a> TryFrom<(&'a Env, &'a Bytes)> for ContractUpgradePayload {
    type Error = WormholeError;

    fn try_from(value: (&'a Env, &'a Bytes)) -> Result<Self, Self::Error> {
        let (env, payload) = value;

        if payload.len() < CONTRACT_UPGRADE_PAYLOAD_MIN_LENGTH {
            return Err(WormholeError::InvalidPayload);
        }

        let mut reader = BytesReader::new(payload);
        let (module, action, chain) = parse_governance_header(env, &mut reader)?;
        let new_contract_hash = reader.read_bytes_n::<32>()?;

        Ok(ContractUpgradePayload {
            module,
            action,
            chain,
            new_contract_hash,
        })
    }
}

impl ContractUpgradePayload {
    /// Validates the governance header matches expected values for upgrade action.
    fn validate(&self) -> Result<(), WormholeError> {
        validate_governance_header(
            &self.module,
            self.action,
            self.chain,
            ACTION_CONTRACT_UPGRADE,
        )
    }
}

/// Governance action handler for contract upgrades.
pub struct ContractUpgradeAction;

impl GovernanceAction for ContractUpgradeAction {
    type Payload = ContractUpgradePayload;

    fn validate_payload(_env: &Env, payload: &Self::Payload) -> Result<(), WormholeError> {
        payload.validate()
    }

    fn execute(
        env: &Env,
        _vaa: &VAA,
        payload: &Self::Payload,
    ) -> Result<(), WormholeError> {
        env.deployer()
            .update_current_contract_wasm(payload.new_contract_hash.clone());

        ContractUpgradeEvent {
            new_contract_hash: payload.new_contract_hash.clone(),
        }
        .publish(env);

        Ok(())
    }
}
