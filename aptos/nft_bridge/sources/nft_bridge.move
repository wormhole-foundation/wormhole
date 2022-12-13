module nft_bridge::nft_bridge {
    #[test_only]
    use aptos_framework::account::{Self};
    use aptos_framework::account::{SignerCapability};
    use deployer::deployer::{claim_signer_capability};
    use nft_bridge::state::{init_nft_bridge_state};
    use wormhole::wormhole;

    /// Initializes the contract.
    entry fun init_module(deployer: &signer) {
        let signer_cap = claim_signer_capability(deployer, @nft_bridge);
        init_internal(signer_cap);
    }

    fun init_internal(signer_cap: SignerCapability) {
        let emitter_cap = wormhole::register_emitter();
        init_nft_bridge_state(signer_cap, emitter_cap);
    }

    #[test_only]
    /// Initialise contracts for testing
    /// Returns the nft_bridge signer and wormhole signer
    public fun init_test(deployer: &signer) {
        let (_nft_bridge, signer_cap) = account::create_resource_account(deployer, b"nft_bridge");
        init_internal(signer_cap);
    }
}
