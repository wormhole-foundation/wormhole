// SPDX-License-Identifier: Apache-2.0
// slither-disable-start reentrancy-benign

pragma solidity 0.8.26;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";
import {QueryTypeStakerFactory} from "src/QueryTypeStakerFactory.sol";
import {QueryTypeStakingPool} from "src/QueryTypeStakingPool.sol";

contract CreateStakingPool is Script {
  // Default configuration constants
  uint256 constant DEFAULT_MINIMUM_STAKE = 5_000 * 10**18; // 5,000 tokens
  uint256 constant DEFAULT_STAKING_TOKEN_CAPACITY = 1_000_000 * 10**18; // 1 million tokens
  uint48 constant DEFAULT_LOCKUP_PERIOD = 900; // 15 minutes
  uint48 constant DEFAULT_ACCESS_PERIOD = 1800; // 30 minutes

  function run() public returns (address) {
    // Get the deployer's private key and factory address from environment
    uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
    address factoryAddress = vm.envAddress("FACTORY_ADDRESS");

    // Get pool parameters from environment
    bytes32 queryType = vm.envBytes32("QUERY_TYPE");
    bytes32 initialEntry = vm.envBytes32("INITIAL_ENTRY");

    // Get configuration parameters from environment or use defaults
    uint256 minimumStake = vm.envOr("MINIMUM_STAKE", DEFAULT_MINIMUM_STAKE);
    uint256 stakingTokenCapacity = vm.envOr("STAKING_TOKEN_CAPACITY", DEFAULT_STAKING_TOKEN_CAPACITY);
    uint48 lockupPeriod = uint48(vm.envOr("LOCKUP_PERIOD", uint256(DEFAULT_LOCKUP_PERIOD)));
    uint48 accessPeriod = uint48(vm.envOr("ACCESS_PERIOD", uint256(DEFAULT_ACCESS_PERIOD)));

    // Start broadcasting transactions
    vm.startBroadcast(deployerPrivateKey);

    // Pool owner is the same as deployer
    address poolOwner = vm.addr(deployerPrivateKey);

    // Create the staking pool (6-arg interface)
    // Note: Factory now accepts lockupPeriod, accessPeriod, minimumStake directly
    QueryTypeStakerFactory factory = QueryTypeStakerFactory(factoryAddress);
    address poolAddress = factory.createStakingPool(
      queryType,
      poolOwner,
      initialEntry,
      lockupPeriod,
      accessPeriod,
      minimumStake
    );

    console.log("========================================");
    console.log("Staking pool created at:", poolAddress);
    console.log("========================================");

    // Configure additional pool settings after creation
    QueryTypeStakingPool pool = QueryTypeStakingPool(poolAddress);

    // Set staking token capacity (not part of createStakingPool)
    pool.setStakingTokenCapacity(stakingTokenCapacity);
    console.log("Staking token capacity set to:", stakingTokenCapacity / 10**18, "tokens");
    console.log("Minimum stake:", minimumStake / 10**18, "tokens");
    console.log("Lockup period:", lockupPeriod / 60, "minutes");
    console.log("Access period:", accessPeriod / 60, "minutes");

    // Update conversion table with rate limits CID
    bytes32 rateLimitsCid = vm.envBytes32("RATE_LIMITS_CID");

    if (rateLimitsCid != bytes32(0)) {
      pool.updateConversionTable(rateLimitsCid);
      console.log("Updated conversion table with rate limits CID");
    }

    vm.stopBroadcast();

    console.log("Pool address:", poolAddress);
    console.log("========================================");

    return poolAddress;
  }
}
