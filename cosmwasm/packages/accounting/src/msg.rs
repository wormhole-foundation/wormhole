use cosmwasm_schema::cw_serde;

use crate::state::{Account, Modification, Transfer};

#[cw_serde]
pub struct Instantiate {
    pub accounts: Vec<Account>,
    pub transfers: Vec<Transfer>,
    pub modifications: Vec<Modification>,
}
