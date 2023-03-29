// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {IRelayProvider} from "../contracts/interfaces/IRelayProvider.sol";
import {RelayProvider} from "../contracts/relayProvider/RelayProvider.sol";
import {RelayProviderSetup} from "../contracts/relayProvider/RelayProviderSetup.sol";
import {RelayProviderImplementation} from "../contracts/relayProvider/RelayProviderImplementation.sol";
import {RelayProviderProxy} from "../contracts/relayProvider/RelayProviderProxy.sol";
import {IWormholeRelayer} from "../contracts/interfaces/IWormholeRelayer.sol";
import {CoreRelayer} from "../contracts/coreRelayer/CoreRelayer.sol";
import {CoreRelayerSetup} from "../contracts/coreRelayer/CoreRelayerSetup.sol";
import {CoreRelayerImplementation} from "../contracts/coreRelayer/CoreRelayerImplementation.sol";
import {CoreRelayerProxy} from "../contracts/coreRelayer/CoreRelayerProxy.sol";
import {CoreRelayerGovernance} from "../contracts/coreRelayer/CoreRelayerGovernance.sol";
import {MockGenericRelayer} from "./MockGenericRelayer.sol";
import {MockWormhole} from "../contracts/mock/MockWormhole.sol";
import {IWormhole} from "../contracts/interfaces/IWormhole.sol";
import {WormholeSimulator, FakeWormholeSimulator} from "./WormholeSimulator.sol";
import "../contracts/libraries/external/BytesLib.sol";

import "forge-std/Test.sol";
import "forge-std/console.sol";
import "forge-std/Vm.sol";

contract TestHelpers {
    using BytesLib for bytes;

    address private constant VM_ADDRESS = address(bytes20(uint160(uint256(keccak256("hevm cheat code")))));

    Vm public constant vm = Vm(VM_ADDRESS);

    WormholeSimulator helperWormholeSimulator;

    constructor() {
        (, helperWormholeSimulator) = setUpWormhole(1);
    }

    function registerCoreRelayerContract(
        CoreRelayer governance,
        IWormhole wormhole,
        uint16 currentChainId,
        uint16 chainId,
        bytes32 coreRelayerContractAddress
    ) public {
        bytes32 coreRelayerModule = 0x000000000000000000000000000000000000000000436F726552656C61796572;
        bytes memory message =
            abi.encodePacked(coreRelayerModule, uint8(2), currentChainId, chainId, coreRelayerContractAddress);
        IWormhole.VM memory preSignedMessage = IWormhole.VM({
            version: 1,
            timestamp: uint32(block.timestamp),
            nonce: 0,
            emitterChainId: wormhole.governanceChainId(),
            emitterAddress: wormhole.governanceContract(),
            sequence: 0,
            consistencyLevel: 200,
            payload: message,
            guardianSetIndex: 0,
            signatures: new IWormhole.Signature[](0),
            hash: bytes32("")
        });

        bytes memory signed = helperWormholeSimulator.encodeAndSignMessage(preSignedMessage);
        governance.registerCoreRelayerContract(signed);
    }

    function setUpWormhole(uint16 chainId)
        public
        returns (IWormhole wormholeContract, WormholeSimulator wormholeSimulator)
    {
        // deploy Wormhole
        MockWormhole wormhole = new MockWormhole({
            initChainId: chainId,
            initEvmChainId: block.chainid
        });

        // replace Wormhole with the Wormhole Simulator contract (giving access to some nice helper methods for signing)
        wormholeSimulator = new FakeWormholeSimulator(
            wormhole
        );

        wormholeContract = wormhole;
    }

    function setUpRelayProvider(uint16 chainId) public returns (RelayProvider relayProvider) {
        vm.prank(msg.sender);
        RelayProviderSetup relayProviderSetup = new RelayProviderSetup();
        vm.prank(msg.sender);
        RelayProviderImplementation relayProviderImplementation = new RelayProviderImplementation();
        vm.prank(msg.sender);
        RelayProviderProxy myRelayProvider = new RelayProviderProxy(
            address(relayProviderSetup),
            abi.encodeCall(
                RelayProviderSetup.setup,
                (
                    address(relayProviderImplementation),
                    chainId
                )
            )
        );

        relayProvider = RelayProvider(address(myRelayProvider));
    }

    function setUpCoreRelayer(uint16 chainId, IWormhole wormhole, address defaultRelayProvider)
        public
        returns (IWormholeRelayer coreRelayer)
    {
        CoreRelayerSetup coreRelayerSetup = new CoreRelayerSetup();
        CoreRelayerImplementation coreRelayerImplementation = new CoreRelayerImplementation();
        CoreRelayerProxy myCoreRelayer = new CoreRelayerProxy(
            address(coreRelayerSetup),
            abi.encodeCall(
                CoreRelayerSetup.setup,
                (
                    address(coreRelayerImplementation),
                    chainId,
                    address(wormhole),
                    defaultRelayProvider,
                    wormhole.governanceChainId(),
                    wormhole.governanceContract(),
                    block.chainid
                )
            )
        );
        coreRelayer = IWormholeRelayer(address(myCoreRelayer));
    }
}
