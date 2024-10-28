use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::Binary;

type HumanAddr = String;

#[cw_serde]
pub struct InstantiateMsg {
    pub token_bridge_contract: HumanAddr,
}

#[cw_serde]
pub enum ExecuteMsg {
    CompleteTransferWithPayload { data: Binary },
}

#[cw_serde]
pub struct MigrateMsg {}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(())]
    WrappedRegistry { chain: u16, address: Binary },
}
