module wormhole::governance_message {
    use sui::tx_context::{TxContext};
    use wormhole::bytes::{Self};
    use wormhole::bytes32::{Self, Bytes32};
    use wormhole::cursor::{Self};
    use wormhole::state::{Self, State, chain_id};
    use wormhole::vaa::{Self, VAA};

    const E_OLD_GUARDIAN_SET_GOVERNANCE: u64 = 0;
    const E_INVALID_GOVERNANCE_CHAIN: u64 = 1;
    const E_INVALID_GOVERNANCE_EMITTER: u64 = 2;
    const E_INVALID_MODULE: u64 = 4;
    const E_INVALID_ACTION: u64 = 5;

    struct GovernanceMessage {
        module_name: Bytes32,
        action: u8,
        chain: u16,
        payload: vector<u8>,
        vaa_hash: Bytes32
    }

    public fun module_name(self: &GovernanceMessage): Bytes32 {
        self.module_name
    }

    public fun action(self: &GovernanceMessage): u8 {
        self.action
    }

    public fun is_global_action(self: &GovernanceMessage): bool {
        self.chain == 0
    }

    public fun is_local_action(self: &GovernanceMessage): bool {
        self.chain == chain_id()
    }

    public fun vaa_hash(self: &GovernanceMessage): Bytes32 {
        self.vaa_hash
    }

    public fun take_payload(msg: GovernanceMessage): vector<u8> {
        let GovernanceMessage {
            module_name: _,
            action: _,
            chain: _,
            vaa_hash: _,
            payload
        } = msg;

        payload
    }

    public fun parse_and_verify_vaa(
        wormhole_state: &mut State,
        vaa_buf: vector<u8>,
        ctx: &TxContext
    ): GovernanceMessage {
        let parsed =
            vaa::parse_and_verify(
                wormhole_state,
                vaa_buf,
                ctx
            );

        // This VAA must have originated from the governance emitter.
        assert_governance_emitter(wormhole_state, &parsed);

        let vaa_hash = vaa::hash(&parsed);

        let cur = cursor::new(vaa::take_payload(parsed));

        let module_name = bytes32::from_cursor(&mut cur);
        let action = bytes::deserialize_u8(&mut cur);
        let chain = bytes::deserialize_u16_be(&mut cur);
        let payload = cursor::rest(cur);

        GovernanceMessage { module_name, action, chain, payload, vaa_hash }
    }

    /// Aborts if the VAA is not governance (i.e. sent from the governance
    /// emitter on the governance chain)
    fun assert_governance_emitter(wormhole_state: &State, parsed: &VAA) {
        // Protect against governance actions enacted using an old guardian set.
        // This is not a protection found in the other Wormhole contracts.
        assert!(
            vaa::guardian_set_index(parsed) == state::guardian_set_index(wormhole_state),
            E_OLD_GUARDIAN_SET_GOVERNANCE
        );

        // Both the emitter chain and address must equal those known by the
        // Wormhole `State`.
        assert!(
            vaa::emitter_chain(parsed) == state::governance_chain(wormhole_state),
            E_INVALID_GOVERNANCE_CHAIN
        );
        assert!(
            vaa::emitter_address(parsed) == state::governance_contract(wormhole_state),
            E_INVALID_GOVERNANCE_EMITTER
        );
    }
}

module wormhole::governance_action_test {
    // TODO
}
