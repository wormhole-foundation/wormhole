mod helpers;

use helpers::{instantiate_contracts};

#[test]
fn basic_init() {
    let (_router, _ibc_wormhole_translator_contract_addr) = instantiate_contracts();
}
