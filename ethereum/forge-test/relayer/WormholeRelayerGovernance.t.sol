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
import {DeliveryProviderStructs} from
    "../../contracts/relayer/deliveryProvider/DeliveryProviderStructs.sol";
import "../../contracts/interfaces/relayer/IWormholeRelayerTyped.sol";
import {WormholeRelayer} from "../../contracts/relayer/wormholeRelayer/WormholeRelayer.sol";
import {MockGenericRelayer} from "./MockGenericRelayer.sol";
import {MockWormhole} from "./MockWormhole.sol";
import {IWormhole} from "../../contracts/interfaces/IWormhole.sol";
import {WormholeSimulator, FakeWormholeSimulator} from "./WormholeSimulator.sol";
import {IWormholeReceiver} from "../../contracts/interfaces/relayer/IWormholeReceiver.sol";
import {MockRelayerIntegration} from "../../contracts/mock/relayer/MockRelayerIntegration.sol";
import {TestHelpers} from "./TestHelpers.sol";
import {toWormholeFormat} from "../../contracts/relayer/libraries/Utils.sol";
import "../../contracts/libraries/external/BytesLib.sol";

import "forge-std/Test.sol";
import "forge-std/console.sol";
import "forge-std/Vm.sol";

contract Brick {
    function checkAndExecuteUpgradeMigration() external view {}
}

contract WormholeRelayerGovernanceTests is Test {
    using BytesLib for bytes;

    TestHelpers helpers;

    bytes32 relayerModule = 0x0000000000000000000000000000000000576f726d686f6c6552656c61796572;
    IWormhole wormhole;
    IDeliveryProvider deliveryProvider;
    WormholeSimulator wormholeSimulator;
    IWormholeRelayer wormholeRelayer;

    function setUp() public {
        helpers = new TestHelpers();
        (wormhole, wormholeSimulator) = helpers.setUpWormhole(1);
        deliveryProvider = helpers.setUpDeliveryProvider(1);
        wormholeRelayer = helpers.setUpWormholeRelayer(wormhole, address(deliveryProvider));
    }

    struct GovernanceStack {
        bytes message;
        IWormhole.VM preSignedMessage;
        bytes signed;
    }

    function signMessage(bytes memory message) internal returns (bytes memory signed) {
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
        signed = wormholeSimulator.encodeAndSignMessage(preSignedMessage);
    }

    function fillInGovernanceStack(bytes memory message)
        internal
        returns (GovernanceStack memory stack)
    {
        stack.message = message;
        stack.preSignedMessage = IWormhole.VM({
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
        stack.signed = wormholeSimulator.encodeAndSignMessage(stack.preSignedMessage);
    }

    function testSetDefaultDeliveryProvider() public {
        IDeliveryProvider deliveryProviderB = helpers.setUpDeliveryProvider(1);
        IDeliveryProvider deliveryProviderC = helpers.setUpDeliveryProvider(1);

        bytes memory signed = signMessage(
            abi.encodePacked(
                relayerModule,
                uint8(3),
                uint16(1),
                bytes32(uint256(uint160(address(deliveryProviderB))))
            )
        );

        WormholeRelayer(payable(address(wormholeRelayer))).setDefaultDeliveryProvider(signed);

        assertTrue(wormholeRelayer.getDefaultDeliveryProvider() == address(deliveryProviderB));

        signed = signMessage(
            abi.encodePacked(
                relayerModule,
                uint8(3),
                uint16(1),
                bytes32(uint256(uint160(address(deliveryProviderC))))
            )
        );

        WormholeRelayer(payable(address(wormholeRelayer))).setDefaultDeliveryProvider(signed);

        assertTrue(wormholeRelayer.getDefaultDeliveryProvider() == address(deliveryProviderC));
    }

    function testRegisterChain() public {
        IWormholeRelayer wormholeRelayer1 =
            helpers.setUpWormholeRelayer(wormhole, address(deliveryProvider));
        IWormholeRelayer wormholeRelayer2 =
            helpers.setUpWormholeRelayer(wormhole, address(deliveryProvider));
        IWormholeRelayer wormholeRelayer3 =
            helpers.setUpWormholeRelayer(wormhole, address(deliveryProvider));

        helpers.registerWormholeRelayerContract(
            WormholeRelayer(payable(address(wormholeRelayer1))),
            wormhole,
            1,
            2,
            toWormholeFormat(address(wormholeRelayer2))
        );

        helpers.registerWormholeRelayerContract(
            WormholeRelayer(payable(address(wormholeRelayer1))),
            wormhole,
            1,
            3,
            toWormholeFormat(address(wormholeRelayer3))
        );

        assertTrue(
            WormholeRelayer(payable(address(wormholeRelayer1))).getRegisteredWormholeRelayerContract(
                2
            ) == toWormholeFormat(address(wormholeRelayer2))
        );

        assertTrue(
            WormholeRelayer(payable(address(wormholeRelayer1))).getRegisteredWormholeRelayerContract(
                3
            ) == toWormholeFormat(address(wormholeRelayer3))
        );

        vm.expectRevert(
            abi.encodeWithSignature(
                "ChainAlreadyRegistered(uint16,bytes32)",
                3,
                toWormholeFormat(address(wormholeRelayer3))
            )
        );
        helpers.registerWormholeRelayerContract(
            WormholeRelayer(payable(address(wormholeRelayer1))),
            wormhole,
            1,
            3,
            toWormholeFormat(address(wormholeRelayer2))
        );
    }

    function testUpgradeContractToItself() public {
        address payable myWormholeRelayer =
            payable(address(helpers.setUpWormholeRelayer(wormhole, address(deliveryProvider))));

        bytes memory noMigrationFunction = signMessage(
            abi.encodePacked(
                relayerModule,
                uint8(2),
                uint16(1),
                toWormholeFormat(address(new DeliveryProviderImplementation()))
            )
        );

        vm.expectRevert();
        WormholeRelayer(myWormholeRelayer).submitContractUpgrade(noMigrationFunction);

        Brick brick = new Brick();
        bytes memory signed = signMessage(
            abi.encodePacked(relayerModule, uint8(2), uint16(1), toWormholeFormat(address(brick)))
        );

        WormholeRelayer(myWormholeRelayer).submitContractUpgrade(signed);

        vm.expectRevert();
        WormholeRelayer(myWormholeRelayer).getDefaultDeliveryProvider();
    }
}
