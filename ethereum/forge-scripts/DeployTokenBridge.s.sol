

// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;
import {BridgeImplementation} from "../contracts/bridge/BridgeImplementation.sol";
import {BridgeSetup} from "../contracts/bridge/BridgeSetup.sol";
import {TokenImplementation} from "../contracts/bridge/token/TokenImplementation.sol";
import "forge-std/Script.sol";

contract DeployTokenBridge is Script {
    // DryRun - Deploy the system
    // dry run: forge script DeployTokenBridge --sig "dryRun()" --rpc-url $RPC
    function dryRun() public {
        _deploy();
    }
    // Deploy the system
    // deploy:  forge script DeployTokenBridge --sig "run()" --rpc-url $RPC --etherscan-api-key $ETHERSCAN_API_KEY --private-key $RAW_PRIVATE_KEY --broadcast --verify
    function run() public returns (address deployedAddress) {
        vm.startBroadcast();
        deployedAddress = _deploy();
        vm.stopBroadcast();
    }

    function _deploy() internal returns (address deployedAddress) {
        BridgeImplementation bridgeImpl = new BridgeImplementation();
        TokenImplementation tokenImpl = new TokenImplementation();
        BridgeSetup bridgeSetup = new BridgeSetup();

        uint16 chainId = uint16(vm.envUint("BRIDGE_INIT_CHAIN_ID"));
        uint16 governanceChainId = uint16(vm.envUint("BRIDGE_INIT_GOV_CHAIN_ID"));
        bytes32 governanceContract = bytes32(vm.envBytes32("BRIDGE_INIT_GOV_CONTRACT"));
        address WETH = vm.envAddress("BRIDGE_INIT_WETH");
        uint8 finality = uint8(vm.envUint("BRIDGE_INIT_FINALITY"));
        uint256 evmChainId = vm.envUint("INIT_EVM_CHAIN_ID");

        address wormhole = vm.envAddress("WORMHOLE_ADDRESS");

        bridgeSetup.setup(
            address(bridgeImpl),
            chainId,
            wormhole,
            governanceChainId,
            governanceContract,
            address(tokenImpl),
            WETH,
            finality,
            evmChainId
        );

        return address(bridgeSetup);

        // TODO: initialize?
    }
}