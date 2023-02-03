#![allow(dead_code)]

use accountant::state::{account, transfer, Modification};
use cosmwasm_schema::cw_serde;
use cosmwasm_std::{
    testing::{MockApi, MockStorage},
    Addr, Binary, Coin, Empty, StdError, StdResult, Uint128,
};
use cw_multi_test::{
    App, AppBuilder, AppResponse, BankKeeper, ContractWrapper, Executor, WasmKeeper,
};
use global_accountant::{
    msg::{
        AllAccountsResponse, AllModificationsResponse, AllPendingTransfersResponse,
        AllTransfersResponse, BatchTransferStatusResponse, ChainRegistrationResponse, ExecuteMsg,
        MigrateMsg, MissingObservationsResponse, ModifyBalance, QueryMsg, TransferStatus,
        SUBMITTED_OBSERVATIONS_PREFIX,
    },
    state,
};
use serde::Serialize;
use wormhole::{
    token::{Action, GovernancePacket},
    vaa::{Body, Header, Signature},
    Address, Chain, Vaa,
};
use wormhole_bindings::{fake, WormholeQuery};

#[cw_serde]
pub struct TransferResponse {
    pub data: transfer::Data,
    pub digest: Binary,
}

pub struct Contract {
    addr: Addr,
    app: FakeApp,
    code_id: u64,
}

impl Contract {
    pub fn addr(&self) -> Addr {
        self.addr.clone()
    }

    pub fn app(&self) -> &FakeApp {
        &self.app
    }

    pub fn app_mut(&mut self) -> &mut FakeApp {
        &mut self.app
    }

    pub fn submit_observations(
        &mut self,
        observations: Binary,
        guardian_set_index: u32,
        signature: Signature,
    ) -> anyhow::Result<AppResponse> {
        self.app.execute_contract(
            Addr::unchecked(USER),
            self.addr(),
            &ExecuteMsg::SubmitObservations {
                observations,
                guardian_set_index,
                signature,
            },
            &[],
        )
    }

    pub fn modify_balance(&mut self, modification: Binary) -> anyhow::Result<AppResponse> {
        let migrate_msg = MigrateMsg {
            modifications: vec![ModifyBalance { modification }],
        };
        // let serialized_msg = serde_json::to_string(&migrate_msg).unwrap();
        // let serialized_msg = migrate_msg.serialize()?;
        // since migrate is an authenticated method, we reuse
        // that with the same code_id to pass sensitive messages
        // like ModifyBalance
        self.app.migrate_contract(
            Addr::unchecked(ADMIN),
            self.addr(),
            &migrate_msg,
            self.code_id,
        )
    }

    pub fn upgrade_contract(&mut self, code_id: u64) -> anyhow::Result<AppResponse> {
        self.code_id = code_id;
        let migrate_msg = MigrateMsg {
            modifications: vec![],
        };
        self.app.migrate_contract(
            Addr::unchecked(ADMIN),
            self.addr(),
            &migrate_msg,
            self.code_id,
        )
    }

    pub fn submit_vaas(&mut self, vaas: Vec<Binary>) -> anyhow::Result<AppResponse> {
        self.app.execute_contract(
            Addr::unchecked(ADMIN),
            self.addr(),
            &ExecuteMsg::SubmitVAAs { vaas },
            &[],
        )
    }

    pub fn query_balance(&self, key: account::Key) -> StdResult<account::Balance> {
        self.app
            .wrap()
            .query_wasm_smart(self.addr(), &QueryMsg::Balance(key))
    }

    pub fn query_all_accounts(
        &self,
        start_after: Option<account::Key>,
        limit: Option<u32>,
    ) -> StdResult<AllAccountsResponse> {
        self.app
            .wrap()
            .query_wasm_smart(self.addr(), &QueryMsg::AllAccounts { start_after, limit })
    }

    pub fn query_transfer_status(&self, key: transfer::Key) -> StdResult<TransferStatus> {
        self.app
            .wrap()
            .query_wasm_smart(self.addr(), &QueryMsg::TransferStatus(key))
    }

    pub fn query_batch_transfer_status(
        &self,
        keys: Vec<transfer::Key>,
    ) -> StdResult<BatchTransferStatusResponse> {
        self.app
            .wrap()
            .query_wasm_smart(self.addr(), &QueryMsg::BatchTransferStatus(keys))
    }

    pub fn query_transfer(&self, key: transfer::Key) -> StdResult<TransferResponse> {
        self.query_transfer_status(key.clone()).and_then(|status| {
            if let TransferStatus::Committed { data, digest } = status {
                Ok(TransferResponse { data, digest })
            } else {
                Err(StdError::not_found(format!("transfer for key {key}")))
            }
        })
    }

    pub fn query_all_transfers(
        &self,
        start_after: Option<transfer::Key>,
        limit: Option<u32>,
    ) -> StdResult<AllTransfersResponse> {
        self.app
            .wrap()
            .query_wasm_smart(self.addr(), &QueryMsg::AllTransfers { start_after, limit })
    }

