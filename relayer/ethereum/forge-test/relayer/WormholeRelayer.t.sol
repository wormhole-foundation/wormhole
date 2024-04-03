// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {IDeliveryProvider} from "../../contracts/interfaces/relayer/IDeliveryProvider.sol";
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
import {
    DeliveryInstruction,
    RedeliveryInstruction,
    DeliveryOverride,
    EvmDeliveryInstruction
} from "../../contracts/relayer/libraries/RelayerInternalStructs.sol";
import {WormholeRelayer} from "../../contracts/relayer/wormholeRelayer/WormholeRelayer.sol";
import {MockGenericRelayer} from "./MockGenericRelayer.sol";
import {MockWormhole} from "./MockWormhole.sol";
import {IWormhole} from "../../contracts/interfaces/IWormhole.sol";
import {WormholeSimulator, FakeWormholeSimulator} from "./WormholeSimulator.sol";
import {IWormholeReceiver} from "../../contracts/interfaces/relayer/IWormholeReceiver.sol";
import {
    MockRelayerIntegration,
    XAddress,
    DeliveryData
} from "../../contracts/mock/relayer/MockRelayerIntegration.sol";
import {BigRevertBufferIntegration} from "./BigRevertBufferIntegration.sol";
import {TestHelpers} from "./TestHelpers.sol";
import {WormholeRelayerSerde} from
    "../../contracts/relayer/wormholeRelayer/WormholeRelayerSerde.sol";
import {
    EvmExecutionInfoV1,
    ExecutionInfoVersion,
    decodeEvmExecutionInfoV1,
    encodeEvmExecutionInfoV1
} from "../../contracts/relayer/libraries/ExecutionParameters.sol";
import {toWormholeFormat, fromWormholeFormat} from "../../contracts/relayer/libraries/Utils.sol";
import {BytesParsing} from "../../contracts/relayer/libraries/BytesParsing.sol";
import "../../contracts/interfaces/relayer/TypedUnits.sol";

import "forge-std/Test.sol";
import "forge-std/console.sol";
import "forge-std/Vm.sol";

