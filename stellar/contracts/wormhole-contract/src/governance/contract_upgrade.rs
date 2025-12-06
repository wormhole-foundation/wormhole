use crate::governance::action::{
    GovernanceAction, parse_governance_header, validate_governance_header,
};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env, contractevent};
use wormhole_soroban_client::*;

/// Event published when a contract upgrade is executed.
///
/// Topics: ["wormhole_core", "upgrade"]
/// - "wormhole_core": Namespace for all core contract governance/lifecycle events
/// - "upgrade": Event type for contract upgrades
#[contractevent(topics = ["wormhole_core", "upgrade"], data_format = "single-value")]
struct ContractUpgradeEvent {
    new_contract_hash: BytesN<32>,
}

#[derive(Debug, PartialEq)]
pub struct ContractUpgradePayload {
    pub module: BytesN<32>,
    pub action: u8,
    pub chain: u16,
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
    fn validate(&self) -> Result<(), WormholeError> {
        validate_governance_header(
            &self.module,
            self.action,
            self.chain,
            ACTION_CONTRACT_UPGRADE,
        )
    }
}

pub struct ContractUpgradeAction;

impl GovernanceAction for ContractUpgradeAction {
    type Payload = ContractUpgradePayload;

    fn validate_payload(_env: &Env, payload: &Self::Payload) -> Result<(), WormholeError> {
        payload.validate()
    }

    fn execute(
        env: &Env,
        _vaa: &crate::vaa::VAA,
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
