use cosmwasm_schema::cw_serde;
use cw20::Cw20ReceiveMsg;

#[cw_serde]
pub struct InstantiateMsg {}

#[cw_serde]
pub struct CountResponse {
    pub count: u32,
}

#[cw_serde]
pub enum ExecuteMsg {
    Increment {},
    Reset {},
    Receive(Cw20ReceiveMsg),
}

#[cw_serde]
pub enum ReceiveMsg {
    Increment {},
    Reset {},
}

#[cw_serde]
pub enum QueryMsg {
    GetCount {},
}
