module wormhole::setup {
    use sui::object::{Self, UID};
    use sui::transfer::{Self};
    use sui::tx_context::{Self, TxContext};

    use wormhole::state::{Self};

    /// Capability created at `init`, which will be destroyed once
    /// `init_and_share_state` is called. This ensures only the deployer can
    /// create the shared `State`.
    struct DeployerCapability has key, store {
        id: UID
    }

    /// Called automatically when module is first published. Transfers
    /// `DeployerCapability` to sender.
    ///
    /// Only `setup::init_and_share_state` requires `DeployerCapability`.
    fun init(ctx: &mut TxContext) {
        let deployer = DeployerCapability { id: object::new(ctx) };
        transfer::transfer(deployer, tx_context::sender(ctx));
    }

    #[test_only]
    public fun init_test_only(ctx: &mut TxContext) {
        init(ctx)
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

    #[test_only]
    public fun init_and_share_state_test_only(
        deployer: DeployerCapability,
        message_fee: u64,
        ctx: &mut TxContext
    ) {
        let governance_chain = 1;
        let governance_contract =
            x"0000000000000000000000000000000000000000000000000000000000000004";
        let guardians = vector[x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"];
        let guardian_set_epochs_to_live = 2;

        init_and_share_state(
            deployer,
            governance_chain,
            governance_contract,
            guardians,
            guardian_set_epochs_to_live,
            message_fee,
            ctx
        )
    }
}

#[test_only]
module wormhole::setup_test {
    // TODO
}
