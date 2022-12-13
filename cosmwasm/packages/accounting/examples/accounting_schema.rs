use cosmwasm_schema::write_api;

use accounting::msg::Instantiate;

fn main() {
    write_api! {
        instantiate: Instantiate,
        // execute: ExecuteMsg,
        // query: QueryMsg,
    }
}
