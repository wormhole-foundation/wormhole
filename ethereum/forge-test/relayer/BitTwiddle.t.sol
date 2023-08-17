// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {IDeliveryProvider} from "../../contracts/interfaces/relayer/IDeliveryProviderTyped.sol";
import {MessageKey} from "../../contracts/interfaces/relayer/IWormholeRelayerTyped.sol";
import {DeliveryProvider} from "../../contracts/relayer/deliveryProvider/DeliveryProvider.sol";
import {WormholeRelayerSend} from "../../contracts/relayer/wormholeRelayer/WormholeRelayerSend.sol";
import {WormholeRelayerBase} from "../../contracts/relayer/wormholeRelayer/WormholeRelayerBase.sol";
import {WormholeRelayerSerde} from
    "../../contracts/relayer/wormholeRelayer/WormholeRelayerSerde.sol";
import {toWormholeFormat, fromWormholeFormat} from "../../contracts/libraries/relayer/Utils.sol";
import {BytesParsing} from "../../contracts/libraries/relayer/BytesParsing.sol";
import "../../contracts/interfaces/relayer/TypedUnits.sol";

import "forge-std/Test.sol";
import "forge-std/console.sol";
import "forge-std/Vm.sol";

// contract Harness is WormholeRelayerSend {
contract Harness  {
    // constructor() WormholeRelayerBase(address(1)) {}

    // function harness_checkMessageKeyTypesSupportedByDeliveryProvider(
    //     IDeliveryProvider provider,
    //     MessageKey[] memory messageKeys
    // ) public view {
    //     checkMessageKeyTypesSupportedByDeliveryProvider(provider, messageKeys);
    // }

    function checkMessageKeyTypesSupportedByDeliveryProvider(
        IDeliveryProvider provider,
        MessageKey[] memory messageKeys
    ) public view {
        uint256 seenKeyTypes = 0;
        for (uint256 i = 0; i < messageKeys.length;) {
            uint256 mask = 1 << messageKeys[i].keyType;
            console.log(seenKeyTypes, mask, seenKeyTypes & mask, seenKeyTypes | mask);
            if (seenKeyTypes & mask > 0) {
                console.log("continue");
                continue;
            }
            if (!provider.isMessageKeyTypeSupported(messageKeys[i].keyType)) {
                revert("boo");
            }
            seenKeyTypes |= mask;

            unchecked {
                ++i;
            }
        }
    }
}

contract MockProvider is IDeliveryProvider {
    mapping(uint8 => bool) supported;

    function quoteDeliveryPrice(
        uint16,
        TargetNative,
        bytes memory
    ) external pure returns (LocalNative, bytes memory) {
        revert("unimplemented");
    }

    function quoteAssetConversion(uint16, LocalNative) external pure returns (TargetNative) {
        revert("unimplemented");
    }

    function getRewardAddress() external pure returns (address payable) {
        revert("unimplemented");
    }

    function isChainSupported(uint16) external pure returns (bool) {
        revert("unimplemented");
    }

    function setSupported(uint8 keyType, bool isSupported) public {
        supported[keyType] = isSupported;
    }

    function isMessageKeyTypeSupported(uint8 keyType) external view returns (bool) {
        console.log(supported[keyType]);
        return supported[keyType];
    }

    function getTargetChainAddress(uint16) external pure returns (bytes32) {
        revert("unimplemented");
    }
}

contract BitTwiddleTest is Test {
    using BytesParsing for bytes;
    using WeiLib for Wei;
    using GasLib for Gas;
    using WeiPriceLib for WeiPrice;
    using GasPriceLib for GasPrice;
    using TargetNativeLib for TargetNative;
    using LocalNativeLib for LocalNative;

    function testX() public {
        MockProvider provider = new MockProvider();
        Harness harness = new Harness();

        MessageKey[] memory messageKeys = new MessageKey[](2);
        messageKeys[0] = MessageKey(1, bytes(""));
        messageKeys[1] = MessageKey(2, bytes(""));

        vm.expectRevert();
        harness.checkMessageKeyTypesSupportedByDeliveryProvider(provider, messageKeys);

        provider.setSupported(1, true);
        vm.expectRevert();
        harness.checkMessageKeyTypesSupportedByDeliveryProvider(provider, messageKeys);

        provider.setSupported(2, true);
        harness.checkMessageKeyTypesSupportedByDeliveryProvider(provider, messageKeys);
    }
}
