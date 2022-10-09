module token_bridge::wrapped {
//   public entry fun create_wrapped_coin<T>(vaa: vector<u8>, witness: T) {
//         //let vaa = token_bridge_vaa::parse_verify_and_replay_protect(vaa);
//         let asset_meta: AssetMeta = asset_meta::parse(vaa::destroy(vaa));

//         let native_token_address = asset_meta::get_token_address(&asset_meta);
//         let native_token_chain = asset_meta::get_token_chain(&asset_meta);
//         let origin_info = state::create_origin_info(native_token_chain, native_token_address);

//     }

}