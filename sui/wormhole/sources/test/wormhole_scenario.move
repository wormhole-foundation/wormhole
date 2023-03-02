#[test_only]
module wormhole::wormhole_scenario {
    use sui::object::{Self};
    use sui::test_scenario::{Self, Scenario};

    use wormhole::setup::{Self, DeployerCap};

    // NOTE: This exists to mock up sui::package for proposed ugprades.
    use wormhole::dummy_sui_package::{Self as package};

    const DEPLOYER: address = @0xDEADBEEF;
    const WALLET_1: address = @0xB0B1;
    const WALLET_2: address = @0xB0B2;
    const WALLET_3: address = @0xB0B3;

    public fun set_up_wormhole(scenario: &mut Scenario, message_fee: u64) {
        // Process effects prior. `init_test_only` will be executed as the
        // Wormhole contract deployer.
        test_scenario::next_tx(scenario, DEPLOYER);

        // `init` Wormhole contract as if it were published.
        wormhole::setup::init_test_only(test_scenario::ctx(scenario));

        // `init_and_share_state` will also be executed as the Wormhole deployer
        // to destroy the `DeployerCap` to create a sharable `State`.
        test_scenario::next_tx(scenario, DEPLOYER);

        // Parameters for Wormhole's `State` are common in the Wormhole testing
        // environment aside from the `guardian_set_epochs_to_live`, which at
        // the moment needs to be discussed on how to configure. As of now,
        // there is no clock with unix timestamp to expire guardian sets in
        // terms of human-interpretable time.
        {
            // This will be created and sent to the transaction sender
            // automatically when the contract is published. This exists in
            // place of grabbing it from the sender.
            let upgrade_cap =
                package::test_publish(
                    object::id_from_address(@0x0),
                    test_scenario::ctx(scenario)
                );

            let governance_chain = 1;
            let governance_contract =
                x"0000000000000000000000000000000000000000000000000000000000000004";
            let initial_guardians =
                vector[x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"];
            let guardian_set_epochs_to_live = 2;

            // Share `State`.
            setup::init_and_share_state(
                test_scenario::take_from_address<DeployerCap>(
                    scenario, DEPLOYER
                ),
                upgrade_cap,
                governance_chain,
                governance_contract,
                initial_guardians,
                guardian_set_epochs_to_live,
                message_fee,
                test_scenario::ctx(scenario)
            );
        };

        // Done.
    }

    public fun deployer(): address {
        DEPLOYER
    }

    public fun person(): address {
        WALLET_1
    }

    public fun two_people(): (address, address) {
        (WALLET_1, WALLET_2)
    }

    public fun three_people(): (address, address, address) {
        (WALLET_1, WALLET_2, WALLET_3)
    }
}
