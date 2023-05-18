// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {IRelayProvider} from "../../contracts/interfaces/relayer/IRelayProvider.sol";
import {RelayProvider} from "../../contracts/relayer/relayProvider/RelayProvider.sol";
import {RelayProviderSetup} from "../../contracts/relayer/relayProvider/RelayProviderSetup.sol";
import {RelayProviderImplementation} from
    "../../contracts/relayer/relayProvider/RelayProviderImplementation.sol";
import {RelayProviderProxy} from "../../contracts/relayer/relayProvider/RelayProviderProxy.sol";
import {IWormholeRelayer} from "../../contracts/interfaces/relayer/IWormholeRelayer.sol";
import {CoreRelayer} from "../../contracts/relayer/coreRelayer/CoreRelayer.sol";
import {Create2Factory} from "../../contracts/relayer/create2Factory/Create2Factory.sol";
import {MockGenericRelayer} from "./MockGenericRelayer.sol";
import {MockWormhole} from "./MockWormhole.sol";
import {IWormhole} from "../../contracts/interfaces/IWormhole.sol";
import {WormholeSimulator, FakeWormholeSimulator} from "./WormholeSimulator.sol";
import "../../contracts/libraries/external/BytesLib.sol";

import "forge-std/Test.sol";
import "forge-std/console.sol";
import "forge-std/Vm.sol";

contract TestHelpers {
    using BytesLib for bytes;

    address private constant VM_ADDRESS =
        address(bytes20(uint160(uint256(keccak256("hevm cheat code")))));

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
        bytes32 coreRelayerModule =
            0x000000000000000000000000000000000000000000436F726552656C61796572;
        bytes memory message = abi.encodePacked(
            coreRelayerModule, uint8(1), currentChainId, chainId, coreRelayerContractAddress
        );
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

    function setUpCoreRelayer(
        IWormhole wormhole,
        address defaultRelayProvider
    ) public returns (IWormholeRelayer coreRelayer) {
        Create2Factory create2Factory = new Create2Factory();

        address proxyAddressComputed =
            create2Factory.computeProxyAddress(address(this), "0xGenericRelayer");

        CoreRelayer coreRelayerImplementation = new CoreRelayer(address(wormhole));

        bytes memory initCall = abi.encodeCall(CoreRelayer.initialize, (defaultRelayProvider));

        coreRelayer = IWormholeRelayer(
            create2Factory.create2Proxy(
                "0xGenericRelayer", address(coreRelayerImplementation), initCall
            )
        );
        require(
            address(coreRelayer) == proxyAddressComputed, "computed must match actual proxy addr"
        );
    }
}
