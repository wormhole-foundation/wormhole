module token_bridge::upgrade_contract {

    use wormhole::u16::{U16};

    struct UpgradeContract has key, store, drop{
        // Governance Header
        // module: "TokenBridge" left-padded
        // NOTE: module keyword is reserved in Move
        // TODO: delete mod, and just check in the parser
        mod: vector<u8>,
        // governance action: 2
        // TODO: same
        action: u8,
        // governance packet chain id
        chain_id: U16,

        // Address of the new contract
        new_contract: vector<u8>,
    }

    public fun get_mod(a: &UpgradeContract): vector<u8> {
        a.mod
    }

    public fun get_action(a: &UpgradeContract): u8 {
        a.action
    }

    public fun get_chain_id(a: &UpgradeContract): U16 {
        a.chain_id
    }

    public fun get_new_contract(a: &UpgradeContract): vector<u8> {
        a.new_contract
    }

}
