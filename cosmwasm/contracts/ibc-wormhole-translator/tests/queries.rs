mod helpers;

use ibc_wormhole_translator::{
    msg::{AllChainChannelsResponse, QueryMsg},
};

use helpers::{instantiate_contracts};

static CHANNEL_0: &str = "Y2hhbm5lbC0w";

#[test]
fn query_chain_channels() {
    let (router, ibc_wormhole_translator_contract_addr) = instantiate_contracts();

    let query_msg = QueryMsg::AllChainChannels {};
    let query_response: AllChainChannelsResponse = router
        .wrap()
        .query_wasm_smart(
            ibc_wormhole_translator_contract_addr,
            &query_msg,
        )
        .unwrap();
    assert_eq!(query_response.chain_channels.len(), 1);
    assert_eq!(query_response.chain_channels[0].0.to_string(), CHANNEL_0.to_string());
    assert_eq!(query_response.chain_channels[0].1, 18);
}
