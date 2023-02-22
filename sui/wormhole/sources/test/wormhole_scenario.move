#[test_only]
module wormhole::wormhole_scenario {
    use sui::test_scenario::{Self, Scenario};

    use wormhole::setup::{Self, DeployerCapability};

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
        // to destroy the `DeployerCapability` to create a sharable `State`.
        test_scenario::next_tx(scenario, DEPLOYER);

        {
            setup::init_and_share_state_test_only(
                test_scenario::take_from_address<DeployerCapability>(
                    scenario, DEPLOYER
                ),
                message_fee,
                test_scenario::ctx(scenario)
            );
        };
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
