module deployer::deployer {
    use aptos_framework::account;
    use aptos_framework::code;

    public entry fun deploy_derived(user: &signer, metadata_serialized: vector<u8>, code: vector<vector<u8>>) {
        let (wormhole, _signer_cap) = account::create_resource_account(user, b"wormhole");
        // TODO(csongor): this throws away the signer_cap. We should move it
        // into the wormhole account after deploy.
        code::publish_package_txn(&wormhole, metadata_serialized, code);
    }
}