    pub fn query_pending_transfer(&self, key: transfer::Key) -> StdResult<Vec<state::Data>> {
        self.query_transfer_status(key.clone()).and_then(|status| {
            if let TransferStatus::Pending(state) = status {
                Ok(state)
            } else {
                Err(StdError::not_found(format!(
                    "pending transfer for key {key}"
                )))
            }
        })
    }

    pub fn query_all_pending_transfers(
        &self,
        start_after: Option<transfer::Key>,
        limit: Option<u32>,
    ) -> StdResult<AllPendingTransfersResponse> {
        self.app.wrap().query_wasm_smart(
            self.addr(),
            &QueryMsg::AllPendingTransfers { start_after, limit },
        )
    }

    pub fn query_modification(&self, sequence: u64) -> StdResult<Modification> {
        self.app
            .wrap()
            .query_wasm_smart(self.addr(), &QueryMsg::Modification { sequence })
    }

    pub fn query_all_modifications(
        &self,
        start_after: Option<u64>,
        limit: Option<u32>,
    ) -> StdResult<AllModificationsResponse> {
        self.app.wrap().query_wasm_smart(
            self.addr(),
            &QueryMsg::AllModifications { start_after, limit },
        )
    }

    pub fn query_chain_registration(&self, chain: u16) -> StdResult<ChainRegistrationResponse> {
        self.app
            .wrap()
            .query_wasm_smart(self.addr(), &QueryMsg::ChainRegistration { chain })
    }

    pub fn query_missing_observations(
        &self,
        guardian_set: u32,
        index: u8,
    ) -> StdResult<MissingObservationsResponse> {
        self.app.wrap().query_wasm_smart(
            self.addr(),
            &QueryMsg::MissingObservations {
                guardian_set,
                index,
            },
        )
    }
}

const USER: &str = "USER";
const ADMIN: &str = "ADMIN";
const NATIVE_DENOM: &str = "denom";

pub type FakeApp =
    App<BankKeeper, MockApi, MockStorage, fake::WormholeKeeper, WasmKeeper<Empty, WormholeQuery>>;

fn fake_app(wh: fake::WormholeKeeper) -> FakeApp {
    AppBuilder::new_custom()
        .with_custom(wh)
        .build(|router, _, storage| {
            router
                .bank
                .init_balance(
                    storage,
                    &Addr::unchecked(USER),
                    vec![Coin {
                        denom: NATIVE_DENOM.to_string(),
                        amount: Uint128::new(1),
                    }],
                )
                .unwrap();
        })
}

pub fn proper_instantiate() -> (fake::WormholeKeeper, Contract) {
    let wh = fake::WormholeKeeper::new();
    let mut app = fake_app(wh.clone());

    let code_id = app.store_code(Box::new(
        ContractWrapper::new(
            global_accountant::contract::execute,
            global_accountant::contract::instantiate,
            global_accountant::contract::query,
        )
        .with_migrate(global_accountant::contract::migrate),
    ));

    // We used to rely on the contract being able to upgrade itself,
    // but then we decided it was easiest to rely on wrapped VAA autentication
    // methods of the x/wormhole module.  So we do not need to worry
    // about upgrade authentication in the contract itself.
    // We will use a fixed admin address so the contract does remain upgradeable.
    let addr = app
        .instantiate_contract(
            code_id,
            Addr::unchecked(ADMIN),
            &Empty {},
            &[],
            "accountant",
            Some(ADMIN.into()),
        )
        .unwrap();

    (wh, Contract { addr, app, code_id })
}

pub fn sign_vaa_body<P: Serialize>(wh: &fake::WormholeKeeper, body: Body<P>) -> (Vaa<P>, Binary) {
    let data = serde_wormhole::to_vec(&body).unwrap();
    let signatures = wh.sign(&data);

    let header = Header {
        version: 1,
        guardian_set_index: wh.guardian_set_index(),
        signatures,
    };

    let v = (header, body).into();
    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();

    (v, data)
}

pub fn sign_observations(wh: &fake::WormholeKeeper, observations: &[u8]) -> Vec<Signature> {
    let mut prepended =
        Vec::with_capacity(SUBMITTED_OBSERVATIONS_PREFIX.len() + observations.len());
    prepended.extend_from_slice(SUBMITTED_OBSERVATIONS_PREFIX);
    prepended.extend_from_slice(observations);

    let mut signatures = wh.sign(&prepended);
    signatures.sort_by_key(|s| s.index);

    signatures
}

pub fn register_emitters(wh: &fake::WormholeKeeper, contract: &mut Contract, count: usize) {
    for i in 0..count {
        let body = Body {
            timestamp: i as u32,
            nonce: i as u32,
            emitter_chain: Chain::Solana,
            emitter_address: wormhole::GOVERNANCE_EMITTER,
            sequence: i as u64,
            consistency_level: 0,
            payload: GovernancePacket {
                chain: Chain::Any,
                action: Action::RegisterChain {
                    chain: (i as u16).into(),
                    emitter_address: Address([i as u8; 32]),
                },
            },
        };

        let (_, data) = sign_vaa_body(wh, body);
        contract.submit_vaas(vec![data]).unwrap();
    }
}
