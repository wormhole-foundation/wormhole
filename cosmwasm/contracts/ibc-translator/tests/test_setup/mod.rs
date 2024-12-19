use serde::de::DeserializeOwned;
use std::marker::PhantomData;

use cosmwasm_std::{
    from_json,
    testing::{mock_env, BankQuerier, MockQuerierCustomHandlerResult, MockStorage},
    Addr, Api, Binary, CanonicalAddr, Coin, ContractResult, CustomQuery, Empty, Env, OwnedDeps,
    Querier, QuerierResult, QueryRequest, RecoverPubkeyError, StdError, StdResult, SystemError,
    SystemResult, VerificationError, WasmQuery,
};
use wormhole_bindings::WormholeQuery;

pub const WORMHOLE_CONTRACT_ADDR: &str =
    "wormhole1yw4wv2zqg9xkn67zvq3azye0t8h0x9kgyg3d53jym24gxt49vdys6s8h7a";
pub const WORMHOLE_USER_ADDR: &str = "wormhole1vhkm2qv784rulx8ylru0zpvyvw3m3cy99e6wy0";
pub const WORMHOLE_CONTRACT_ADDR_BYTES: [u8; 32] = [
    0x23, 0xaa, 0xe6, 0x28, 0x40, 0x41, 0x4d, 0x69, 0xeb, 0xc2, 0x60, 0x23, 0xd1, 0x13, 0x2f, 0x59,
    0xee, 0xf3, 0x16, 0xc8, 0x22, 0x22, 0xda, 0x46, 0x44, 0xda, 0xaa, 0x83, 0x2e, 0xa5, 0x63, 0x49,
];
pub const WORMHOLE_USER_ADDR_BYTES: [u8; 20] = [
    0x65, 0xed, 0xb5, 0x01, 0x9e, 0x3d, 0x47, 0xcf, 0x98, 0xe4, 0xf8, 0xf8, 0xf1, 0x05, 0x84, 0x63,
    0xa3, 0xb8, 0xe0, 0x85,
];

// Custom API mock implementation for testing.
// The custom impl helps us with correct addr_validate, addr_canonicalize, and addr_humanize methods for Wormchain.
#[derive(Clone)]
pub struct CustomApi {
    contract_addr: String,
    user_addr: String,
    contract_addr_bin: Binary,
    user_addr_bin: Binary,
}

impl CustomApi {
    pub fn new(
        contract_addr: &str,
        user_addr: &str,
        contract_addr_bytes: [u8; 32],
        user_addr_bytes: [u8; 20],
    ) -> Self {
        CustomApi {
            contract_addr: contract_addr.to_string(),
            user_addr: user_addr.to_string(),
            contract_addr_bin: Binary::from(contract_addr_bytes),
            user_addr_bin: Binary::from(user_addr_bytes),
        }
    }
}

impl Api for CustomApi {
    fn addr_validate(&self, input: &str) -> StdResult<Addr> {
        if input == self.contract_addr {
            return Ok(Addr::unchecked(self.contract_addr.clone()));
        }

        if input == self.user_addr {
            return Ok(Addr::unchecked(self.user_addr.clone()));
        }

        Err(StdError::GenericErr {
            msg: "case not found".to_string(),
        })
    }

    fn addr_canonicalize(&self, input: &str) -> StdResult<CanonicalAddr> {
        if input == self.contract_addr {
            return Ok(CanonicalAddr(self.contract_addr_bin.clone()));
        }

        if input == self.user_addr {
            return Ok(CanonicalAddr(self.user_addr_bin.clone()));
        }

        Err(StdError::GenericErr {
            msg: "case not found".to_string(),
        })
    }

    fn addr_humanize(&self, canonical: &CanonicalAddr) -> StdResult<Addr> {
        if *canonical == self.contract_addr_bin {
            return Ok(Addr::unchecked(self.contract_addr.clone()));
        }

        if *canonical == self.user_addr_bin {
            return Ok(Addr::unchecked(self.user_addr.clone()));
        }

        Err(StdError::GenericErr {
            msg: "case not found".to_string(),
        })
    }

    fn secp256k1_verify(
        &self,
        message_hash: &[u8],
        signature: &[u8],
        public_key: &[u8],
    ) -> Result<bool, VerificationError> {
        Ok(cosmwasm_crypto::secp256k1_verify(
            message_hash,
            signature,
            public_key,
        )?)
    }

