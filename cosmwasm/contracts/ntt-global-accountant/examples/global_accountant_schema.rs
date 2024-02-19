use cosmwasm_schema::write_api;

use cosmwasm_std::Empty;
use ntt_global_accountant::msg::{ExecuteMsg, QueryMsg};

fn main() {
    write_api! {
        instantiate: Empty,
        execute: ExecuteMsg,
        query: QueryMsg,
    }
}
