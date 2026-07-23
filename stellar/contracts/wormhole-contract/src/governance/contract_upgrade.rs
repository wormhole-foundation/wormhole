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
    ACTION_CONTRACT_UPGRADE, BytesReader, CONTRACT_UPGRADE_PAYLOAD_MIN_LENGTH, VAA, WormholeError,
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

        if reader.remaining() != 0 {
            return Err(WormholeError::InvalidPayload);
        }

        Ok(ContractUpgradePayload {
            module,
            action,
            chain,
            new_contract_hash,
        })
    }
}

impl ContractUpgradePayload {
    /// Validates the governance header matches expected values for upgrade
    /// action.
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

    fn execute(env: &Env, _vaa: &VAA, payload: &Self::Payload) -> Result<(), WormholeError> {
        env.deployer()
            .update_current_contract_wasm(payload.new_contract_hash.clone());

        ContractUpgradeEvent {
            new_contract_hash: payload.new_contract_hash.clone(),
        }
        .publish(env);

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Wormhole;
    use soroban_sdk::{Bytes, BytesN, Env, vec};
    use wormhole_soroban_client::{CHAIN_ID_STELLAR, MODULE_CORE};

    fn build_payload(env: &Env, module: [u8; 32], action: u8, chain: u16) -> Bytes {
        let mut payload = Bytes::new(env);
        payload.append(&Bytes::from_array(env, &module));
        payload.push_back(action);
        payload.append(&Bytes::from_slice(env, &chain.to_be_bytes()));
        payload.append(&Bytes::from_array(env, &[0xAB; 32]));
        payload
    }

    #[test]
    fn test_contract_upgrade_payload_rejects_short_payload() {
        let env = Env::default();
        let short = Bytes::from_slice(&env, &[0u8; 10]);
        let res = ContractUpgradePayload::try_from((&env, &short));
        assert_eq!(res, Err(WormholeError::InvalidPayload));
    }

    #[test]
    fn test_contract_upgrade_validate_header_errors() {
        let env = Env::default();
        let bad_module = ContractUpgradePayload {
            module: BytesN::from_array(&env, &[7u8; 32]),
            action: ACTION_CONTRACT_UPGRADE,
            chain: CHAIN_ID_STELLAR,
            new_contract_hash: BytesN::from_array(&env, &[0u8; 32]),
        };
        assert_eq!(
            bad_module.validate(),
            Err(WormholeError::InvalidGovernanceModule)
        );

        let bad_action = ContractUpgradePayload {
            module: BytesN::from_array(&env, &MODULE_CORE),
            action: 9,
            chain: CHAIN_ID_STELLAR,
            new_contract_hash: BytesN::from_array(&env, &[0u8; 32]),
        };
        assert_eq!(
            bad_action.validate(),
            Err(WormholeError::InvalidGovernanceAction)
        );

        let bad_chain = ContractUpgradePayload {
            module: BytesN::from_array(&env, &MODULE_CORE),
            action: ACTION_CONTRACT_UPGRADE,
            chain: 2,
            new_contract_hash: BytesN::from_array(&env, &[0u8; 32]),
        };
        assert_eq!(
            bad_chain.validate(),
            Err(WormholeError::InvalidGovernanceChain)
        );
    }

    #[test]
    fn test_contract_upgrade_payload_parse_success() {
        let env = Env::default();
        let payload = build_payload(&env, MODULE_CORE, ACTION_CONTRACT_UPGRADE, CHAIN_ID_STELLAR);
        let parsed = ContractUpgradePayload::try_from((&env, &payload)).unwrap();
        assert_eq!(parsed.action, ACTION_CONTRACT_UPGRADE);
        assert_eq!(parsed.chain, CHAIN_ID_STELLAR);
        assert_eq!(parsed.module, BytesN::from_array(&env, &MODULE_CORE));
        assert_eq!(
            parsed.new_contract_hash,
            BytesN::from_array(&env, &[0xAB; 32])
        );
    }

    #[test]
    #[should_panic(expected = "InvalidInput")]
    fn test_contract_upgrade_execute_rejects_invalid_wasm_hash() {
        let env = Env::default();
        let contract_id = env.register(
            Wormhole,
            (
                vec![&env, BytesN::<20>::from_array(&env, &[0u8; 20])],
                BytesN::<32>::from_array(&env, &wormhole_soroban_client::GOVERNANCE_EMITTER),
            ),
        );

        let wasm_hash = env.deployer().upload_contract_wasm(Bytes::from_slice(
            &env,
            &[0x00, 0x61, 0x73, 0x6d, 1, 0, 0, 0],
        ));
        let payload = ContractUpgradePayload {
            module: BytesN::from_array(&env, &MODULE_CORE),
            action: ACTION_CONTRACT_UPGRADE,
            chain: CHAIN_ID_STELLAR,
            new_contract_hash: wasm_hash,
        };
        let vaa = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: vec![&env],
            timestamp: 0,
            nonce: 0,
            emitter_chain: 1,
            emitter_address: BytesN::from_array(&env, &[0u8; 32]),
            sequence: 0,
            consistency_level: wormhole_soroban_client::ConsistencyLevel::Confirmed,
            payload: Bytes::new(&env),
        };

        env.as_contract(&contract_id, || {
            let _ = ContractUpgradeAction::execute(&env, &vaa, &payload);
        });
    }
}
