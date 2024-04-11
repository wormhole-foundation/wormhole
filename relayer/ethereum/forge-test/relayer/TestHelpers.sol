// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {IDeliveryProvider} from "../../contracts/interfaces/relayer/IDeliveryProviderTyped.sol";
import {DeliveryProvider} from "../../contracts/relayer/deliveryProvider/DeliveryProvider.sol";
import {DeliveryProviderSetup} from
    "../../contracts/relayer/deliveryProvider/DeliveryProviderSetup.sol";
import {DeliveryProviderImplementation} from
    "../../contracts/relayer/deliveryProvider/DeliveryProviderImplementation.sol";
import {DeliveryProviderProxy} from
    "../../contracts/relayer/deliveryProvider/DeliveryProviderProxy.sol";
import {IWormholeRelayer} from "../../contracts/interfaces/relayer/IWormholeRelayerTyped.sol";
import {WormholeRelayer} from "../../contracts/relayer/wormholeRelayer/WormholeRelayer.sol";
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

    function registerWormholeRelayerContract(
        WormholeRelayer governance,
        IWormhole wormhole,
        uint16 currentChainId,
        uint16 chainId,
        bytes32 coreRelayerContractAddress
    ) public {
        bytes32 wormholeRelayerModule =
            0x0000000000000000000000000000000000576f726d686f6c6552656c61796572;
        bytes memory message = abi.encodePacked(
            wormholeRelayerModule, uint8(1), currentChainId, chainId, coreRelayerContractAddress
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
        governance.registerWormholeRelayerContract(signed);
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

    function setUpDeliveryProvider(
        uint16 chainId
    ) public returns (DeliveryProvider deliveryProvider) {
        vm.prank(msg.sender);
        DeliveryProviderSetup deliveryProviderSetup = new DeliveryProviderSetup();
        vm.prank(msg.sender);
        DeliveryProviderImplementation deliveryProviderImplementation =
            new DeliveryProviderImplementation();
        vm.prank(msg.sender);
        DeliveryProviderProxy myDeliveryProvider = new DeliveryProviderProxy(
            address(deliveryProviderSetup),
            abi.encodeCall(
                DeliveryProviderSetup.setup,
                (
                    address(deliveryProviderImplementation),
                    chainId
                )
            )
        );

        deliveryProvider = DeliveryProvider(address(myDeliveryProvider));
    }

    function setUpWormholeRelayer(
        IWormhole wormhole,
        address defaultDeliveryProvider
    ) public returns (IWormholeRelayer coreRelayer) {
        Create2Factory create2Factory = new Create2Factory();

        address proxyAddressComputed =
            create2Factory.computeProxyAddress(address(this), "0xGenericRelayer");

        WormholeRelayer coreRelayerImplementation = new WormholeRelayer(address(wormhole));

        bytes memory initCall =
            abi.encodeCall(WormholeRelayer.initialize, (defaultDeliveryProvider));

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