    fn secp256k1_recover_pubkey(
        &self,
        message_hash: &[u8],
        signature: &[u8],
        recovery_param: u8,
    ) -> Result<Vec<u8>, RecoverPubkeyError> {
        let pubkey =
            cosmwasm_crypto::secp256k1_recover_pubkey(message_hash, signature, recovery_param)?;
        Ok(pubkey.to_vec())
    }

    fn ed25519_verify(
        &self,
        message: &[u8],
        signature: &[u8],
        public_key: &[u8],
    ) -> Result<bool, VerificationError> {
        Ok(cosmwasm_crypto::ed25519_verify(
            message, signature, public_key,
        )?)
    }

    fn ed25519_batch_verify(
        &self,
        messages: &[&[u8]],
        signatures: &[&[u8]],
        public_keys: &[&[u8]],
    ) -> Result<bool, VerificationError> {
        Ok(cosmwasm_crypto::ed25519_batch_verify(
            messages,
            signatures,
            public_keys,
        )?)
    }

    fn debug(&self, message: &str) {
        println!("{message}");
    }
}

#[allow(dead_code)]
pub fn default_custom_mock_deps() -> OwnedDeps<MockStorage, CustomApi, MockQuerier, Empty> {
    OwnedDeps {
        storage: MockStorage::default(),
        api: CustomApi::new(
            WORMHOLE_CONTRACT_ADDR,
            WORMHOLE_USER_ADDR,
            WORMHOLE_CONTRACT_ADDR_BYTES,
            WORMHOLE_USER_ADDR_BYTES,
        ),
        querier: MockQuerier::default(),
        custom_query_type: PhantomData,
    }
}

#[allow(dead_code)]
pub fn execute_custom_mock_deps() -> OwnedDeps<MockStorage, CustomApi, MockQuerier, WormholeQuery> {
    OwnedDeps {
        storage: MockStorage::default(),
        api: CustomApi::new(
            WORMHOLE_CONTRACT_ADDR,
            WORMHOLE_USER_ADDR,
            WORMHOLE_CONTRACT_ADDR_BYTES,
            WORMHOLE_USER_ADDR_BYTES,
        ),
        querier: MockQuerier::default(),
        custom_query_type: PhantomData,
    }
}

#[allow(dead_code)]
pub fn mock_env_custom_contract(contract_addr: impl Into<String>) -> Env {
    let mut env = mock_env();
    env.contract.address = Addr::unchecked(contract_addr);
    env
}

/// MockQuerier holds an immutable table of bank balances
/// and configurable handlers for Wasm queries and custom queries.
pub struct MockQuerier<C: DeserializeOwned = Empty> {
    bank: BankQuerier,
    #[cfg(feature = "staking")]
    staking: StakingQuerier,
    wasm: WasmQuerier,
    #[cfg(feature = "stargate")]
    ibc: IbcQuerier,
    /// A handler to handle custom queries. This is set to a dummy handler that
    /// always errors by default. Update it via `with_custom_handler`.
    ///
    /// Use box to avoid the need of another generic type
    custom_handler: Box<dyn for<'a> Fn(&'a C) -> MockQuerierCustomHandlerResult>,
}

impl<C: DeserializeOwned> MockQuerier<C> {
    pub fn new(balances: &[(&str, &[Coin])]) -> Self {
        MockQuerier {
            bank: BankQuerier::new(balances),
            #[cfg(feature = "staking")]
            staking: StakingQuerier::default(),
            wasm: WasmQuerier::default(),
            #[cfg(feature = "stargate")]
            ibc: IbcQuerier::default(),
            // strange argument notation suggested as a workaround here: https://github.com/rust-lang/rust/issues/41078#issuecomment-294296365
            custom_handler: Box::from(|_: &_| -> MockQuerierCustomHandlerResult {
                SystemResult::Ok(ContractResult::Ok(Binary::from_base64("e30=").unwrap()))
            }),
        }
    }

    // set a new balance for the given address and return the old balance
    #[allow(dead_code)]
    pub fn update_balance(
        &mut self,
        addr: impl Into<String>,
        balance: Vec<Coin>,
    ) -> Option<Vec<Coin>> {
        self.bank.update_balance(addr, balance)
    }

