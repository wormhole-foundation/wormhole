module token_bridge::register_chain {

    use wormhole::u16::{U16};

    struct RegisterChain has key, store, drop {
        // TODO: same as above -- we shouldn't keep this in the parsed type,
        // only check in the parser.
        // Governance Header
        // module: "TokenBridge" left-padded
        // NOTE: module keyword is reserved in Move
        mod: vector<u8>,
        // governance action: 1
        // TODO: same; remove
        action: u8,
        // governance paket chain id: this or 0
        chain_id: U16,

        // Chain ID
        emitter_chain_id: U16,
        // Emitter address. Left-zero-padded if shorter than 32 bytes
        emitter_address: vector<u8>,
    }

    public fun get_mod(a: &RegisterChain): vector<u8> {
        a.mod
    }

    public fun get_action(a: &RegisterChain): u8 {
        a.action
    }

    public fun get_chain_id(a: &RegisterChain): U16 {
        a.chain_id
    }

    public fun get_emitter_chain_id(a: &RegisterChain): U16 {
        a.emitter_chain_id
    }

    public fun get_emitter_address(a: &RegisterChain): vector<u8> {
        a.emitter_address
    }

}
