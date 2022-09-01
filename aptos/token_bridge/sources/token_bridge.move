module token_bridge::token_bridge {
    use deployer::deployer::{claim_signer_capability};
    use token_bridge::bridge_state::{init_token_bridge_state};
    use wormhole::wormhole;

    /// This function automatically gets called when the module is deployed,
    /// with the signer the account the module is deployed under (i.e. the token
    /// bridge resource account)
    entry fun init_module(token_bridge: &signer) {
        let signer_cap = claim_signer_capability(token_bridge, @token_bridge);
        let emitter_cap = wormhole::register_emitter();
        init_token_bridge_state(token_bridge, signer_cap, emitter_cap);
    }
}
