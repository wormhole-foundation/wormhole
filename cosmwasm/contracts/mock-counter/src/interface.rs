use cosmwasm_std::{Addr, StdError, StdResult};
use cw_multi_test::{App, ContractWrapper, Executor};

use crate::{CountResponse, ExecuteMsg, InstantiateMsg, QueryMsg};

pub struct CounterContract(Addr);

impl CounterContract {
    pub fn addr(&self) -> &Addr {
        &self.0
    }

    pub fn store_code(app: &mut App) -> u64 {
        let contract = ContractWrapper::new(
            crate::contract::execute,
            crate::contract::instantiate,
            crate::contract::query,
        );
        app.store_code(Box::new(contract))
    }

    #[track_caller]
    pub fn instantiate(app: &mut App, code_id: u64, sender: &str, label: &str) -> StdResult<Self> {
        let msg = InstantiateMsg {};
        let addr = app
            .instantiate_contract(code_id, Addr::unchecked(sender), &msg, &[], label, None)
            .unwrap();
        Ok(CounterContract(addr))
    }

    #[track_caller]
    pub fn increment(&self, app: &mut App, sender: &str) -> StdResult<()> {
        let msg = ExecuteMsg::Increment {};
        let _ = app
            .execute_contract(Addr::unchecked(sender), self.0.clone(), &msg, &[])
            .map_err(|err| StdError::generic_err(err.to_string()))?;

        Ok(())
    }

    #[track_caller]
    pub fn reset(&self, app: &mut App, sender: &str) -> StdResult<()> {
        let msg = ExecuteMsg::Reset {};
        let _ = app
            .execute_contract(Addr::unchecked(sender), self.0.clone(), &msg, &[])
            .map_err(|err| StdError::generic_err(err.to_string()))?;

        Ok(())
    }

    #[track_caller]
    pub fn query_count(&self, app: &App) -> StdResult<u32> {
        let msg = QueryMsg::GetCount {};
        let result: CountResponse = app.wrap().query_wasm_smart(self.0.clone(), &msg)?;
        Ok(result.count)
    }
}
