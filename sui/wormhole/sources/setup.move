module wormhole::setup {
    use sui::object::{Self, UID};
    use sui::transfer::{Self};
    use sui::tx_context::{TxContext};

    use wormhole::state::{Self};

    // Needs `new_capability`
    friend wormhole::wormhole;

    /// Capability created at `init`, which will be destroyed once
    /// `init_and_share_state` is called. This ensures only the deployer can
    /// create the shared `State`.
    struct DeployerCapability has key, store {
        id: UID
    }

    public(friend) fun new_capability(ctx: &mut TxContext): DeployerCapability {
        DeployerCapability { id: object::new(ctx) }
    }

    // creates a shared state object, so that anyone can get a reference to
    // &mut State and pass it into various functions
    public entry fun init_and_share_state(
        deployer: DeployerCapability,
        governance_chain: u16,
        governance_contract: vector<u8>,
        initial_guardians: vector<vector<u8>>,
        guardian_set_epochs_to_live: u32,
        message_fee: u64,
        ctx: &mut TxContext
    ) {
        let DeployerCapability{ id } = deployer;
        object::delete(id);

        // permanently shares state
        transfer::share_object(
            state::new(
                governance_chain,
                governance_contract,
                initial_guardians,
                guardian_set_epochs_to_live,
                message_fee,
                ctx
            )
        );
    }

}