contract WormholeRelayerTests is Test {
    using BytesParsing for bytes;
    using WeiLib for Wei;
    using GasLib for Gas;
    using WeiPriceLib for WeiPrice;
    using GasPriceLib for GasPrice;
    using TargetNativeLib for TargetNative;
    using LocalNativeLib for LocalNative;

    Gas REASONABLE_GAS_LIMIT = Gas.wrap(500000);
    Gas REASONABLE_GAS_LIMIT_FORWARDS = Gas.wrap(1000000);
    Gas TOO_LOW_GAS_LIMIT = Gas.wrap(10000);

    struct GasParameters {
        uint32 evmGasOverhead;
        uint32 targetGasLimit;
        uint56 targetGasPrice;
        uint56 sourceGasPrice;
    }

    struct FeeParameters {
        uint56 targetNativePrice;
        uint56 sourceNativePrice;
        uint32 wormholeFeeOnSource;
        uint32 wormholeFeeOnTarget;
        uint64 receiverValueTarget;
    }

    struct GasParametersTyped {
        Gas evmGasOverhead;
        Gas targetGasLimit;
        GasPrice targetGasPrice;
        GasPrice sourceGasPrice;
    }

    struct FeeParametersTyped {
        WeiPrice targetNativePrice;
        WeiPrice sourceNativePrice;
        Wei wormholeFeeOnSource;
        Wei wormholeFeeOnTarget;
        Wei receiverValueTarget;
    }

    function toGasParametersTyped(GasParameters memory gasParams)
        internal
        pure
        returns (GasParametersTyped memory)
    {
        return GasParametersTyped({
            evmGasOverhead: Gas.wrap(gasParams.evmGasOverhead),
            targetGasLimit: Gas.wrap(gasParams.targetGasLimit),
            targetGasPrice: GasPrice.wrap(gasParams.targetGasPrice),
            sourceGasPrice: GasPrice.wrap(gasParams.sourceGasPrice)
        });
    }

    function toFeeParametersTyped(FeeParameters memory feeParams)
        internal
        pure
        returns (FeeParametersTyped memory)
    {
        return FeeParametersTyped({
            targetNativePrice: WeiPrice.wrap(feeParams.targetNativePrice),
            sourceNativePrice: WeiPrice.wrap(feeParams.sourceNativePrice),
            wormholeFeeOnSource: Wei.wrap(feeParams.wormholeFeeOnSource),
            wormholeFeeOnTarget: Wei.wrap(feeParams.wormholeFeeOnTarget),
            receiverValueTarget: Wei.wrap(feeParams.receiverValueTarget)
        });
    }

    IWormhole relayerWormhole;
    WormholeSimulator relayerWormholeSimulator;
    MockGenericRelayer genericRelayer;
    TestHelpers helpers;

    /**
     *
     *  SETUP
     *
     */

    function setUp() public {
        // deploy Wormhole
        MockWormhole wormhole = new MockWormhole({
            initChainId: 2,
            initEvmChainId: block.chainid
        });

        relayerWormhole = wormhole;
        relayerWormholeSimulator = new FakeWormholeSimulator(
            wormhole
        );

        helpers = new TestHelpers();

        genericRelayer =
            new MockGenericRelayer(address(wormhole), address(relayerWormholeSimulator));

        setUpChains(5);
    }

    struct StandardSetupTwoChains {
        uint16 sourceChain;
        uint16 targetChain;
        uint16 differentChainId;
        Contracts source;
        Contracts target;
    }

    function standardAssume(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        uint32 minTargetGasLimit
    ) public pure {
        vm.assume(gasParams.evmGasOverhead > 0);
        vm.assume(gasParams.targetGasLimit > 0);
        vm.assume(feeParams.targetNativePrice > 0);
        vm.assume(gasParams.targetGasPrice > 0);
        vm.assume(gasParams.sourceGasPrice > 0);
        vm.assume(feeParams.sourceNativePrice > 0);

        vm.assume(
            (
                uint256(gasParams.sourceGasPrice) * feeParams.sourceNativePrice
                    + feeParams.targetNativePrice - 1
            ) / uint256(feeParams.targetNativePrice) < type(uint88).max
        );
        vm.assume(
            (
                uint256(gasParams.targetGasPrice) * feeParams.targetNativePrice
                    + feeParams.sourceNativePrice - 1
            ) / uint256(feeParams.sourceNativePrice) < type(uint88).max
        );

        vm.assume(
            feeParams.targetNativePrice
                < (uint256(2) ** 126)
                    / (
                        uint256(1) * gasParams.targetGasPrice
                            * (uint256(0) + gasParams.targetGasLimit + gasParams.evmGasOverhead)
                            + feeParams.wormholeFeeOnTarget
                    )
        );
        vm.assume(
            feeParams.sourceNativePrice
                < (uint256(2) ** 126)
                    / (
                        uint256(1) * gasParams.sourceGasPrice
                            * (uint256(gasParams.targetGasLimit) + gasParams.evmGasOverhead)
                            + feeParams.wormholeFeeOnSource
                    )
        );

        vm.assume(gasParams.targetGasLimit >= minTargetGasLimit);
    }

    function standardAssumeAndSetupTwoChains(
        GasParameters memory gasParams_,
        FeeParameters memory feeParams_,
        uint32 minTargetGasLimit
    ) public returns (StandardSetupTwoChains memory s) {
        return standardAssumeAndSetupTwoChains(gasParams_, feeParams_, Gas.wrap(minTargetGasLimit));
    }

    function standardAssumeAndSetupTwoChains(
        GasParameters memory gasParams_,
        FeeParameters memory feeParams_,
        Gas minTargetGasLimit
    ) public returns (StandardSetupTwoChains memory s) {
        standardAssume(gasParams_, feeParams_, uint32(minTargetGasLimit.unwrap()));
        GasParametersTyped memory gasParams = toGasParametersTyped(gasParams_);
        FeeParametersTyped memory feeParams = toFeeParametersTyped(feeParams_);

        s.sourceChain = 1;
        s.targetChain = 2;
        s.differentChainId = 3;
        s.source = map[s.sourceChain];
        s.target = map[s.targetChain];

        vm.deal(s.source.relayer, type(uint256).max / 2);
        vm.deal(s.target.relayer, type(uint256).max / 2);
        vm.deal(address(this), type(uint256).max / 2);
        // vm.deal(address(s.target.integration), type(uint256).max / 2);
        // vm.deal(address(s.source.integration), type(uint256).max / 2);

        // set deliveryProvider prices
        s.source.deliveryProvider.updatePrice(
            s.targetChain, gasParams.targetGasPrice, feeParams.targetNativePrice
        );
        s.source.deliveryProvider.updatePrice(
            s.sourceChain, gasParams.sourceGasPrice, feeParams.sourceNativePrice
        );
        s.target.deliveryProvider.updatePrice(
            s.targetChain, gasParams.targetGasPrice, feeParams.targetNativePrice
        );
        s.target.deliveryProvider.updatePrice(
            s.sourceChain, gasParams.sourceGasPrice, feeParams.sourceNativePrice
        );

        s.source.deliveryProvider.updateDeliverGasOverhead(s.targetChain, gasParams.evmGasOverhead);
        s.target.deliveryProvider.updateDeliverGasOverhead(s.sourceChain, gasParams.evmGasOverhead);

        s.source.wormholeSimulator.setMessageFee(feeParams.wormholeFeeOnSource.unwrap());
        s.target.wormholeSimulator.setMessageFee(feeParams.wormholeFeeOnTarget.unwrap());
    }

    struct Contracts {
        IWormhole wormhole;
        WormholeSimulator wormholeSimulator;
        DeliveryProvider deliveryProvider;
        IWormholeRelayer coreRelayer;
        WormholeRelayer coreRelayerFull;
        MockRelayerIntegration integration;
        address relayer;
        address payable rewardAddress;
        address payable refundAddress;
        uint16 chainId;
    }

    mapping(uint16 => Contracts) map;

    function setUpChains(uint16 numChains) internal {
        for (uint16 i = 1; i <= numChains; i++) {
            Contracts memory mapEntry;
            (mapEntry.wormhole, mapEntry.wormholeSimulator) = helpers.setUpWormhole(i);
            mapEntry.deliveryProvider = helpers.setUpDeliveryProvider(i);
            mapEntry.deliveryProvider.updateSupportedMessageKeyTypes(VAA_KEY_TYPE, true);
            mapEntry.coreRelayer =
                helpers.setUpWormholeRelayer(mapEntry.wormhole, address(mapEntry.deliveryProvider));
            mapEntry.coreRelayerFull = WormholeRelayer(payable(address(mapEntry.coreRelayer)));
            genericRelayer.setWormholeRelayerContract(i, address(mapEntry.coreRelayer));
            mapEntry.integration =
            new MockRelayerIntegration(address(mapEntry.wormhole), address(mapEntry.coreRelayer));
            mapEntry.relayer =
                address(uint160(uint256(keccak256(abi.encodePacked(bytes("relayer"), i)))));
            genericRelayer.setProviderDeliveryAddress(i, mapEntry.relayer);
            mapEntry.refundAddress = payable(
                address(uint160(uint256(keccak256(abi.encodePacked(bytes("refundAddress"), i)))))
            );
            mapEntry.rewardAddress = payable(
                address(uint160(uint256(keccak256(abi.encodePacked(bytes("rewardAddress"), i)))))
            );
            mapEntry.chainId = i;
            map[i] = mapEntry;
        }

        uint192 maxBudget = uint192(2 ** 192 - 1);
        for (uint16 i = 1; i <= numChains; i++) {
            for (uint16 j = 1; j <= numChains; j++) {
                map[i].deliveryProvider.updateSupportedChain(j, true);
                map[i].deliveryProvider.updateAssetConversionBuffer(j, 500, 10000);
                map[i].deliveryProvider.updateTargetChainAddress(
                    j, bytes32(uint256(uint160(address(map[j].deliveryProvider))))
                );
                map[i].deliveryProvider.updateRewardAddress(map[i].rewardAddress);
                helpers.registerWormholeRelayerContract(
                    map[i].coreRelayerFull,
                    map[i].wormhole,
                    i,
                    j,
                    bytes32(uint256(uint160(address(map[j].coreRelayer))))
                );
                map[i].deliveryProvider.updateMaximumBudget(j, Wei.wrap(maxBudget));
                map[i].integration.registerEmitter(
                    j, bytes32(uint256(uint160(address(map[j].integration))))
                );
                XAddress[] memory addresses = new XAddress[](1);
                addresses[0] = XAddress(j, bytes32(uint256(uint160(address(map[j].integration)))));
                map[i].integration.registerEmitters(addresses);
            }
        }
    }

    function getDeliveryVAAHash(Vm.Log[] memory logs) internal pure returns (bytes32 vaaHash) {
        (vaaHash,) = logs[0].data.asBytes32(0);
    }

    function getDeliveryStatus(Vm.Log memory log)
        internal
        view
        returns (IWormholeRelayerDelivery.DeliveryStatus status)
    {
        (uint256 parsed,) = log.data.asUint256(32);
        console.log(parsed);
        status = IWormholeRelayerDelivery.DeliveryStatus(parsed);
    }

    function getDeliveryStatus()
        internal
        returns (IWormholeRelayerDelivery.DeliveryStatus status)
    {
        Vm.Log[] memory logs = vm.getRecordedLogs();
        status = getDeliveryStatus(logs[logs.length - 1]);
    }

    function getRefundStatus(Vm.Log memory log)
        internal
        pure
        returns (IWormholeRelayerDelivery.RefundStatus status)
    {
        (uint256 parsed,) = log.data.asUint256(32 + 32 + 32);
        status = IWormholeRelayerDelivery.RefundStatus(parsed);
    }

    function getRefundStatus() internal returns (IWormholeRelayerDelivery.RefundStatus status) {
        Vm.Log[] memory logs = vm.getRecordedLogs();
        status = getRefundStatus(logs[logs.length - 1]);
    }

    function vaaKeyArray(
        uint16 chainId,
        uint64 sequence,
        address emitterAddress
    ) internal pure returns (VaaKey[] memory vaaKeys) {
        vaaKeys = new VaaKey[](1);
        vaaKeys[0] = VaaKey(chainId, toWormholeFormat(emitterAddress), sequence);
    }

    function sendMessageToTargetChain(
        StandardSetupTwoChains memory setup,
        Gas gasLimit,
        uint128 receiverValue,
        bytes memory message
    ) internal returns (uint64 sequence) {
        return sendMessageToTargetChain(setup, uint32(gasLimit.unwrap()), receiverValue, message);
    }

    function sendMessageToTargetChain(
        StandardSetupTwoChains memory setup,
        uint32 gasLimit,
        uint128 receiverValue,
        bytes memory message
    ) internal returns (uint64 sequence) {
        (LocalNative deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(receiverValue), Gas.wrap(gasLimit)
        );
        sequence = setup.source.integration.sendMessage{value: LocalNative.unwrap(deliveryCost)}(
            message, setup.targetChain, gasLimit, receiverValue
        );
    }

    function sendMessageToTargetChainExpectingForwardedResponse(
        StandardSetupTwoChains memory setup,
        uint32 gasLimit,
        uint128 receiverValue,
        bytes memory message,
        bytes memory forwardedMessage,
        bool forwardShouldSucceed
    ) internal returns (uint64 sequence) {
        LocalNative forwardDeliveryCost;
        if (forwardShouldSucceed) {
            (forwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(
                setup.sourceChain, TargetNative.wrap(0), REASONABLE_GAS_LIMIT
            );
        } else {
            (forwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(
                setup.sourceChain, TargetNative.wrap(0), Gas.wrap(10_000)
            );
        }

        uint256 neededReceiverValue = forwardDeliveryCost.unwrap();
        vm.assume(neededReceiverValue <= type(uint128).max);
        if (forwardShouldSucceed) {
            vm.assume(receiverValue >= neededReceiverValue);
        } else {
            vm.assume(receiverValue < neededReceiverValue);
        }

        (LocalNative deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(receiverValue), Gas.wrap(gasLimit)
        );

        sequence = setup.source.integration.sendMessageWithForwardedResponse{
            value: deliveryCost.unwrap()
        }(message, forwardedMessage, setup.targetChain, gasLimit, receiverValue);
    }

    function resendMessageToTargetChain(
        StandardSetupTwoChains memory setup,
        uint64 sequence,
        uint32 gasLimit,
        uint128 receiverValue,
        bytes memory
    ) internal {
        (LocalNative newDeliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(receiverValue), Gas.wrap(gasLimit)
        );

        setup.source.integration.resend{value: newDeliveryCost.unwrap()}(
            setup.sourceChain, sequence, setup.targetChain, gasLimit, receiverValue
        );
    }

    /**
     *
     * TEST SUITE!
     *
     */

    /**
     * Basic Functionality Tests: Send, Forward, and Resend
     */

    function testSend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.SUCCESS);
    }

    function testSendWithResponse(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message,
        bytes memory forwardedMessage
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT_FORWARDS);

        vm.recordLogs();

        sendMessageToTargetChainExpectingForwardedResponse(
            setup,
            gasParams.targetGasLimit,
            feeParams.receiverValueTarget,
            message,
            forwardedMessage,
            true
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        genericRelayer.relay(setup.targetChain);

        assertTrue(keccak256(setup.source.integration.getMessage()) == keccak256(forwardedMessage));
    }

    function testResend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        vm.assume(keccak256(message) != keccak256(bytes("")));

        uint64 sequence = sendMessageToTargetChain(setup, TOO_LOW_GAS_LIMIT, 0, message);

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(message));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE);

        resendMessageToTargetChain(
            setup, sequence, uint32(REASONABLE_GAS_LIMIT.unwrap()), 0, message
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.SUCCESS);
    }

    /**
     * More functionality tests
     */

    function testRevertUnsupportedMessageKeyType(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(
            gasParams, feeParams, Gas.wrap(gasParams.targetGasLimit)
        );

        MessageKey[] memory messageKeys = new MessageKey[](2);
        messageKeys[0] = MessageKey(VAA_KEY_TYPE, bytes(""));
        messageKeys[1] = MessageKey(RANDOM_KEY_TYPE, RANDOM_KEY_TYPE_BODY);

        (LocalNative cost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(0), Gas.wrap(0)
        );

        vm.expectRevert(
            abi.encodeWithSelector(
                DeliveryProviderDoesNotSupportMessageKeyType.selector, RANDOM_KEY_TYPE
            )
        );
        setup.source.coreRelayer.sendToEvm{value: cost.unwrap()}(
            setup.targetChain,
            address(0x0),
            bytes(""),
            TargetNative.wrap(0),
            LocalNative.wrap(0),
            Gas.wrap(0),
            setup.sourceChain,
            address(0x0),
            address(setup.source.deliveryProvider),
            messageKeys,
            15
        );
    }

    function testForwardFailure(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        feeParams.receiverValueTarget = 0;
        vm.assume(
            uint256(20) * feeParams.targetNativePrice * gasParams.targetGasPrice
                < uint256(1) * feeParams.sourceNativePrice * gasParams.sourceGasPrice
        );

        vm.recordLogs();
        gasParams.targetGasLimit = 600000;
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(
            gasParams, feeParams, Gas.wrap(gasParams.targetGasLimit)
        );

        sendMessageToTargetChainExpectingForwardedResponse(
            setup,
            gasParams.targetGasLimit,
            feeParams.receiverValueTarget,
            bytes("Hello!"),
            bytes("Forwarded Message!"),
            false
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(bytes("Hello!")));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE);
        console.log(uint256(IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE));
    }

    function testMultipleForwards(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message,
        bytes memory forwardedMessage
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        (LocalNative firstForwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(
            setup.sourceChain, TargetNative.wrap(0), REASONABLE_GAS_LIMIT
        );
        (LocalNative secondForwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(0), REASONABLE_GAS_LIMIT
        );

        uint256 receiverValue =
            firstForwardDeliveryCost.unwrap() + secondForwardDeliveryCost.unwrap();
        vm.assume(receiverValue <= type(uint128).max);

        (LocalNative deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain,
            TargetNative.wrap(receiverValue),
            Gas.wrap(REASONABLE_GAS_LIMIT_FORWARDS.unwrap() * 2)
        );

        vm.recordLogs();

        setup.source.integration.sendMessageWithMultiForwardedResponse{value: deliveryCost.unwrap()}(
            message,
            forwardedMessage,
            setup.targetChain,
            uint32(REASONABLE_GAS_LIMIT_FORWARDS.unwrap() * 2),
            uint128(receiverValue)
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        genericRelayer.relay(setup.targetChain);

        assertTrue(keccak256(setup.source.integration.getMessage()) == keccak256(forwardedMessage));

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(forwardedMessage));
    }

    function testResendFailAndSucceed(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        vm.assume(keccak256(message) != keccak256(bytes("")));

        uint64 sequence = sendMessageToTargetChain(setup, TOO_LOW_GAS_LIMIT, 0, message);

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(message));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE);

        for (uint32 i = 2; i < 10; i++) {
            resendMessageToTargetChain(
                setup, sequence, uint32(TOO_LOW_GAS_LIMIT.unwrap() * i), 0, message
            );
            genericRelayer.relay(setup.sourceChain);
            assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(message));
            assertTrue(
                getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE
            );
        }

        resendMessageToTargetChain(
            setup, sequence, uint32(REASONABLE_GAS_LIMIT.unwrap()), 0, message
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.SUCCESS);
    }

    /**
     * Funds correct test (testing each address receives the correct payment)
     */

    struct FundsCorrectTest {
        uint256 refundAddressBalance;
        uint256 relayerBalance;
        uint256 rewardAddressBalance;
        uint256 destinationBalance;
        uint256 sourceContractBalance;
        uint256 targetContractBalance;
        uint128 receiverValue;
        uint256 deliveryPrice;
        uint256 targetChainRefundPerGasUnused;
        uint32 gasAmount;
        uint256 refundAddressAmount;
        uint256 relayerPayment;
        uint256 rewardAddressAmount;
        uint256 destinationAmount;
    }

    function setupFundsCorrectTest(
        GasParameters memory gasParams_,
        FeeParameters memory feeParams_,
        Gas minGasLimit
    ) public returns (StandardSetupTwoChains memory s, FundsCorrectTest memory test) {
        return setupFundsCorrectTest(gasParams_, feeParams_, uint32(minGasLimit.unwrap()));
    }

    function setupFundsCorrectTest(
        GasParameters memory gasParams_,
        FeeParameters memory feeParams_,
        uint32 minGasLimit
    ) public returns (StandardSetupTwoChains memory s, FundsCorrectTest memory test) {
        s = standardAssumeAndSetupTwoChains(gasParams_, feeParams_, Gas.wrap(minGasLimit));

        test.refundAddressBalance = s.target.refundAddress.balance;
        test.relayerBalance = s.target.relayer.balance;
        test.rewardAddressBalance = s.source.rewardAddress.balance;
        test.destinationBalance = address(s.target.integration).balance;
        test.sourceContractBalance = address(s.source.coreRelayer).balance;
        test.targetContractBalance = address(s.target.coreRelayer).balance;
        test.receiverValue = feeParams_.receiverValueTarget;
        (LocalNative deliveryPrice, GasPrice targetChainRefundPerGasUnused) = s
            .source
            .coreRelayer
            .quoteEVMDeliveryPrice(
            s.targetChain,
            TargetNative.wrap(test.receiverValue),
            Gas.wrap(gasParams_.targetGasLimit)
        );
        test.deliveryPrice = deliveryPrice.unwrap();
        test.targetChainRefundPerGasUnused = targetChainRefundPerGasUnused.unwrap();
        vm.assume(test.targetChainRefundPerGasUnused > 0);
    }

    function testFundsCorrectForASend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 170000);

        setup.source.integration.sendMessageWithRefund{value: test.deliveryPrice}(
            bytes("Hello!"),
            setup.targetChain,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.targetChain,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256("Hello!"));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.SUCCESS);

        test.refundAddressAmount = setup.target.refundAddress.balance - test.refundAddressBalance;
        test.rewardAddressAmount = setup.source.rewardAddress.balance - test.rewardAddressBalance;
        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;
        test.destinationAmount = address(setup.target.integration).balance - test.destinationBalance;

        assertTrue(test.sourceContractBalance == address(setup.source.coreRelayer).balance);
        assertTrue(test.targetContractBalance == address(setup.target.coreRelayer).balance);
        assertTrue(
            test.destinationAmount == test.receiverValue, "Receiver value was sent to the contract"
        );
        assertTrue(
            test.rewardAddressAmount + feeParams.wormholeFeeOnSource == test.deliveryPrice,
            "Reward address was paid correctly"
        );

        test.gasAmount = uint32(
            gasParams.targetGasLimit - test.refundAddressAmount / test.targetChainRefundPerGasUnused
        );
        console.log(test.gasAmount);
        assertTrue(
            test.gasAmount >= 140000,
            "Gas amount (calculated from refund address payment) lower than expected. NOTE: This assert is purely to ensure the gas usage is consistent, and thus (since this was computed using the refund amount) the refund amount is correct."
        );
        assertTrue(
            test.gasAmount <= 160000,
            "Gas amount (calculated from refund address payment) higher than expected. NOTE: This assert is purely to ensure the gas usage is consistent, and thus (since this was computed using the refund amount) the refund amount is correct."
        );
        assertTrue(
            test.relayerPayment == test.destinationAmount + test.refundAddressAmount,
            "Relayer paid the correct amount"
        );
    }

    function testFundsCorrectForASendFailureDueToGasExceeded(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        gasParams.targetGasLimit = uint32(TOO_LOW_GAS_LIMIT.unwrap());
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 0);

        setup.source.integration.sendMessageWithRefund{value: test.deliveryPrice}(
            bytes("Hello!"),
            setup.targetChain,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.targetChain,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(bytes("Hello!")));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE);

        test.refundAddressAmount = setup.target.refundAddress.balance - test.refundAddressBalance;
        test.rewardAddressAmount = setup.source.rewardAddress.balance - test.rewardAddressBalance;
        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;
        test.destinationAmount = address(setup.target.integration).balance - test.destinationBalance;

        assertTrue(test.sourceContractBalance == address(setup.source.coreRelayer).balance);
        assertTrue(test.targetContractBalance == address(setup.target.coreRelayer).balance);
        assertTrue(test.destinationAmount == 0, "No receiver value was sent to the contract");
        assertTrue(
            test.rewardAddressAmount + feeParams.wormholeFeeOnSource == test.deliveryPrice,
            "Reward address was paid correctly"
        );
        assertTrue(test.refundAddressAmount == test.receiverValue, "Receiver value was refunded");
        assertTrue(
            test.relayerPayment == test.destinationAmount + test.refundAddressAmount,
            "Relayer paid the correct amount"
        );
    }

    function testFundsCorrectForASendFailureDueToRevert(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, REASONABLE_GAS_LIMIT_FORWARDS);

        setup.target.deliveryProvider.updateSupportedChain(1, false);

        setup.source.integration.sendMessageWithForwardedResponse{value: test.deliveryPrice}(
            bytes("Hello!"),
            bytes("Forwarded Message"),
            setup.targetChain,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.targetChain,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(bytes("Hello!")));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE);

        test.refundAddressAmount = setup.target.refundAddress.balance - test.refundAddressBalance;
        test.rewardAddressAmount = setup.source.rewardAddress.balance - test.rewardAddressBalance;
        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;
        test.destinationAmount = address(setup.target.integration).balance - test.destinationBalance;

        assertTrue(test.sourceContractBalance == address(setup.source.coreRelayer).balance);
        assertTrue(test.targetContractBalance == address(setup.target.coreRelayer).balance);
        assertTrue(test.destinationAmount == 0, "No receiver value was sent to the contract");
        assertTrue(
            test.rewardAddressAmount + feeParams.wormholeFeeOnSource == test.deliveryPrice,
            "Reward address was paid correctly"
        );
        test.gasAmount = uint32(
            gasParams.targetGasLimit
                - (test.refundAddressAmount - test.receiverValue) / test.targetChainRefundPerGasUnused
        );
        console.log(test.gasAmount);
        assertTrue(
            test.gasAmount >= 165_000,
            "Gas amount (calculated from refund address payment) lower than expected. NOTE: This assert is purely to ensure the gas usage is consistent, and thus (since this was computed using the refund amount) the refund amount is correct."
        );
        assertTrue(
            test.gasAmount <= 280_000,
            "Gas amount (calculated from refund address payment) higher than expected. NOTE: This assert is purely to ensure the gas usage is consistent, and thus (since this was computed using the refund amount) the refund amount is correct."
        );
        assertTrue(
            test.relayerPayment == test.destinationAmount + test.refundAddressAmount,
            "Relayer paid the correct amount"
        );
    }

    function testFundsCorrectForASendCrossChainRefundSuccess(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 170000);

        uint256 refundRewardAddressBalance = setup.target.rewardAddress.balance;
        uint256 refundAddressBalance = setup.source.refundAddress.balance;

        setup.source.integration.sendMessageWithRefund{value: test.deliveryPrice}(
            bytes("Hello!"),
            setup.targetChain,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.sourceChain,
            setup.source.refundAddress
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(bytes("Hello!")));

        genericRelayer.relay(setup.targetChain);

        assertTrue(
            test.deliveryPrice
                == setup.source.rewardAddress.balance - test.rewardAddressBalance
                    + feeParams.wormholeFeeOnSource,
            "The source to target relayer's reward address was paid appropriately"
        );

        uint256 amountToGetInRefundTarget =
            (setup.target.rewardAddress.balance - refundRewardAddressBalance);

        vm.assume(amountToGetInRefundTarget > 0);

        uint256 refundSource;
        (LocalNative baseFee,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(
            setup.sourceChain, TargetNative.wrap(0), Gas.wrap(0)
        );

        TargetNative tmp = setup.target.coreRelayer.quoteNativeForChain(
            setup.sourceChain,
            LocalNative.wrap(
                amountToGetInRefundTarget + feeParams.wormholeFeeOnTarget - baseFee.unwrap()
            ),
            setup.target.coreRelayer.getDefaultDeliveryProvider()
        );
        refundSource = tmp.unwrap();

        // Calculate amount that must have been spent on gas, by reverse engineering from the amount that was paid to the provider's reward address on the target chain
        test.gasAmount = uint32(
            gasParams.targetGasLimit
                - (amountToGetInRefundTarget + feeParams.wormholeFeeOnTarget)
                    / test.targetChainRefundPerGasUnused
        );
        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;
        test.destinationAmount = address(setup.target.integration).balance - test.destinationBalance;

        assertTrue(
            test.destinationAmount == feeParams.receiverValueTarget,
            "Receiver value was sent to the contract"
        );
        assertTrue(
            test.relayerPayment
                == amountToGetInRefundTarget + feeParams.wormholeFeeOnTarget
                    + feeParams.receiverValueTarget,
            "Relayer paid the correct amount"
        );
        assertTrue(
            refundSource == setup.source.refundAddress.balance - refundAddressBalance,
            "Refund wasn't the correct amount"
        );
        console.log(test.gasAmount);
        assertTrue(
            test.gasAmount >= 140000,
            "Gas amount (calculated from refund address payment) lower than expected. NOTE: This assert is purely to ensure the gas usage is consistent, and thus (since this was computed using the refund amount) the refund amount is correct."
        );
        assertTrue(
            test.gasAmount <= 160000,
            "Gas amount (calculated from refund address payment) higher than expected. NOTE: This assert is purely to ensure the gas usage is consistent, and thus (since this was computed using the refund amount) the refund amount is correct."
        );
    }

    function testFundsCorrectForASendCrossChainRefundFailProviderNotSupported(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) = setupFundsCorrectTest(
            gasParams, feeParams, uint32(170000 + REASONABLE_GAS_LIMIT.unwrap())
        );

        setup.target.deliveryProvider.updateSupportedChain(setup.sourceChain, false);
        vm.assume(
            test.targetChainRefundPerGasUnused * REASONABLE_GAS_LIMIT.unwrap()
                >= feeParams.wormholeFeeOnTarget
                    + uint256(1) * gasParams.evmGasOverhead * gasParams.sourceGasPrice
                        * (uint256(feeParams.sourceNativePrice) / feeParams.targetNativePrice + 1)
        );

        setup.source.integration.sendMessageWithRefund{value: test.deliveryPrice}(
            bytes("Hello!"),
            setup.targetChain,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.sourceChain,
            setup.source.refundAddress
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(bytes("Hello!")));

        assertTrue(
            test.deliveryPrice
                == setup.source.rewardAddress.balance - test.rewardAddressBalance
                    + feeParams.wormholeFeeOnSource,
            "The source to target relayer's reward address was paid appropriately"
        );

        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;
        test.destinationAmount = address(setup.target.integration).balance - test.destinationBalance;

        assertTrue(
            test.destinationAmount == feeParams.receiverValueTarget,
            "Receiver value was sent to the contract"
        );
        assertTrue(
            test.relayerPayment == feeParams.receiverValueTarget,
            "Relayer only paid the receiver value, and received the full transaction fee refund"
        );
        uint8 refundStatus = uint8(getRefundStatus());
        assertTrue(
            refundStatus
                == uint8(
                    IWormholeRelayerDelivery.RefundStatus.CROSS_CHAIN_REFUND_FAIL_PROVIDER_NOT_SUPPORTED
                )
        );
    }

    function testFundsCorrectForASendCrossChainRefundNotEnough(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 170000);
        vm.assume(
            uint256(1) * gasParams.evmGasOverhead * gasParams.sourceGasPrice
                * feeParams.sourceNativePrice
                > uint256(1) * feeParams.targetNativePrice * test.targetChainRefundPerGasUnused
                    * gasParams.targetGasLimit
        );

        setup.source.integration.sendMessageWithRefund{value: test.deliveryPrice}(
            bytes("Hello!"),
            setup.targetChain,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.sourceChain,
            setup.source.refundAddress
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(bytes("Hello!")));

        assertTrue(
            test.deliveryPrice
                == setup.source.rewardAddress.balance - test.rewardAddressBalance
                    + feeParams.wormholeFeeOnSource,
            "The source to target relayer's reward address was paid appropriately"
        );

        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;
        test.destinationAmount = address(setup.target.integration).balance - test.destinationBalance;

        assertTrue(
            test.destinationAmount == feeParams.receiverValueTarget,
            "Receiver value was sent to the contract"
        );
        assertTrue(
            test.relayerPayment == feeParams.receiverValueTarget,
            "Relayer only paid the receiver value, and received the full transaction fee refund"
        );

        assertTrue(
            uint8(getRefundStatus())
                == uint8(IWormholeRelayerDelivery.RefundStatus.CROSS_CHAIN_REFUND_FAIL_NOT_ENOUGH)
        );
    }

    /**
     * Unit tests for Send and Resend: Ensuring the correct struct is logged
     */

    struct UnitTestParams {
        address targetAddress;
        bytes payload;
        uint128 receiverValue;
        uint128 paymentForExtraReceiverValue;
        uint32 gasLimit;
        uint16 refundChain;
        address refundAddress;
        VaaKey[3] vaaKeysFixed;
    }

    uint8 constant RANDOM_KEY_TYPE = 159;
    bytes constant RANDOM_KEY_TYPE_BODY = hex"12911894719274912740817248912740817240";

    function testUnitTestSend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        UnitTestParams memory params
    ) public {
        gasParams.targetGasLimit = params.gasLimit;
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, params.gasLimit);
        MessageKey[] memory messageKeys = new MessageKey[](4);
        for (uint256 j = 0; j < 3; j++) {
            messageKeys[j] =
                MessageKey(VAA_KEY_TYPE, WormholeRelayerSerde.encodeVaaKey(params.vaaKeysFixed[j]));
        }
        setup.source.deliveryProvider.updateSupportedMessageKeyTypes(RANDOM_KEY_TYPE, true);
        messageKeys[3] = MessageKey(RANDOM_KEY_TYPE, RANDOM_KEY_TYPE_BODY); // random keyType and encodedKey
        vm.recordLogs();

        (LocalNative deliveryCost, GasPrice targetChainRefundPerGasUnused) = setup
            .source
            .coreRelayer
            .quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(params.receiverValue), Gas.wrap(params.gasLimit)
        );
        uint256 value = deliveryCost.unwrap() + params.paymentForExtraReceiverValue;
        setup.source.integration.sendToEvm{value: value}(
            setup.targetChain,
            params.targetAddress,
            params.gasLimit,
            params.refundChain,
            params.refundAddress,
            params.receiverValue,
            params.paymentForExtraReceiverValue,
            params.payload,
            messageKeys
        );

        bytes memory encodedExecutionInfo = abi.encode(
            uint8(ExecutionInfoVersion.EVM_V1), params.gasLimit, targetChainRefundPerGasUnused
        );
        TargetNative extraReceiverValue = setup.source.coreRelayer.quoteNativeForChain(
            setup.targetChain,
            LocalNative.wrap(params.paymentForExtraReceiverValue),
            address(setup.source.deliveryProvider)
        );

        DeliveryInstruction memory expectedInstruction = DeliveryInstruction({
            targetChain: setup.targetChain,
            targetAddress: toWormholeFormat(params.targetAddress),
            payload: params.payload,
            requestedReceiverValue: TargetNative.wrap(params.receiverValue),
            extraReceiverValue: extraReceiverValue,
            encodedExecutionInfo: encodedExecutionInfo,
            refundChain: params.refundChain,
            refundAddress: toWormholeFormat(params.refundAddress),
            refundDeliveryProvider: setup.source.deliveryProvider.getTargetChainAddress(
                setup.targetChain
                ),
            sourceDeliveryProvider: toWormholeFormat(address(setup.source.deliveryProvider)),
            senderAddress: toWormholeFormat(address(setup.source.integration)),
            messageKeys: messageKeys
        });

        checkInstructionEquality(
            relayerWormholeSimulator.parseVMFromLogs(vm.getRecordedLogs()[0]).payload,
            expectedInstruction
        );
    }

    struct UnitTestResendParams {
        VaaKey deliveryVaaKey;
        uint128 newReceiverValue;
        uint32 newGasLimit;
        address senderAddress;
    }

    function testUnitTestResend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        UnitTestResendParams memory params
    ) public {
        gasParams.targetGasLimit = params.newGasLimit;
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, params.newGasLimit);

        vm.recordLogs();

        (LocalNative deliveryCost, GasPrice targetChainRefundPerGasUnused) = setup
            .source
            .coreRelayer
            .quoteEVMDeliveryPrice(
            setup.targetChain,
            TargetNative.wrap(params.newReceiverValue),
            Gas.wrap(params.newGasLimit)
        );
        uint256 value = deliveryCost.unwrap();
        vm.deal(params.senderAddress, value);
        vm.prank(params.senderAddress);
        setup.source.coreRelayer.resendToEvm{value: value}(
            params.deliveryVaaKey,
            setup.targetChain,
            TargetNative.wrap(params.newReceiverValue),
            Gas.wrap(params.newGasLimit),
            address(setup.source.deliveryProvider)
        );

        bytes memory encodedExecutionInfo = abi.encode(
            uint8(ExecutionInfoVersion.EVM_V1), params.newGasLimit, targetChainRefundPerGasUnused
        );

        RedeliveryInstruction memory expectedInstruction = RedeliveryInstruction({
            deliveryVaaKey: params.deliveryVaaKey,
            targetChain: setup.targetChain,
            newRequestedReceiverValue: TargetNative.wrap(params.newReceiverValue),
            newEncodedExecutionInfo: encodedExecutionInfo,
            newSourceDeliveryProvider: toWormholeFormat(address(setup.source.deliveryProvider)),
            newSenderAddress: toWormholeFormat(params.senderAddress)
        });

        checkRedeliveryInstructionEquality(
            relayerWormholeSimulator.parseVMFromLogs(vm.getRecordedLogs()[0]).payload,
            expectedInstruction
        );
    }

    function checkMessageKey(
        bytes memory data,
        uint256 _index,
        MessageKey memory messageKey
    ) public returns (uint256 index) {
        MessageKey memory decodedMessageKey;
        index = _index;
        (decodedMessageKey.keyType, index) = data.asUint8(index);
        assertEq(
            decodedMessageKey.keyType, messageKey.keyType, "decodedMessageKey.keyType incorrect"
        );
        if (decodedMessageKey.keyType == VAA_KEY_TYPE) {
            (VaaKey memory vaaKey,) = WormholeRelayerSerde.decodeVaaKey(messageKey.encodedKey, 0);
            index = checkVaaKey(data, index, vaaKey);
        } else if (decodedMessageKey.keyType == RANDOM_KEY_TYPE) {
            uint32 encodedKeyLen;
            (encodedKeyLen, index) = data.asUint32(index);
            bytes memory encodedKey;
            (encodedKey, index) = data.sliceUnchecked(index, encodedKeyLen);
            assertEq(
                encodedKeyLen,
                RANDOM_KEY_TYPE_BODY.length,
                "encodedKeyLen for RANDOM_KEY_TYPE must be RANDOM_KEY_TYPE_BODY.length"
            );
            assertEq(encodedKey, messageKey.encodedKey, "decoded encodedKey must equal expected");
            assertEq(encodedKey, RANDOM_KEY_TYPE_BODY);
        } else {
            assertFalse(true, "Unsupported keyType found");
        }
    }

    function checkVaaKey(
        bytes memory data,
        uint256 _index,
        VaaKey memory vaaKey
    ) public returns (uint256 index) {
        VaaKey memory decodedVaaKey;
        index = _index;
        console.log(data.length, index);
        (decodedVaaKey.chainId, index) = data.asUint16(index);
        console.log(data.length, index);
        assertTrue(decodedVaaKey.chainId == vaaKey.chainId, "Wrong chain id");
        (decodedVaaKey.emitterAddress, index) = data.asBytes32(index);
        console.log(data.length, index);
        assertTrue(decodedVaaKey.emitterAddress == vaaKey.emitterAddress, "Wrong emitter address");
        (decodedVaaKey.sequence, index) = data.asUint64(index);
        console.log(data.length, index);
        assertTrue(decodedVaaKey.sequence == vaaKey.sequence, "Wrong sequence");
    }

    function checkInstructionEquality(
        bytes memory data,
        DeliveryInstruction memory expectedInstruction
    ) public {
        uint256 index = 0;
        uint32 length = 0;
        uint8 payloadId;
        DeliveryInstruction memory decodedInstruction;
        (payloadId, index) = data.asUint8(index);
        assertTrue(payloadId == 1, "Is a delivery instruction");
        (decodedInstruction.targetChain, index) = data.asUint16(index);
        assertTrue(
            decodedInstruction.targetChain == expectedInstruction.targetChain,
            "Wrong target chain id"
        );
        (decodedInstruction.targetAddress, index) = data.asBytes32(index);
        assertTrue(
            decodedInstruction.targetAddress == expectedInstruction.targetAddress,
            "Wrong target address"
        );
        (length, index) = data.asUint32(index);
        (decodedInstruction.payload, index) = data.slice(index, length);
        assertTrue(
            keccak256(decodedInstruction.payload) == keccak256(expectedInstruction.payload),
            "Wrong payload"
        );
        uint256 requestedReceiverValue;
        uint256 extraReceiverValue;
        (requestedReceiverValue, index) = data.asUint256(index);
        assertTrue(
            requestedReceiverValue == expectedInstruction.requestedReceiverValue.unwrap(),
            "Wrong requested receiver value"
        );
        (extraReceiverValue, index) = data.asUint256(index);
        assertTrue(
            extraReceiverValue == expectedInstruction.extraReceiverValue.unwrap(),
            "Wrong extra receiver value"
        );
        (length, index) = data.asUint32(index);
        (decodedInstruction.encodedExecutionInfo, index) = data.slice(index, length);
        assertTrue(
            keccak256(decodedInstruction.encodedExecutionInfo)
                == keccak256(expectedInstruction.encodedExecutionInfo),
            "Wrong encoded execution info"
        );
        (decodedInstruction.refundChain, index) = data.asUint16(index);
        assertTrue(
            decodedInstruction.refundChain == expectedInstruction.refundChain,
            "Wrong refund chain id"
        );
        (decodedInstruction.refundAddress, index) = data.asBytes32(index);
        assertTrue(
            decodedInstruction.refundAddress == expectedInstruction.refundAddress,
            "Wrong refund address"
        );
        (decodedInstruction.refundDeliveryProvider, index) = data.asBytes32(index);
        assertTrue(
            decodedInstruction.refundDeliveryProvider == expectedInstruction.refundDeliveryProvider,
            "Wrong refund relay provider"
        );
        (decodedInstruction.sourceDeliveryProvider, index) = data.asBytes32(index);
        assertTrue(
            decodedInstruction.sourceDeliveryProvider == expectedInstruction.sourceDeliveryProvider,
            "Wrong source relay provider"
        );
        (decodedInstruction.senderAddress, index) = data.asBytes32(index);
        assertTrue(
            decodedInstruction.senderAddress == expectedInstruction.senderAddress,
            "Wrong sender address"
        );
        uint8 messageKeysLength;
        (messageKeysLength, index) = data.asUint8(index);
        decodedInstruction.messageKeys = new MessageKey[](messageKeysLength);
        for (uint256 i = 0; i < messageKeysLength; i++) {
            index = checkMessageKey(data, index, expectedInstruction.messageKeys[i]);
        }
        assertTrue(index == data.length, "Wrong length of data");
    }

    function checkRedeliveryInstructionEquality(
        bytes memory data,
        RedeliveryInstruction memory expectedInstruction
    ) public {
        uint256 index = 0;
        uint32 length = 0;
        uint8 payloadId;
        RedeliveryInstruction memory decodedInstruction;
        (payloadId, index) = data.asUint8(index);
        assertTrue(payloadId == 2, "Is a redelivery instruction");
        uint8 vaaKeyType;
        (vaaKeyType, index) = data.asUint8(index);
        assertTrue(vaaKeyType == 1, "Is a vaa key");
        index = checkVaaKey(data, index, expectedInstruction.deliveryVaaKey);
        (decodedInstruction.targetChain, index) = data.asUint16(index);
        assertTrue(
            decodedInstruction.targetChain == expectedInstruction.targetChain,
            "Wrong target chain id"
        );
        uint256 requestedReceiverValue;
        (requestedReceiverValue, index) = data.asUint256(index);
        assertTrue(
            requestedReceiverValue == expectedInstruction.newRequestedReceiverValue.unwrap(),
            "Wrong requested receiver value"
        );
        (length, index) = data.asUint32(index);
        (decodedInstruction.newEncodedExecutionInfo, index) = data.slice(index, length);
        assertTrue(
            keccak256(decodedInstruction.newEncodedExecutionInfo)
                == keccak256(expectedInstruction.newEncodedExecutionInfo),
            "Wrong encoded execution info"
        );
        (decodedInstruction.newSourceDeliveryProvider, index) = data.asBytes32(index);
        assertTrue(
            decodedInstruction.newSourceDeliveryProvider
                == expectedInstruction.newSourceDeliveryProvider,
            "Wrong source relay provider"
        );
        (decodedInstruction.newSenderAddress, index) = data.asBytes32(index);
        assertTrue(
            decodedInstruction.newSenderAddress == expectedInstruction.newSenderAddress,
            "Wrong sender address"
        );
        assertTrue(index == data.length, "Wrong length of data");
    }

    /**
     * Tests related to reverts in deliver()
     *
     */

    function invalidateVM(bytes memory message, WormholeSimulator simulator) internal {
        change(message, message.length - 1);
        simulator.invalidateVM(message);
    }

    function change(bytes memory message, uint256 index) internal pure {
        if (message[index] == 0x02) {
            message[index] = 0x04;
        } else {
            message[index] = 0x02;
        }
    }

    struct DeliveryStack {
        bytes32 deliveryVaaHash;
        uint256 payment;
        Vm.Log[] entries;
        bytes encodedDeliveryVAA;
        bytes[] encodedVMs;
        IWormhole.VM parsed;
        uint256 budget;
        address payable relayerRefundAddress;
        DeliveryInstruction instruction;
    }

    function prepareDeliveryStack(
        DeliveryStack memory stack,
        StandardSetupTwoChains memory setup,
        uint256 numVaas
    ) internal {
        stack.entries = vm.getRecordedLogs();
        stack.encodedVMs = new bytes[](0);

        stack.encodedDeliveryVAA = relayerWormholeSimulator.fetchSignedMessageFromLogs(
            stack.entries[numVaas], setup.sourceChain, address(setup.source.coreRelayer)
        );

        stack.relayerRefundAddress = payable(setup.target.relayer);
        stack.parsed = relayerWormhole.parseVM(stack.encodedDeliveryVAA);
        stack.instruction = WormholeRelayerSerde.decodeDeliveryInstruction(stack.parsed.payload);
        stack.deliveryVaaHash = stack.parsed.hash;
        EvmExecutionInfoV1 memory executionInfo =
            decodeEvmExecutionInfoV1(stack.instruction.encodedExecutionInfo);
        stack.budget = Wei.unwrap(
            executionInfo.gasLimit.toWei(executionInfo.targetChainRefundPerGasUnused)
                + stack.instruction.extraReceiverValue.asNative()
        );
    }

    function testRevertDeliveryInvalidDeliveryVAA(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        prepareDeliveryStack(stack, setup, 0);

        bytes memory fakeVM = abi.encodePacked(stack.encodedDeliveryVAA);

        invalidateVM(fakeVM, setup.target.wormholeSimulator);

        stack.encodedDeliveryVAA = fakeVM;

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("InvalidDeliveryVaa(string)", ""));
        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs, stack.encodedDeliveryVAA, stack.relayerRefundAddress, bytes("")
        );
    }

    function testRevertDeliveryInvalidEmitter(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        prepareDeliveryStack(stack, setup, 0);

        // Create valid VAA with wrong emitter address
        IWormhole.VM memory vm_ = relayerWormholeSimulator.parseVMFromLogs(stack.entries[0]);
        vm_.version = uint8(1);
        vm_.timestamp = uint32(block.timestamp);
        vm_.emitterChainId = setup.sourceChain;
        vm_.emitterAddress = toWormholeFormat(address(setup.source.integration));
        bytes memory deliveryVaaWithWrongEmitter =
            relayerWormholeSimulator.encodeAndSignMessage(vm_);

        vm.prank(setup.target.relayer);
        vm.expectRevert(
            abi.encodeWithSelector(
                InvalidEmitter.selector,
                setup.source.integration,
                setup.source.coreRelayer,
                setup.source.chainId
            )
        );
        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs, deliveryVaaWithWrongEmitter, stack.relayerRefundAddress, bytes("")
        );
    }

    function testRevertDeliveryInsufficientRelayerFunds(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        vm.assume(gasParams.targetGasPrice > 1);

        DeliveryStack memory stack;

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        prepareDeliveryStack(stack, setup, 0);

        vm.prank(setup.target.relayer);
        vm.expectRevert(
            abi.encodeWithSelector(
                InsufficientRelayerFunds.selector, stack.budget - 1, stack.budget
            )
        );
        setup.target.coreRelayerFull.deliver{value: stack.budget - 1}(
            stack.encodedVMs, stack.encodedDeliveryVAA, stack.relayerRefundAddress, bytes("")
        );
    }

    function testRevertDeliveryTargetChainIsNotThisChain(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        prepareDeliveryStack(stack, setup, 0);

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("TargetChainIsNotThisChain(uint16)", 2));
        map[setup.differentChainId].coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs, stack.encodedDeliveryVAA, stack.relayerRefundAddress, bytes("")
        );
    }

    // aka: replay protection doesn't fire when it shouldn't
    function testNoFalseFiresReplayProtection(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        prepareDeliveryStack(stack, setup, 0);

        vm.prank(setup.target.relayer);
        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs, stack.encodedDeliveryVAA, stack.relayerRefundAddress, bytes("")
        );
        assertEq(
            setup.target.coreRelayerFull.deliverySuccessBlock(stack.deliveryVaaHash), block.number
        );
        assertEq(setup.target.coreRelayerFull.deliveryFailureBlock(stack.deliveryVaaHash), 0);

        vm.recordLogs();
        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        prepareDeliveryStack(stack, setup, 0);

        vm.prank(setup.target.relayer);
        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs, stack.encodedDeliveryVAA, stack.relayerRefundAddress, bytes("")
        );
        assertEq(
            setup.target.coreRelayerFull.deliverySuccessBlock(stack.deliveryVaaHash), block.number
        );
        assertEq(setup.target.coreRelayerFull.deliveryFailureBlock(stack.deliveryVaaHash), 0);
    }

    function testReplayProtection(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        prepareDeliveryStack(stack, setup, 0);

        vm.prank(setup.target.relayer);
        assertFalse(setup.target.coreRelayerFull.deliveryAttempted(stack.deliveryVaaHash));
        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs, stack.encodedDeliveryVAA, stack.relayerRefundAddress, bytes("")
        );
        assertTrue(setup.target.coreRelayerFull.deliveryAttempted(stack.deliveryVaaHash));
        assertEq(
            setup.target.coreRelayerFull.deliverySuccessBlock(stack.deliveryVaaHash), block.number
        );
        assertEq(setup.target.coreRelayerFull.deliveryFailureBlock(stack.deliveryVaaHash), 0);

        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs, stack.encodedDeliveryVAA, stack.relayerRefundAddress, bytes("")
        );

        assertTrue(
            getDeliveryStatus()
                == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE,
            "Should have failed due to Replay Protection"
        );
    }

    function testRevertDeliveryVaaKeysLengthDoesNotMatchVaasLength(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        prepareDeliveryStack(stack, setup, 0);

        stack.encodedVMs = new bytes[](1);
        stack.encodedVMs[0] = stack.encodedDeliveryVAA;

        vm.prank(setup.target.relayer);
        vm.expectRevert(
            abi.encodeWithSignature(
                "MessageKeysLengthDoesNotMatchMessagesLength(uint256,uint256)", 0, 1
            )
        );
        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs, stack.encodedDeliveryVAA, stack.relayerRefundAddress, bytes("")
        );
    }

    function testRevertDeliveryVaaKeysDoNotMatchVaas(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        (LocalNative payment_,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(0), Gas.wrap(gasParams.targetGasLimit)
        );
        stack.payment = payment_.unwrap();

        uint64 sequence = setup.source.wormhole.publishMessage{value: feeParams.wormholeFeeOnSource}(
            1, bytes(""), 200
        );
        setup.source.integration.sendToEvm{value: stack.payment}(
            setup.targetChain,
            address(setup.target.integration),
            gasParams.targetGasLimit,
            setup.sourceChain,
            address(this),
            0,
            0,
            message,
            vaaKeyArray(setup.sourceChain, sequence, address(this))
        );

        prepareDeliveryStack(stack, setup, 1);

        stack.encodedVMs = new bytes[](1);
        stack.encodedVMs[0] = stack.encodedDeliveryVAA;

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("VaaKeysDoNotMatchVaas(uint8)", 0));
        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs, stack.encodedDeliveryVAA, stack.relayerRefundAddress, bytes("")
        );
    }

    /**
     * Tests related to reverts due to delivering with deliveryOverrides
     */

    function testDeliveryWithOverrides(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        prepareDeliveryStack(stack, setup, 0);

        DeliveryOverride memory deliveryOverride = DeliveryOverride(
            stack.instruction.requestedReceiverValue,
            stack.instruction.encodedExecutionInfo,
            stack.deliveryVaaHash //really redeliveryHash
        );

        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs,
            stack.encodedDeliveryVAA,
            stack.relayerRefundAddress,
            WormholeRelayerSerde.encode(deliveryOverride)
        );
        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    function testRevertDeliveryWithOverrideGasLimit(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        prepareDeliveryStack(stack, setup, 0);

        EvmExecutionInfoV1 memory executionInfo =
            decodeEvmExecutionInfoV1(stack.instruction.encodedExecutionInfo);

        DeliveryOverride memory deliveryOverride = DeliveryOverride(
            stack.instruction.requestedReceiverValue,
            encodeEvmExecutionInfoV1(
                EvmExecutionInfoV1({
                    gasLimit: executionInfo.gasLimit - Gas.wrap(1),
                    targetChainRefundPerGasUnused: executionInfo.targetChainRefundPerGasUnused
                })
            ),
            stack.deliveryVaaHash //really redeliveryHash
        );

        vm.expectRevert(abi.encodeWithSignature("InvalidOverrideGasLimit()"));
        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs,
            stack.encodedDeliveryVAA,
            stack.relayerRefundAddress,
            WormholeRelayerSerde.encode(deliveryOverride)
        );
    }

    function testRevertDeliveryWithOverrideReceiverValue(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        vm.assume(feeParams.receiverValueTarget > 0);

        DeliveryStack memory stack;

        sendMessageToTargetChain(
            setup, gasParams.targetGasLimit, feeParams.receiverValueTarget, message
        );

        prepareDeliveryStack(stack, setup, 0);

        DeliveryOverride memory deliveryOverride = DeliveryOverride(
            stack.instruction.requestedReceiverValue - TargetNative.wrap(1),
            stack.instruction.encodedExecutionInfo,
            stack.deliveryVaaHash //really redeliveryHash
        );

        vm.expectRevert(abi.encodeWithSignature("InvalidOverrideReceiverValue()"));
        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs,
            stack.encodedDeliveryVAA,
            stack.relayerRefundAddress,
            WormholeRelayerSerde.encode(deliveryOverride)
        );
    }

    function testAllowDeliveryWithOverrideMaximumRefund(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.assume(gasParams.targetGasPrice > 1);

        vm.recordLogs();

        DeliveryStack memory stack;

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        prepareDeliveryStack(stack, setup, 0);

        EvmExecutionInfoV1 memory executionInfo =
            decodeEvmExecutionInfoV1(stack.instruction.encodedExecutionInfo);

        DeliveryOverride memory deliveryOverride = DeliveryOverride(
            stack.instruction.requestedReceiverValue,
            encodeEvmExecutionInfoV1(
                EvmExecutionInfoV1({
                    gasLimit: executionInfo.gasLimit,
                    targetChainRefundPerGasUnused: GasPrice.wrap(
                        executionInfo.targetChainRefundPerGasUnused.unwrap() - 1
                        )
                })
            ),
            stack.deliveryVaaHash //really redeliveryHash
        );

        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs,
            stack.encodedDeliveryVAA,
            stack.relayerRefundAddress,
            WormholeRelayerSerde.encode(deliveryOverride)
        );
    }

    function testRevertDeliveryWithOverrideUnexpectedExecutionInfoVersion(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        vm.assume(feeParams.receiverValueTarget > 0);

        DeliveryStack memory stack;

        sendMessageToTargetChain(
            setup, gasParams.targetGasLimit, feeParams.receiverValueTarget, message
        );

        prepareDeliveryStack(stack, setup, 0);

        DeliveryOverride memory deliveryOverride = DeliveryOverride(
            stack.instruction.requestedReceiverValue,
            abi.encodePacked(uint8(4), stack.instruction.encodedExecutionInfo),
            stack.deliveryVaaHash //really redeliveryHash
        );

        // Note: Reverts when trying to abi.decode the ExecutionInfoVersion. No revert message
        vm.expectRevert();
        setup.target.coreRelayerFull.deliver{value: stack.budget}(
            stack.encodedVMs,
            stack.encodedDeliveryVAA,
            stack.relayerRefundAddress,
            WormholeRelayerSerde.encode(deliveryOverride)
        );
    }

    function testRevertSendMsgValueTooLow(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        (LocalNative deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(0), Gas.wrap(gasParams.targetGasLimit)
        );

        vm.expectRevert(
            abi.encodeWithSelector(
                InvalidMsgValue.selector, deliveryCost.unwrap() - 1, deliveryCost.unwrap()
            )
        );
        setup.source.integration.sendMessage{value: deliveryCost.unwrap() - 1}(
            message, setup.targetChain, gasParams.targetGasLimit, 0
        );
    }

    function testRevertSendMsgValueTooHigh(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        (LocalNative deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(0), Gas.wrap(gasParams.targetGasLimit)
        );

        vm.expectRevert(
            abi.encodeWithSelector(
                InvalidMsgValue.selector, deliveryCost.unwrap() + 1, deliveryCost.unwrap()
            )
        );
        setup.source.integration.sendMessage{value: deliveryCost.unwrap() + 1}(
            message, setup.targetChain, gasParams.targetGasLimit, 0
        );
    }

    function testRevertSendProviderNotSupported(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        (LocalNative deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(0), Gas.wrap(gasParams.targetGasLimit)
        );

        vm.expectRevert(
            abi.encodeWithSelector(
                DeliveryProviderDoesNotSupportTargetChain.selector,
                address(setup.source.deliveryProvider),
                uint16(32)
            )
        );
        setup.source.integration.sendMessage{value: deliveryCost.unwrap() - 1}(
            message, 32, gasParams.targetGasLimit, 0
        );
    }

    function testRevertResendProviderNotSupported(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        vm.expectRevert(
            abi.encodeWithSelector(
                DeliveryProviderDoesNotSupportTargetChain.selector,
                address(setup.source.deliveryProvider),
                uint16(32)
            )
        );
        setup.source.integration.resend{value: 0}(
            setup.sourceChain, 1, 32, uint32(REASONABLE_GAS_LIMIT.unwrap()), 0
        );
    }

    function testSendCheckConsistencyLevel(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        uint8 consistencyLevel
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        (LocalNative deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(0), Gas.wrap(0)
        );

        setup.source.coreRelayer.sendToEvm{value: deliveryCost.unwrap()}(
            setup.targetChain,
            address(0x0),
            bytes(""),
            TargetNative.wrap(0),
            LocalNative.wrap(0),
            Gas.wrap(0),
            setup.sourceChain,
            address(0x0),
            address(setup.source.deliveryProvider),
            vaaKeyArray(setup.sourceChain, 22345, address(this)),
            consistencyLevel
        );

        Vm.Log memory log = vm.getRecordedLogs()[0];

        // Parse the consistency level from the published VAA
        (uint8 actualConsistencyLevel,) = log.data.asUint8(32 + 32 + 32 + 32 - 1);

        assertTrue(consistencyLevel == actualConsistencyLevel);
    }

    function testToAndFromWormholeFormat(address msg1) public {
        assertTrue(toWormholeFormat(msg1) == bytes32(uint256(uint160(msg1))));
        assertTrue(fromWormholeFormat(toWormholeFormat(msg1)) == msg1);
    }

    function testRevertDeliveryReentrantCall(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
        vm.recordLogs();

        (LocalNative deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, TargetNative.wrap(0), Gas.wrap(500_000)
        );

        DeliveryStack memory stack;

        setup.source.integration.sendMessageWithReentrantDelivery{value: deliveryCost.unwrap()}(
            setup.targetChain, 500_000, 0
        );

        prepareDeliveryStack(stack, setup, 0);

        vm.deal(payable(address(setup.target.integration)), 1e30);
        vm.recordLogs();
        setup.target.integration.deliverReentrant{value: 1e28}(stack.encodedDeliveryVAA);
        Vm.Log[] memory logs = vm.getRecordedLogs();
        assertTrue(
            getDeliveryStatus(logs[logs.length - 1])
                == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE,
            "Outer delivery should have failed because inner call reverts"
        );
    }

    function testEncodeAndDecodeVaaKey() public {
        VaaKey memory vaaKey = VaaKey({chainId: 1, emitterAddress: bytes32(""), sequence: 23});
        (VaaKey memory newVaaKey,) =
            WormholeRelayerSerde.decodeVaaKey(WormholeRelayerSerde.encodeVaaKey(vaaKey), 0);
        checkVaaKey(WormholeRelayerSerde.encodeVaaKey(newVaaKey), 0, vaaKey);
    }

    function testEncodeAndDecodeMessageKey() public {
        VaaKey memory vaaKey = VaaKey({chainId: 1, emitterAddress: bytes32(""), sequence: 23});
        MessageKey memory messageKey = MessageKey({
            keyType: VAA_KEY_TYPE,
            encodedKey: WormholeRelayerSerde.encodeVaaKey(vaaKey)
        });

        (MessageKey memory newMessageKey,) = WormholeRelayerSerde.decodeMessageKey(
            WormholeRelayerSerde.encodeMessageKey(messageKey), 0
        );
        checkMessageKey(WormholeRelayerSerde.encodeMessageKey(newMessageKey), 0, messageKey);
    }

    function testRevertEncodeAndDecodeTooLongMessageKeyArray() public {
        uint256 len = uint256(type(uint8).max) + 1;
        MessageKey[] memory messageKeys = new MessageKey[](len);
        for (uint256 i = 0; i < len; ++i) {
            messageKeys[i] =
                MessageKey({keyType: RANDOM_KEY_TYPE, encodedKey: RANDOM_KEY_TYPE_BODY});
        }
        vm.expectRevert(abi.encodeWithSignature("TooManyMessageKeys(uint256)", len));
        WormholeRelayerSerde.encodeMessageKeyArray(messageKeys);
    }

    function testEncodeAndDecodeMessageKeyArray(uint8 len, uint8 idx) public {
        vm.assume(idx < len || len == 0);
        MessageKey[] memory messageKeys = new MessageKey[](len);
        for (uint256 i = 0; i < len; ++i) {
            messageKeys[i] =
                MessageKey({keyType: RANDOM_KEY_TYPE, encodedKey: RANDOM_KEY_TYPE_BODY});
        }
        VaaKey memory vaaKey = VaaKey({chainId: 1, emitterAddress: bytes32(""), sequence: 23});
        if (len > 0) {
            messageKeys[idx] = MessageKey(VAA_KEY_TYPE, WormholeRelayerSerde.encodeVaaKey(vaaKey));
        }

        (MessageKey[] memory newMessageKeys,) = WormholeRelayerSerde.decodeMessageKeyArray(
            WormholeRelayerSerde.encodeMessageKeyArray(messageKeys), 0
        );
        for (uint256 i = 0; i < len; ++i) {
            checkMessageKey(
                WormholeRelayerSerde.encodeMessageKey(newMessageKeys[i]), 0, messageKeys[i]
            );
        }
    }

    function testEncodeAndDecodeMessageKeyDifferentType() public {
        MessageKey memory messageKey =
            MessageKey({keyType: VAA_KEY_TYPE, encodedKey: bytes("my USDC transfer")});

        (MessageKey memory newMessageKey,) = WormholeRelayerSerde.decodeMessageKey(
            WormholeRelayerSerde.encodeMessageKey(messageKey), 0
        );
        checkMessageKey(WormholeRelayerSerde.encodeMessageKey(newMessageKey), 0, messageKey);
    }

    function testEncodeAndDecodeDeliveryInstruction(bytes memory payload) public {
        MessageKey[] memory messageKeys = new MessageKey[](3);
        messageKeys[0] = MessageKey({
            keyType: VAA_KEY_TYPE,
            encodedKey: WormholeRelayerSerde.encodeVaaKey(
                VaaKey({chainId: 1, emitterAddress: bytes32(""), sequence: 23})
                )
        });
        messageKeys[1] = messageKeys[0];
        messageKeys[2] = messageKeys[0];

        DeliveryInstruction memory instruction = DeliveryInstruction({
            targetChain: 1,
            targetAddress: bytes32(""),
            payload: payload,
            requestedReceiverValue: TargetNative.wrap(456),
            extraReceiverValue: TargetNative.wrap(123),
            encodedExecutionInfo: bytes("abcdefghijklmnopqrstuvwxyz"),
            refundChain: 2,
            refundAddress: keccak256(bytes("refundAddress")),
            refundDeliveryProvider: keccak256(bytes("refundRelayProvider")),
            sourceDeliveryProvider: keccak256(bytes("sourceRelayProvider")),
            senderAddress: keccak256(bytes("senderAddress")),
            messageKeys: messageKeys
        });

        DeliveryInstruction memory newInstruction =
            WormholeRelayerSerde.decodeDeliveryInstruction(WormholeRelayerSerde.encode(instruction));

        checkInstructionEquality(WormholeRelayerSerde.encode(instruction), newInstruction);
        checkInstructionEquality(WormholeRelayerSerde.encode(newInstruction), instruction);
    }

    function testDeliveryData(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        sendMessageToTargetChain(setup, gasParams.targetGasLimit, 0, message);

        genericRelayer.relay(setup.sourceChain);

        bytes32 deliveryVaaHash = getDeliveryVAAHash(vm.getRecordedLogs());

        DeliveryData memory deliveryData = setup.target.integration.getDeliveryData();

        assertTrue(
            fromWormholeFormat(deliveryData.sourceAddress) == address(setup.source.integration),
            "Source address wrong"
        );
        assertTrue(deliveryData.sourceChain == setup.sourceChain, "Source chain id wrong");
        assertTrue(deliveryData.deliveryHash == deliveryVaaHash, "delivery vaa hash wrong");
        assertTrue(
            keccak256(setup.target.integration.getMessage()) == keccak256(message), "payload wrong"
        );
    }

    function testProviderRefundAddressZeros(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(
            gasParams,
            feeParams,
            1000000
        );
        vm.recordLogs();
        setup.source.deliveryProvider.updateTargetChainAddress(
            setup.targetChain,
            bytes32(0x0)
        );
        (LocalNative deliveryCost, ) = setup
            .source
            .coreRelayer
            .quoteEVMDeliveryPrice(
                setup.targetChain,
                TargetNative.wrap(0),
                Gas.wrap(gasParams.targetGasLimit)
            );
        setup.source.integration.sendMessageWithRefund{
            value: LocalNative.unwrap(deliveryCost)
        }(
            message,
            setup.targetChain,
            gasParams.targetGasLimit,
            0,
            setup.sourceChain,
            address(this)
        );
        genericRelayer.relay(setup.sourceChain);
    }

    function testSendTargetAddressZeros(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(
            gasParams,
            feeParams,
            1000000
        );
        vm.recordLogs();
        (LocalNative deliveryCost, ) = setup
            .source
            .coreRelayer
            .quoteEVMDeliveryPrice(
                setup.targetChain,
                TargetNative.wrap(25),
                Gas.wrap(gasParams.targetGasLimit)
            );
        setup.source.coreRelayer.sendPayloadToEvm{
            value: LocalNative.unwrap(deliveryCost)
        }(
            setup.targetChain,
            address(0x1234123412341234123412341234123412341234),
            message,
            TargetNative.wrap(25),
            Gas.wrap(gasParams.targetGasLimit)
        );
        genericRelayer.relay(setup.sourceChain);
    }
}