    #[cfg(feature = "staking")]
    pub fn update_staking(
        &mut self,
        denom: &str,
        validators: &[crate::query::Validator],
        delegations: &[crate::query::FullDelegation],
    ) {
        self.staking = StakingQuerier::new(denom, validators, delegations);
    }

    #[cfg(feature = "stargate")]
    pub fn update_ibc(&mut self, port_id: &str, channels: &[IbcChannel]) {
        self.ibc = IbcQuerier::new(port_id, channels);
    }

    pub fn update_wasm<WH>(&mut self, handler: WH)
    where
        WH: 'static + Fn(&WasmQuery) -> QuerierResult,
    {
        self.wasm.update_handler(handler)
    }

    #[allow(dead_code)]
    pub fn with_custom_handler<CH>(mut self, handler: CH) -> Self
    where
        CH: 'static + Fn(&C) -> MockQuerierCustomHandlerResult,
    {
        self.custom_handler = Box::from(handler);
        self
    }
}

impl Default for MockQuerier {
    fn default() -> Self {
        MockQuerier::new(&[])
    }
}

impl<C: CustomQuery + DeserializeOwned> Querier for MockQuerier<C> {
    fn raw_query(&self, bin_request: &[u8]) -> QuerierResult {
        let request: QueryRequest<C> = match from_json(bin_request) {
            Ok(v) => v,
            Err(e) => {
                return SystemResult::Err(SystemError::InvalidRequest {
                    error: format!("Parsing query request: {e}"),
                    request: bin_request.into(),
                })
            }
        };
        self.handle_query(&request)
    }
}

impl<C: CustomQuery + DeserializeOwned> MockQuerier<C> {
    pub fn handle_query(&self, request: &QueryRequest<C>) -> QuerierResult {
        match &request {
            QueryRequest::Bank(bank_query) => self.bank.query(bank_query),
            QueryRequest::Custom(custom_query) => (*self.custom_handler)(custom_query),
            #[cfg(feature = "staking")]
            QueryRequest::Staking(staking_query) => self.staking.query(staking_query),
            QueryRequest::Wasm(msg) => self.wasm.query(msg),
            #[cfg(feature = "stargate")]
            QueryRequest::Stargate { .. } => SystemResult::Err(SystemError::UnsupportedRequest {
                kind: "Stargate".to_string(),
            }),
            #[cfg(feature = "stargate")]
            QueryRequest::Ibc(msg) => self.ibc.query(msg),
            //_ => SystemResult::Err(SystemError::UnsupportedRequest {
            //    kind: "Unknown".to_string(),
            //}),
            _ => SystemResult::Ok(ContractResult::Ok(Binary::default())),
        }
    }
}

struct WasmQuerier {
    /// A handler to handle Wasm queries. This is set to a dummy handler that
    /// always errors by default. Update it via `with_custom_handler`.
    ///
    /// Use box to avoid the need of generic type.
    handler: Box<dyn for<'a> Fn(&'a WasmQuery) -> QuerierResult>,
}

impl WasmQuerier {
    fn new(handler: Box<dyn for<'a> Fn(&'a WasmQuery) -> QuerierResult>) -> Self {
        Self { handler }
    }

    fn update_handler<WH>(&mut self, handler: WH)
    where
        WH: 'static + Fn(&WasmQuery) -> QuerierResult,
    {
        self.handler = Box::from(handler)
    }

    fn query(&self, request: &WasmQuery) -> QuerierResult {
        (*self.handler)(request)
    }
}

impl Default for WasmQuerier {
    fn default() -> Self {
        let handler = Box::from(|request: &WasmQuery| -> QuerierResult {
            let err = match request {
                WasmQuery::Smart { contract_addr, .. } => SystemError::NoSuchContract {
                    addr: contract_addr.clone(),
                },
                WasmQuery::Raw { contract_addr, .. } => SystemError::NoSuchContract {
                    addr: contract_addr.clone(),
                },
                WasmQuery::ContractInfo { contract_addr, .. } => SystemError::NoSuchContract {
                    addr: contract_addr.clone(),
                },
                #[cfg(feature = "cosmwasm_1_2")]
                WasmQuery::CodeInfo { code_id, .. } => {
                    SystemError::NoSuchCode { code_id: *code_id }
                }
                _ => SystemError::Unknown {},
            };
            SystemResult::Err(err)
        });
        Self::new(handler)
    }
}
