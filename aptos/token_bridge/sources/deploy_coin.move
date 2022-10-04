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
    use std::signer::{Self};
    use std::vector::{Self};
    use std::code::{publish_package_txn};
    use std::bcs::{Self};

    public entry fun deploy_coin(deployer: &signer) {
        let addr = signer::address_of(deployer);
        let addr_bytes = bcs::to_bytes(&addr);
        let code = x"a11ceb0b05000000050100020202040706130819200a390500000001080004636f696e01540b64756d6d795f6669656c64";
        vector::append(&mut code, addr_bytes);
        vector::append(&mut code, x"000201020100");
        let metadata_serialized: vector<u8> = x"0b57726170706564436f696e010000000000000000404237423334333744324439304635433830464246313246423231423543424543383332443335453138364542373539304431434134324338334631333639324586021f8b08000000000002ffb590316bc330108577fd0a21af8d9dae810ea5a55bc9d0408710cac9ba388765e990649752fadf2b252e5d0299bce9ddbd77ef437b86b6870e0fc2c180f241aaf700cc689e3c3925260c91bc2bf375bdaeef9518b90b60f083bda5f6ab2c5a3f3024d2169510d56efbbcdd482627e76c941a8f3ea01c809cc324035a8488a2da1b6474065d4b180fa27ae4e4e34bc81c9f3ef4f9f4b7ec28958a534a1c374d93e569d4756e6ca0985716749c9f6deea8b341ddc9386a43a1042fabc14fd81cff0ecffe7f9d1301a76237386542257f44f59a336fc958d2cb8114b98ae792eb10e71f599ae232bc89b1f33dbaa5295229b906f10b3085ae64a80200000104636f696e731f8b08000000000002ff0dc0dd0984300c00e0f74e91116a9a26c5396e81246db9e37e8453411177d7ef3bd5f5d3206e28d235e66cac929cd02333d68c899db3172677eb5849ade2d007214dd18b49e1a656c8c6d1a7d70f8e00b779f9afbec0039e3ac3bbed709ce10c176825e8506c00000000000000";
        let code_array = vector::empty<vector<u8>>();
        vector::push_back(&mut code_array, code);
        publish_package_txn(deployer, metadata_serialized, code_array);
    }
}
