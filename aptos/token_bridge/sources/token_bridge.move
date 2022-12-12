module token_bridge::token_bridge {
    #[test_only]
    use aptos_framework::account::{Self};
    use aptos_framework::account::{SignerCapability};
    use deployer::deployer::{claim_signer_capability};
    use token_bridge::state::{init_token_bridge_state};
    use wormhole::wormhole;

    /// Initializes the contract.
    /// The native `init_module` cannot be used, because it runs on each upgrade
    /// (oddly).
    /// TODO: the above behaviour has been remedied in the Aptos VM, so we could
    /// use `init_module` now. Let's reconsider before the mainnet launch.
    /// Can only be called by the deployer (checked by the
    /// `deployer::claim_signer_capability` function).
    public entry fun init(deployer: &signer) {
        let signer_cap = claim_signer_capability(deployer, @token_bridge);
        init_internal(signer_cap);
    }

    fun init_internal(signer_cap: SignerCapability) {
        let emitter_cap = wormhole::register_emitter();
        init_token_bridge_state(signer_cap, emitter_cap);
    }

    #[test_only]
    /// Initialise contracts for testing
    /// Returns the token_bridge signer and wormhole signer
    public fun init_test(deployer: &signer) {
        let (_token_bridge, signer_cap) = account::create_resource_account(deployer, b"token_bridge");
        init_internal(signer_cap);
    }
}
