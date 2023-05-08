// mod helpers;

// use ibc_wormhole_translator::{
//     msg::{AllChainChannelsResponse, QueryMsg},
// };

// use helpers::{CHANNEL_18, instantiate_contracts};

// #[test]
// fn query_chain_channels() {
//     let (router, ibc_wormhole_translator_contract_addr) = instantiate_contracts();

//     let query_msg = QueryMsg::AllChainChannels {};
//     let query_response: AllChainChannelsResponse = router
//         .wrap()
//         .query_wasm_smart(
//             ibc_wormhole_translator_contract_addr,
//             &query_msg,
//         )
//         .unwrap();
//     assert_eq!(query_response.chain_channels.len(), 1);
//     assert_eq!(query_response.chain_channels[0].0.to_string(), CHANNEL_18.to_string());
//     assert_eq!(query_response.chain_channels[0].1, 18);
// }
