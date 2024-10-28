use cosmwasm_schema::write_api;
use cw_wormhole::msg::{InstantiateMsg, QueryMsg};
use wormhole_ibc::msg::ExecuteMsg;

fn main() {
    write_api! {
        instantiate: InstantiateMsg,
        execute: ExecuteMsg,
        query: QueryMsg,
    }
}