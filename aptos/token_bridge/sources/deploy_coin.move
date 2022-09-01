// This module is for dynamically deploying a module/CoinType on-chain,
// as opposed to submitting a tx from off-chain to publish a module containing the CoinType.
//
// Specifically, we wish to dynamically deploy the the following module
//
// module deployer::coin {
//      struct T has key {}
// }
//
// where deployer is a fixed address, but which will be spliced out on-chain.
//
// We create a Move package [INSERT_LINK_HERE], compile it using "aptos move compile --save-metadata", obtain the
// package_metadata.bcs and coin.mv files, then upload them on-chain in byte format (see the deployCoin function below).
//
// We replace the deployer address embedded in the source code with the new deployer address and call publish_package_txn
// to publish the code at the new deployer's account.
//
//
// TODO: find out if we need to update the source_digest in the package metadata as well

module token_bridge::deploy_coin {
    use 0x1::signer::{Self};
    use 0x1::vector::{Self};
    use 0x1::code::{publish_package_txn};
    use 0x1::bcs::{Self};

    public entry fun deploy_coin(deployer: &signer) {
        let addr = signer::address_of(deployer);
        let addr_bytes = bcs::to_bytes(&addr);
        let code = x"a11ceb0b05000000050100020202040706130819200a390500000001080004636f696e01540b64756d6d795f6669656c64";
        vector::append(&mut code, addr_bytes);
        vector::append(&mut code, x"000201020100");
        let metadata_serialized: vector<u8> = x"0b57726170706564436f696e01000000000000000040433035424637374641463734303832443639354442393334414133353235343246354144423132414533363641334641393032303734434232433731324637399f021f8b08000000000002ffb5923d6bc330108677fd8a43596b3b2db443a04369e9563234a5430845b22e8eb0f58124bb94d2ffde93e3902590c99b4ef77e3c086dbda85bd1e08e5961101e817f06e13daa67a72d670386a89dcdf7cb7259de72d6fb2608855fde75bafec98bda192f92961d72c6169bf5cb7a055e5b98bc1124ee5d4030425b8b0902762822b2ad428f56a1ad35c61d7bf2c9c5d74014df2eb414fc0b8d4eb9e090928fabaaa2f1d0cb92fa2a91c54527649c8e35359424e037107ba974c8c6e3cab801abfd2978d29f6772041cb25ce1407c1cfed81b59de93eab49c8f238f451c4b2e328cef3137c4f1f21ac5c6b568e78648b9e422c3023eeeee1faef47b67139a822c63664f8e7356fe7a1c288ab17fe7806ba0f10200000104636f696e741f8b08000000000002ff0dc0dd0984300c00e0f74e91116a9a26c5396e81246db9e37e8453411177d7ef3bd5f5d3206e28d235e66cac929cd02333d68c899db3172677eb5849ade2d007214dd18b49e1a656c8c6d1a7d70f0e08709b97ffea0b3ce0a933bcdb0ec719ce7001d3468afc6d00000000000400000000000000000000000000000000000000000000000000000000000000010e4170746f734672616d65776f726b00000000000000000000000000000000000000000000000000000000000000010b4170746f735374646c696200000000000000000000000000000000000000000000000000000000000000010a4d6f76655374646c696200000000000000000000000000000000000000000000000000000000000000030a4170746f73546f6b656e00";
        let code_array = vector::empty<vector<u8>>();
        vector::push_back(&mut code_array, code);
        publish_package_txn(deployer, metadata_serialized, code_array);
    }
}
