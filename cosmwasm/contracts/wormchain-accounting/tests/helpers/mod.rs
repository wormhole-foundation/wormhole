#![allow(dead_code)]

use accounting::state::{account, transfer, Account, Kind, Modification, Transfer};
use cosmwasm_std::{
    testing::{MockApi, MockStorage},
    to_binary, Addr, Binary, Coin, Empty, StdResult, Uint128, Uint256,
};
use cw_multi_test::{
    App, AppBuilder, AppResponse, BankKeeper, ContractWrapper, Executor, WasmKeeper,
};
use wormchain_accounting::{
    msg::{
        AllAccountsResponse, AllModificationsResponse, AllPendingTransfersResponse,
        AllTransfersResponse, ExecuteMsg, Instantiate, InstantiateMsg, QueryMsg,
    },
    state,
};
use wormhole_bindings::{fake, WormholeQuery};

mod fake_tokenbridge;

pub struct Contract {
    addr: Addr,
    app: FakeApp,
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
        signature: wormhole_bindings::Signature,
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

    pub fn modify_balance(
        &mut self,
        modification: Binary,
        guardian_set_index: u32,
        signatures: Vec<wormhole_bindings::Signature>,
    ) -> anyhow::Result<AppResponse> {
        self.app.execute_contract(
            Addr::unchecked(USER),
            self.addr(),
            &ExecuteMsg::ModifyBalance {
                modification,
                guardian_set_index,
                signatures,
            },
            &[],
        )
    }

    pub fn upgrade_contract(
        &mut self,
        upgrade: Binary,
        guardian_set_index: u32,
        signatures: Vec<wormhole_bindings::Signature>,
    ) -> anyhow::Result<AppResponse> {
        self.app.execute_contract(
            Addr::unchecked(ADMIN),
            self.addr(),
            &ExecuteMsg::UpgradeContract {
                upgrade,
                guardian_set_index,
                signatures,
            },
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

    pub fn query_transfer(&self, key: transfer::Key) -> StdResult<transfer::Data> {
        self.app
            .wrap()
            .query_wasm_smart(self.addr(), &QueryMsg::Transfer(key))
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
        self.app
            .wrap()
            .query_wasm_smart(self.addr(), &QueryMsg::PendingTransfer(key))
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

pub fn create_accounts(count: usize) -> Vec<Account> {
    let mut out = Vec::with_capacity(count * count);
    for i in 0..count {
        for j in 0..count {
            let key = account::Key::new(i as u16, j as u16, [i as u8; 32].into());
            let balance = Uint256::from(j as u128).into();
            out.push(Account { key, balance });
        }
    }

    out
}

pub fn create_transfers(count: usize) -> Vec<Transfer> {
    let mut out = Vec::with_capacity(count);
    for i in 0..count {
        let key = transfer::Key::new(i as u16, [i as u8; 32].into(), i as u64);
        let data = transfer::Data {
            amount: Uint256::from(i as u128),
            token_chain: i as u16,
            token_address: [i as u8; 32].into(),
            recipient_chain: i as u16,
        };

        out.push(Transfer { key, data });
    }

    out
}

pub fn create_modifications(count: usize) -> Vec<Modification> {
    let mut out = Vec::with_capacity(count);
    for i in 0..count {
        let m = Modification {
            sequence: i as u64,
            chain_id: i as u16,
            token_chain: i as u16,
            token_address: [i as u8; 32].into(),
            kind: if i % 2 == 0 { Kind::Add } else { Kind::Sub },
            amount: Uint256::from(i as u128),
            reason: format!("{i}"),
        };
        out.push(m);
    }

    out
}

pub fn proper_instantiate(
    accounts: Vec<Account>,
    transfers: Vec<Transfer>,
    modifications: Vec<Modification>,
) -> (fake::WormholeKeeper, Contract) {
    let wh = fake::WormholeKeeper::new();
    let mut app = fake_app(wh.clone());

    let tokenbridge_id = app.store_code(Box::new(ContractWrapper::new_with_empty(
        fake_tokenbridge::execute,
        fake_tokenbridge::instantiate,
        fake_tokenbridge::query,
    )));

    let accounting_id = app.store_code(Box::new(ContractWrapper::new(
        wormchain_accounting::contract::execute,
        wormchain_accounting::contract::instantiate,
        wormchain_accounting::contract::query,
    )));

    let tokenbridge_addr = app
        .instantiate_contract(
            tokenbridge_id,
            Addr::unchecked(ADMIN),
            &Empty {},
            &[],
            "tokenbridge",
            None,
        )
        .unwrap()
        .into();

    let instantiate = to_binary(&Instantiate {
        tokenbridge_addr,
        accounts,
        transfers,
        modifications,
    })
    .unwrap();

    let signatures = wh.sign(&instantiate);
    let msg = InstantiateMsg {
        instantiate,
        guardian_set_index: wh.guardian_set_index(),
        signatures,
    };

    // We want the contract to be able to upgrade itself, which means we have to set the contract
    // as its own admin.  So we have a bit of a catch-22 where we need to know the contract
    // address to register it but we need to register it to get its address.  The hacky solution
    // here is to rely on the internal details of the test framework to figure out what the
    // address of the contract is going to be and then use that.
    //
    // TODO: Figure out a better way to do this.  One option is to do something like:
    //
    // ```
    // let mut data = app.contract_data(&addr).unwrap();
    // data.admin = Some(addr.clone());
    // app.init_modules(|router, _, storage| router.wasm.save_contract(storage, &addr, &data))
    //     .unwrap();
    // ```
    //
    // Unfortunately, the `wasm` field of `router` is private to the `cw-multi-test` crate so we
    // can't use it here.  Maybe something to bring up with upstream.
    let addr = app
        .instantiate_contract(
            accounting_id,
            Addr::unchecked(ADMIN),
            &msg,
            &[],
            "accounting",
            Some("contract1".into()),
        )
        .unwrap();

    (wh, Contract { addr, app })
}
