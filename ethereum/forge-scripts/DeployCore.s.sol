// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {Implementation} from "../contracts/Implementation.sol";
import {Setup} from "../contracts/Setup.sol";
import {Wormhole} from "../contracts/Wormhole.sol";
import "forge-std/Script.sol";

contract DeployCore is Script {
    // DryRun - Deploy the system
    // dry run: forge script ./forge-scripts/DeployCore.s.sol:DeployCore --sig "dryRun()" --rpc-url $RPC
    function dryRun(
        address[] memory initialSigners,
        uint16 chainId,
        uint16 governanceChainId,
        bytes32 governanceContract,
        uint256 evmChainId
    ) public {
        _deploy(
            initialSigners,
            chainId,
            governanceChainId,
            governanceContract,
            evmChainId
        );
    }

    // Deploy the system
    function run(
        address[] memory initialSigners,
        uint16 chainId,
        uint16 governanceChainId,
        bytes32 governanceContract,
        uint256 evmChainId
    )
        public
        returns (
            address deployedAddress,
            address setupAddress,
            address implAddress
        )
    {
        vm.startBroadcast();
        (deployedAddress, setupAddress, implAddress) = _deploy(
            initialSigners,
            chainId,
            governanceChainId,
            governanceContract,
            evmChainId
        );
        vm.stopBroadcast();
    }

    function _deploy(
        address[] memory initialSigners,
        uint16 chainId,
        uint16 governanceChainId,
        bytes32 governanceContract,
        uint256 evmChainId
    )
        internal
        returns (
            address deployedAddress,
            address setupAddress,
            address implAddress
        )
    {
        Implementation impl = new Implementation();
        Setup setup = new Setup();
        
        bytes memory initData = abi.encodeWithSignature(
            "setup(address,address[],uint16,uint16,bytes32,uint256)",
            address(impl),
            initialSigners,
            chainId,
            governanceChainId,
            governanceContract,
            evmChainId
        ); 

        Wormhole wormhole = new Wormhole(
            address(setup),
            initData
        );

        return (address(wormhole), address(setup), address(impl));
    }
}
