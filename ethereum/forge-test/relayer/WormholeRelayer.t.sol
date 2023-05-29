// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {IRelayProvider} from "../../contracts/interfaces/relayer/IRelayProvider.sol";
import {RelayProvider} from "../../contracts/relayer/relayProvider/RelayProvider.sol";
import {RelayProviderSetup} from "../../contracts/relayer/relayProvider/RelayProviderSetup.sol";
import {RelayProviderImplementation} from
    "../../contracts/relayer/relayProvider/RelayProviderImplementation.sol";
import {RelayProviderProxy} from "../../contracts/relayer/relayProvider/RelayProviderProxy.sol";
import {RelayProviderMessages} from
    "../../contracts/relayer/relayProvider/RelayProviderMessages.sol";
import {RelayProviderStructs} from "../../contracts/relayer/relayProvider/RelayProviderStructs.sol";
import "../../contracts/interfaces/relayer/IWormholeRelayer.sol";
import {
    DeliveryInstruction,
    RedeliveryInstruction,
    DeliveryOverride
} from "../../contracts/libraries/relayer/RelayerInternalStructs.sol";
import {WormholeRelayer} from "../../contracts/relayer/coreRelayer/WormholeRelayer.sol";
import {MockGenericRelayer} from "./MockGenericRelayer.sol";
import {MockWormhole} from "./MockWormhole.sol";
import {IWormhole} from "../../contracts/interfaces/IWormhole.sol";
import {WormholeSimulator, FakeWormholeSimulator} from "./WormholeSimulator.sol";
import {IWormholeReceiver} from "../../contracts/interfaces/relayer/IWormholeReceiver.sol";
import {AttackForwardIntegration} from "./AttackForwardIntegration.sol";
import {
    MockRelayerIntegration,
    XAddress,
    DeliveryData
} from "../../contracts/mock/relayer/MockRelayerIntegration.sol";
import {ForwardTester} from "./ForwardTester.sol";
import {TestHelpers} from "./TestHelpers.sol";
import {WormholeRelayerSerde} from "../../contracts/relayer/coreRelayer/WormholeRelayerSerde.sol";
import {
    EvmExecutionInfoV1,
    ExecutionInfoVersion,
    decodeEvmExecutionInfoV1,
    encodeEvmExecutionInfoV1
} from "../../contracts/libraries/relayer/ExecutionParameters.sol";
import {toWormholeFormat, fromWormholeFormat} from "../../contracts/libraries/relayer/Utils.sol";
import {BytesParsing} from "../../contracts/libraries/relayer/BytesParsing.sol";
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

    uint32 REASONABLE_GAS_LIMIT = 500000;
    uint32 REASONABLE_GAS_LIMIT_FORWARDS = 1000000;
    uint32 TOO_LOW_GAS_LIMIT = 10000;

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
    ) public {
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
        standardAssume(gasParams_, feeParams_, minTargetGasLimit);
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
        vm.deal(address(s.target.integration), type(uint256).max / 2);
        vm.deal(address(s.source.integration), type(uint256).max / 2);

        // set relayProvider prices
        s.source.relayProvider.updatePrice(
            s.targetChain, gasParams.targetGasPrice, feeParams.targetNativePrice
        );
        s.source.relayProvider.updatePrice(
            s.sourceChain, gasParams.sourceGasPrice, feeParams.sourceNativePrice
        );
        s.target.relayProvider.updatePrice(
            s.targetChain, gasParams.targetGasPrice, feeParams.targetNativePrice
        );
        s.target.relayProvider.updatePrice(
            s.sourceChain, gasParams.sourceGasPrice, feeParams.sourceNativePrice
        );

        s.source.relayProvider.updateDeliverGasOverhead(s.targetChain, gasParams.evmGasOverhead);
        s.target.relayProvider.updateDeliverGasOverhead(s.sourceChain, gasParams.evmGasOverhead);

        s.source.wormholeSimulator.setMessageFee(feeParams.wormholeFeeOnSource.unwrap());
        s.target.wormholeSimulator.setMessageFee(feeParams.wormholeFeeOnTarget.unwrap());
    }

    struct Contracts {
        IWormhole wormhole;
        WormholeSimulator wormholeSimulator;
        RelayProvider relayProvider;
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
            mapEntry.relayProvider = helpers.setUpRelayProvider(i, address(mapEntry.wormhole));
            mapEntry.coreRelayer =
                helpers.setUpWormholeRelayer(mapEntry.wormhole, address(mapEntry.relayProvider));
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
                map[i].relayProvider.updateSupportedChain(j, true);
                map[i].relayProvider.updateAssetConversionBuffer(j, 500, 10000);
                map[i].relayProvider.updateTargetChainAddress(
                    j, bytes32(uint256(uint160(address(map[j].relayProvider))))
                );
                map[i].relayProvider.updateRewardAddress(map[i].rewardAddress);
                helpers.registerWormholeRelayerContract(
                    map[i].coreRelayerFull,
                    map[i].wormhole,
                    i,
                    j,
                    bytes32(uint256(uint160(address(map[j].coreRelayer))))
                );
                map[i].relayProvider.updateMaximumBudget(j, Wei.wrap(maxBudget));
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
        pure
        returns (IWormholeRelayerDelivery.DeliveryStatus status)
    {
        (uint256 parsed,) = log.data.asUint256(32);
        status = IWormholeRelayerDelivery.DeliveryStatus(parsed);
    }

    function getDeliveryStatus()
        internal
        returns (IWormholeRelayerDelivery.DeliveryStatus status)
    {
        Vm.Log[] memory logs = vm.getRecordedLogs();
        status = getDeliveryStatus(logs[logs.length - 1]);
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
        uint32 gasLimit,
        uint128 receiverValue,
        bytes memory message
    ) internal returns (uint64 sequence) {
        (uint256 deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, receiverValue, gasLimit
        );
        sequence = setup.source.integration.sendMessage{
            value: deliveryCost + setup.source.wormhole.messageFee()
        }(message, setup.targetChain, gasLimit, receiverValue);
    }

    function sendMessageToTargetChainExpectingForwardedResponse(
        StandardSetupTwoChains memory setup,
        uint32 gasLimit,
        uint128 receiverValue,
        bytes memory message,
        bytes memory forwardedMessage,
        bool forwardShouldSucceed
    ) internal returns (uint64 sequence) {
        (uint256 forwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(
            setup.sourceChain, 0, REASONABLE_GAS_LIMIT
        );
        uint256 neededReceiverValue = forwardDeliveryCost + setup.target.wormhole.messageFee();
        vm.assume(neededReceiverValue <= type(uint128).max);
        if (forwardShouldSucceed) {
            vm.assume(receiverValue >= neededReceiverValue);
        } else {
            vm.assume(receiverValue < neededReceiverValue);
        }

        (uint256 deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, uint128(receiverValue), gasLimit
        );

        sequence = setup.source.integration.sendMessageWithForwardedResponse{
            value: deliveryCost + setup.source.wormhole.messageFee()
        }(message, forwardedMessage, setup.targetChain, gasLimit, receiverValue);
    }

    function resendMessageToTargetChain(
        StandardSetupTwoChains memory setup,
        uint64 sequence,
        uint32 gasLimit,
        uint128 receiverValue,
        bytes memory message
    ) internal {
        (uint256 newDeliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, receiverValue, gasLimit
        );

        setup.source.integration.resend{value: newDeliveryCost + setup.source.wormhole.messageFee()}(
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

    function testForward(
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

        resendMessageToTargetChain(setup, sequence, REASONABLE_GAS_LIMIT, 0, message);

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.SUCCESS);
    }

    /**
     * More functionality tests
     */

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
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, gasParams.targetGasLimit);

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
        assertTrue(
            getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.FORWARD_REQUEST_FAILURE
        );
    }

    function testMultipleForwards(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message,
        bytes memory forwardedMessage
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        (uint256 firstForwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(
            setup.sourceChain, 0, REASONABLE_GAS_LIMIT
        );
        (uint256 secondForwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, 0, REASONABLE_GAS_LIMIT
        );

        uint256 receiverValue = firstForwardDeliveryCost + secondForwardDeliveryCost
            + 2 * setup.target.wormhole.messageFee();
        vm.assume(receiverValue <= type(uint128).max);

        (uint256 deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, uint128(receiverValue), REASONABLE_GAS_LIMIT_FORWARDS * 2
        );

        vm.recordLogs();

        setup.source.integration.sendMessageWithMultiForwardedResponse{
            value: deliveryCost + feeParams.wormholeFeeOnSource
        }(
            message,
            forwardedMessage,
            setup.targetChain,
            REASONABLE_GAS_LIMIT_FORWARDS * 2,
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
            resendMessageToTargetChain(setup, sequence, TOO_LOW_GAS_LIMIT * i, 0, message);
            genericRelayer.relay(setup.sourceChain);
            assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(message));
            assertTrue(
                getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE
            );
        }

        resendMessageToTargetChain(setup, sequence, REASONABLE_GAS_LIMIT, 0, message);

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
        uint32 minGasLimit
    ) public returns (StandardSetupTwoChains memory s, FundsCorrectTest memory test) {
        s = standardAssumeAndSetupTwoChains(gasParams_, feeParams_, minGasLimit);

        test.refundAddressBalance = s.target.refundAddress.balance;
        test.relayerBalance = s.target.relayer.balance;
        test.rewardAddressBalance = s.source.rewardAddress.balance;
        test.destinationBalance = address(s.target.integration).balance;
        test.sourceContractBalance = address(s.source.coreRelayer).balance;
        test.targetContractBalance = address(s.target.coreRelayer).balance;
        test.receiverValue = feeParams_.receiverValueTarget;
        (test.deliveryPrice, test.targetChainRefundPerGasUnused) = s
            .source
            .coreRelayer
            .quoteEVMDeliveryPrice(s.targetChain, test.receiverValue, gasParams_.targetGasLimit);
        vm.assume(test.targetChainRefundPerGasUnused > 0);
    }

    function testFundsCorrectForASend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 170000);

        setup.source.integration.sendMessageWithRefund{
            value: test.deliveryPrice + feeParams.wormholeFeeOnSource
        }(
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
        DeliveryData memory deliveryData = setup.target.integration.getDeliveryData();

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
            test.rewardAddressAmount == test.deliveryPrice, "Reward address was paid correctly"
        );
        // assertTrue(test.targetChainRefundPerGasUnused == deliveryData.targetChainRefundPerGasUnused, "Correct value of targetChainRefundPerGasUnused is reported to receiver in deliveryData");

        test.gasAmount = uint32(
            gasParams.targetGasLimit - test.refundAddressAmount / test.targetChainRefundPerGasUnused
        );
        console.log(test.gasAmount);
        assertTrue(
            test.gasAmount >= 160000,
            "Gas amount (calculated from refund address payment) lower than expected"
        );
        assertTrue(
            test.gasAmount <= 170000,
            "Gas amount (calculated from refund address payment) higher than expected"
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
        gasParams.targetGasLimit = TOO_LOW_GAS_LIMIT;
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 0);

        setup.source.integration.sendMessageWithRefund{
            value: test.deliveryPrice + feeParams.wormholeFeeOnSource
        }(
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
            test.rewardAddressAmount == test.deliveryPrice, "Reward address was paid correctly"
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

        setup.target.relayProvider.updateSupportedChain(1, false);

        setup.source.integration.sendMessageWithForwardedResponse{
            value: test.deliveryPrice + feeParams.wormholeFeeOnSource
        }(
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
            test.rewardAddressAmount == test.deliveryPrice, "Reward address was paid correctly"
        );
        test.gasAmount = uint32(
            gasParams.targetGasLimit
                - (test.refundAddressAmount - test.receiverValue) / test.targetChainRefundPerGasUnused
        );
        console.log(test.gasAmount);
        assertTrue(
            test.gasAmount >= 190000,
            "Gas amount (calculated from refund address payment) lower than expected"
        );
        assertTrue(
            test.gasAmount <= 210000,
            "Gas amount (calculated from refund address payment) higher than expected"
        );
        assertTrue(
            test.relayerPayment == test.destinationAmount + test.refundAddressAmount,
            "Relayer paid the correct amount"
        );
    }

    function testFundsCorrectForAForward(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, REASONABLE_GAS_LIMIT_FORWARDS);

        (uint256 forwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(
            setup.sourceChain, 0, REASONABLE_GAS_LIMIT
        );
        uint256 receiverValue = forwardDeliveryCost + setup.target.wormhole.messageFee();
        vm.assume(receiverValue <= type(uint128).max);
        vm.assume(feeParams.receiverValueTarget >= receiverValue);

        uint256 rewardAddressBalanceTarget = setup.target.rewardAddress.balance;

        setup.source.integration.sendMessageWithForwardedResponse{
            value: test.deliveryPrice + feeParams.wormholeFeeOnSource
        }(
            bytes("Hello!"),
            bytes("Forwarded Message!"),
            setup.targetChain,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.targetChain,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(bytes("Hello!")));
        DeliveryData memory deliveryData = setup.target.integration.getDeliveryData();

        genericRelayer.relay(setup.targetChain);

        assertTrue(
            keccak256(setup.source.integration.getMessage())
                == keccak256(bytes("Forwarded Message!"))
        );

        test.refundAddressAmount = setup.target.refundAddress.balance - test.refundAddressBalance;

        test.rewardAddressAmount = setup.source.rewardAddress.balance - test.rewardAddressBalance;

        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;

        test.destinationAmount = test.destinationBalance - address(setup.target.integration).balance;

        assertTrue(
            test.sourceContractBalance == address(setup.source.coreRelayer).balance,
            "Source contract has extra balance"
        );
        assertTrue(
            test.targetContractBalance == address(setup.target.coreRelayer).balance,
            "Target contract has extra balance"
        );
        assertTrue(test.refundAddressAmount == 0, "All refund amount was forwarded");
        assertTrue(test.destinationAmount == 0, "All receiver amount was sent to forward");
        assertTrue(
            test.rewardAddressAmount == test.deliveryPrice,
            "Source reward address was paid correctly"
        );

        uint256 refundIntermediate = (
            setup.target.rewardAddress.balance - rewardAddressBalanceTarget
        ) + feeParams.wormholeFeeOnTarget - test.receiverValue;

        // assertTrue(
        //     test.targetChainRefundPerGasUnused == deliveryData.targetChainRefundPerGasUnused,
        //     "Correct value of targetChainRefundPerGasUnused is reported to receiver in deliveryData"
        // );
        test.gasAmount = uint32(
            gasParams.targetGasLimit - refundIntermediate / test.targetChainRefundPerGasUnused
        );

        console.log(test.gasAmount);

        assertTrue(
            test.relayerPayment == test.receiverValue + refundIntermediate,
            "Relayer paid the correct amount"
        );
        assertTrue(
            test.gasAmount >= 500000,
            "Gas amount (calculated from refund address payment) lower than expected"
        );
        assertTrue(
            test.gasAmount <= 600000,
            "Gas amount (calculated from refund address payment) higher than expected"
        );
    }

    function testFundsCorrectForAForwardFailure(
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
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 600000);

        (uint256 forwardDeliveryCost,) =
            setup.target.coreRelayer.quoteEVMDeliveryPrice(setup.sourceChain, 0, TOO_LOW_GAS_LIMIT);
        uint256 receiverValue = forwardDeliveryCost + setup.target.wormhole.messageFee();
        vm.assume(receiverValue <= type(uint128).max);
        vm.assume(feeParams.receiverValueTarget < receiverValue);

        setup.source.integration.sendMessageWithForwardedResponse{
            value: test.deliveryPrice + feeParams.wormholeFeeOnSource
        }(
            bytes("Hello!"),
            bytes("Forwarded Message!"),
            setup.targetChain,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.targetChain,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(bytes("Hello!")));
        assertTrue(
            getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.FORWARD_REQUEST_FAILURE
        );

        test.refundAddressAmount = setup.target.refundAddress.balance - test.refundAddressBalance;

        test.rewardAddressAmount = setup.source.rewardAddress.balance - test.rewardAddressBalance;

        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;

        test.destinationAmount = test.destinationBalance - address(setup.target.integration).balance;

        assertTrue(
            test.sourceContractBalance == address(setup.source.coreRelayer).balance,
            "Source contract has extra balance"
        );
        assertTrue(
            test.targetContractBalance == address(setup.target.coreRelayer).balance,
            "Target contract has extra balance"
        );
        assertTrue(test.destinationAmount == 0, "No receiver value was sent to contract");
        assertTrue(
            test.rewardAddressAmount == test.deliveryPrice,
            "Source reward address was paid correctly"
        );

        test.gasAmount = uint32(
            gasParams.targetGasLimit
                - (test.refundAddressAmount - test.receiverValue) / test.targetChainRefundPerGasUnused
        );

        console.log(test.gasAmount);

        assertTrue(
            test.relayerPayment == test.refundAddressAmount, "Relayer paid the correct amount"
        );
        assertTrue(
            test.gasAmount >= 500000,
            "Gas amount (calculated from refund address payment) lower than expected"
        );
        assertTrue(
            test.gasAmount <= 600000,
            "Gas amount (calculated from refund address payment) higher than expected"
        );
    }

    function testFundsCorrectForAResend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 170000);

        (uint256 notEnoughDeliveryPrice,) =
            setup.source.coreRelayer.quoteEVMDeliveryPrice(setup.targetChain, 0, TOO_LOW_GAS_LIMIT);

        uint64 sequence = setup.source.integration.sendMessageWithRefund{
            value: notEnoughDeliveryPrice + feeParams.wormholeFeeOnSource
        }(
            bytes("Hello!"),
            setup.targetChain,
            TOO_LOW_GAS_LIMIT,
            0,
            setup.targetChain,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChain);
        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(bytes("Hello!")));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE);

        (, test) = setupFundsCorrectTest(gasParams, feeParams, 170000);

        //call a resend for the orignal message
        setup.source.integration.resend{
            value: test.deliveryPrice + setup.source.wormhole.messageFee()
        }(
            setup.sourceChain,
            sequence,
            setup.targetChain,
            gasParams.targetGasLimit,
            test.receiverValue
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(bytes("Hello!")));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.SUCCESS);
        DeliveryData memory deliveryData = setup.target.integration.getDeliveryData();

        test.refundAddressAmount = setup.target.refundAddress.balance - test.refundAddressBalance;
        test.rewardAddressAmount = setup.source.rewardAddress.balance - test.rewardAddressBalance;
        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;
        test.destinationAmount = address(setup.target.integration).balance - test.destinationBalance;

        assertTrue(test.sourceContractBalance == address(setup.source.coreRelayer).balance);
        assertTrue(test.targetContractBalance == address(setup.target.coreRelayer).balance);
        console.log(test.destinationAmount);
        console.log(feeParams.receiverValueTarget);
        assertTrue(
            test.destinationAmount == feeParams.receiverValueTarget,
            "Receiver value was sent to the contract"
        );
        assertTrue(
            test.rewardAddressAmount == test.deliveryPrice, "Reward address was paid correctly"
        );
        test.gasAmount = uint32(
            gasParams.targetGasLimit - test.refundAddressAmount / test.targetChainRefundPerGasUnused
        );
        // assertTrue(test.targetChainRefundPerGasUnused == deliveryData.targetChainRefundPerGasUnused);
        console.log(test.gasAmount);
        assertTrue(
            test.gasAmount >= 155000,
            "Gas amount (calculated from refund address payment) lower than expected"
        );
        assertTrue(
            test.gasAmount <= 170000,
            "Gas amount (calculated from refund address payment) higher than expected"
        );
        assertTrue(
            test.relayerPayment == test.destinationAmount + test.refundAddressAmount,
            "Relayer paid the correct amount"
        );
    }

    function testFundsCorrectForASendCrossChainRefund(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 170000);

        uint256 refundRewardAddressBalance = setup.target.rewardAddress.balance;
        uint256 refundAddressBalance = setup.source.refundAddress.balance;

        setup.source.integration.sendMessageWithRefund{
            value: test.deliveryPrice + feeParams.wormholeFeeOnSource
        }(
            bytes("Hello!"),
            setup.targetChain,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.sourceChain,
            setup.source.refundAddress
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(bytes("Hello!")));
        DeliveryData memory deliveryData = setup.target.integration.getDeliveryData();

        genericRelayer.relay(setup.targetChain);

        assertTrue(
            test.deliveryPrice == setup.source.rewardAddress.balance - test.rewardAddressBalance,
            "The source to target relayer's reward address was paid appropriately"
        );
        // Calculate maximum refund for source->target delivery, and check against Delivery Data
        // assertTrue(
        //     test.targetChainRefundPerGasUnused == deliveryData.targetChainRefundPerGasUnused,
        //     "Correct value of targetChainRefundPerGasUnused is reported to receiver in deliveryData"
        // );
        uint256 amountToGetInRefundTarget =
            (setup.target.rewardAddress.balance - refundRewardAddressBalance);

        uint256 refundSource = 0;
        (uint256 baseFee,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(setup.sourceChain, 0, 0);

        vm.assume(amountToGetInRefundTarget > baseFee);
        if (amountToGetInRefundTarget > baseFee) {
            refundSource = setup.target.coreRelayer.quoteNativeForChain(
                setup.sourceChain,
                uint128(amountToGetInRefundTarget - baseFee),
                setup.target.coreRelayer.getDefaultRelayProvider()
            );
        }

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
        assertTrue(
            test.gasAmount >= 160000,
            "Gas amount (calculated from refund address payment) lower than expected"
        );
        assertTrue(
            test.gasAmount <= 170000,
            "Gas amount (calculated from refund address payment) higher than expected"
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

    function testUnitTestSend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        UnitTestParams memory params
    ) public {
        gasParams.targetGasLimit = params.gasLimit;
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, params.gasLimit);
        VaaKey[] memory vaaKeys = new VaaKey[](3);
        for (uint256 j = 0; j < 3; j++) {
            vaaKeys[j] = params.vaaKeysFixed[j];
        }
        vm.recordLogs();

        (uint256 deliveryCost, uint256 targetChainRefundPerGasUnused) = setup
            .source
            .coreRelayer
            .quoteEVMDeliveryPrice(setup.targetChain, params.receiverValue, params.gasLimit);
        uint256 value =
            deliveryCost + feeParams.wormholeFeeOnSource + params.paymentForExtraReceiverValue;
        setup.source.integration.sendToEvm{value: value}(
            setup.targetChain,
            params.targetAddress,
            params.gasLimit,
            params.refundChain,
            params.refundAddress,
            params.receiverValue,
            params.paymentForExtraReceiverValue,
            params.payload,
            vaaKeys
        );

        bytes memory encodedExecutionInfo = abi.encode(
            uint8(ExecutionInfoVersion.EVM_V1), params.gasLimit, targetChainRefundPerGasUnused
        );
        Wei extraReceiverValue = Wei.wrap(
            setup.source.coreRelayer.quoteNativeForChain(
                setup.targetChain,
                params.paymentForExtraReceiverValue,
                address(setup.source.relayProvider)
            )
        );

        DeliveryInstruction memory expectedInstruction = DeliveryInstruction({
            targetChain: setup.targetChain,
            targetAddress: toWormholeFormat(params.targetAddress),
            payload: params.payload,
            requestedReceiverValue: Wei.wrap(params.receiverValue),
            extraReceiverValue: extraReceiverValue,
            encodedExecutionInfo: encodedExecutionInfo,
            refundChain: params.refundChain,
            refundAddress: toWormholeFormat(params.refundAddress),
            refundRelayProvider: setup.source.relayProvider.getTargetChainAddress(setup.targetChain),
            sourceRelayProvider: toWormholeFormat(address(setup.source.relayProvider)),
            senderAddress: toWormholeFormat(address(setup.source.integration)),
            vaaKeys: vaaKeys
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

        (uint256 deliveryCost, uint256 targetChainRefundPerGasUnused) = setup
            .source
            .coreRelayer
            .quoteEVMDeliveryPrice(setup.targetChain, params.newReceiverValue, params.newGasLimit);
        uint256 value = deliveryCost + feeParams.wormholeFeeOnSource;
        vm.deal(params.senderAddress, value);
        vm.prank(params.senderAddress);
        setup.source.coreRelayer.resendToEvm{value: value}(
            params.deliveryVaaKey,
            setup.targetChain,
            Wei.wrap(params.newReceiverValue),
            Gas.wrap(params.newGasLimit),
            address(setup.source.relayProvider)
        );

        bytes memory encodedExecutionInfo = abi.encode(
            uint8(ExecutionInfoVersion.EVM_V1), params.newGasLimit, targetChainRefundPerGasUnused
        );

        RedeliveryInstruction memory expectedInstruction = RedeliveryInstruction({
            deliveryVaaKey: params.deliveryVaaKey,
            targetChain: setup.targetChain,
            newRequestedReceiverValue: Wei.wrap(params.newReceiverValue),
            newEncodedExecutionInfo: encodedExecutionInfo,
            newSourceRelayProvider: toWormholeFormat(address(setup.source.relayProvider)),
            newSenderAddress: toWormholeFormat(params.senderAddress)
        });

        checkRedeliveryInstructionEquality(
            relayerWormholeSimulator.parseVMFromLogs(vm.getRecordedLogs()[0]).payload,
            expectedInstruction
        );
    }

    function checkVaaKey(
        bytes memory data,
        uint256 _index,
        VaaKey memory vaaKey
    ) public returns (uint256 index) {
        VaaKey memory decodedVaaKey;
        uint8 payloadId;
        index = _index;
        (payloadId, index) = data.asUint8(index);
        assertTrue(payloadId == 1, "Is a vaa key version 1");
        (decodedVaaKey.chainId, index) = data.asUint16(index);
        assertTrue(decodedVaaKey.chainId == vaaKey.chainId, "Wrong chain id");
        (decodedVaaKey.emitterAddress, index) = data.asBytes32(index);
        assertTrue(decodedVaaKey.emitterAddress == vaaKey.emitterAddress, "Wrong emitter address");
        (decodedVaaKey.sequence, index) = data.asUint64(index);
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
        (decodedInstruction.refundRelayProvider, index) = data.asBytes32(index);
        assertTrue(
            decodedInstruction.refundRelayProvider == expectedInstruction.refundRelayProvider,
            "Wrong refund relay provider"
        );
        (decodedInstruction.sourceRelayProvider, index) = data.asBytes32(index);
        assertTrue(
            decodedInstruction.sourceRelayProvider == expectedInstruction.sourceRelayProvider,
            "Wrong source relay provider"
        );
        (decodedInstruction.senderAddress, index) = data.asBytes32(index);
        assertTrue(
            decodedInstruction.senderAddress == expectedInstruction.senderAddress,
            "Wrong sender address"
        );
        uint8 vaaKeysLength;
        (vaaKeysLength, index) = data.asUint8(index);
        decodedInstruction.vaaKeys = new VaaKey[](vaaKeysLength);
        for (uint256 i = 0; i < vaaKeysLength; i++) {
            index = checkVaaKey(data, index, expectedInstruction.vaaKeys[i]);
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
        (decodedInstruction.newSourceRelayProvider, index) = data.asBytes32(index);
        assertTrue(
            decodedInstruction.newSourceRelayProvider == expectedInstruction.newSourceRelayProvider,
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
        EvmExecutionInfoV1 memory executionInfo =
            decodeEvmExecutionInfoV1(stack.instruction.encodedExecutionInfo);
        stack.budget = Wei.unwrap(
            executionInfo.gasLimit.toWei(executionInfo.targetChainRefundPerGasUnused)
                + stack.instruction.extraReceiverValue
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
            abi.encodeWithSignature("VaaKeysLengthDoesNotMatchVaasLength(uint256,uint256)", 0, 1)
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

        (stack.payment,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, 0, gasParams.targetGasLimit
        );
        stack.payment += setup.source.wormhole.messageFee();

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
            stack.instruction.requestedReceiverValue - Wei.wrap(1),
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

    function testRevertDeliveryWithOverrideMaximumRefund(
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
                    gasLimit: executionInfo.gasLimit,
                    targetChainRefundPerGasUnused: GasPrice.wrap(
                        executionInfo.targetChainRefundPerGasUnused.unwrap() - 1
                        )
                })
            ),
            stack.deliveryVaaHash //really redeliveryHash
        );

        vm.expectRevert(abi.encodeWithSignature("InvalidOverrideRefundPerGasUnused()"));
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

        (uint256 deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, 0, gasParams.targetGasLimit
        );

        vm.expectRevert(
            abi.encodeWithSelector(
                InvalidMsgValue.selector,
                deliveryCost + feeParams.wormholeFeeOnSource - 1,
                deliveryCost + feeParams.wormholeFeeOnSource
            )
        );
        setup.source.integration.sendMessage{
            value: deliveryCost + feeParams.wormholeFeeOnSource - 1
        }(message, setup.targetChain, gasParams.targetGasLimit, 0);
    }

    function testRevertSendMsgValueTooHigh(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        (uint256 deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, 0, gasParams.targetGasLimit
        );

        vm.expectRevert(
            abi.encodeWithSelector(
                InvalidMsgValue.selector,
                deliveryCost + feeParams.wormholeFeeOnSource + 1,
                deliveryCost + feeParams.wormholeFeeOnSource
            )
        );
        setup.source.integration.sendMessage{
            value: deliveryCost + feeParams.wormholeFeeOnSource + 1
        }(message, setup.targetChain, gasParams.targetGasLimit, 0);
    }

    function testRevertSendProviderNotSupported(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        (uint256 deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, 0, gasParams.targetGasLimit
        );

        vm.expectRevert(
            abi.encodeWithSelector(
                RelayProviderDoesNotSupportTargetChain.selector,
                address(setup.source.relayProvider),
                uint16(32)
            )
        );
        setup.source.integration.sendMessage{
            value: deliveryCost + feeParams.wormholeFeeOnSource - 1
        }(message, 32, gasParams.targetGasLimit, 0);
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
                RelayProviderDoesNotSupportTargetChain.selector,
                address(setup.source.relayProvider),
                uint16(32)
            )
        );
        setup.source.integration.resend{value: 0}(setup.sourceChain, 1, 32, REASONABLE_GAS_LIMIT, 0);
    }

    function testSendCheckConsistencyLevel(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        uint8 consistencyLevel
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        (uint256 deliveryCost,) =
            setup.source.coreRelayer.quoteEVMDeliveryPrice(setup.targetChain, 0, 0);

        setup.source.coreRelayer.sendToEvm{value: deliveryCost + feeParams.wormholeFeeOnSource}(
            setup.targetChain,
            address(0x0),
            bytes(""),
            Wei.wrap(0),
            Wei.wrap(0),
            Gas.wrap(0),
            setup.sourceChain,
            address(0x0),
            address(setup.source.relayProvider),
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

    /**
     * Forward Revert Tests using Forward Tester
     */

    ForwardTester forwardTester;

    function executeForwardTest(
        ForwardTester.Action test,
        IWormholeRelayerDelivery.DeliveryStatus desiredOutcome,
        StandardSetupTwoChains memory setup,
        FeeParameters memory feeParams
    ) internal {
        vm.recordLogs();
        forwardTester =
        new ForwardTester(address(setup.target.wormhole), address(setup.target.coreRelayer), address(setup.target.wormholeSimulator));
        vm.deal(address(forwardTester), type(uint256).max / 2);

        (uint256 forwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(
            setup.sourceChain, 0, REASONABLE_GAS_LIMIT
        );
        uint256 receiverValue = forwardDeliveryCost + setup.target.wormhole.messageFee();
        vm.assume(receiverValue <= type(uint128).max);

        (uint256 deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(
            setup.targetChain, uint128(receiverValue), REASONABLE_GAS_LIMIT_FORWARDS
        );

        setup.source.coreRelayer.sendPayloadToEvm{
            value: deliveryCost + feeParams.wormholeFeeOnSource
        }(
            setup.targetChain,
            address(forwardTester),
            abi.encodePacked(uint8(test)),
            Wei.wrap(receiverValue),
            Gas.wrap(REASONABLE_GAS_LIMIT_FORWARDS)
        );
        genericRelayer.relay(setup.sourceChain);
        IWormholeRelayerDelivery.DeliveryStatus status = getDeliveryStatus();
        assertTrue(status == desiredOutcome);
    }

    function testForwardTester(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
        executeForwardTest(
            ForwardTester.Action.WorksCorrectly,
            IWormholeRelayerDelivery.DeliveryStatus.FORWARD_REQUEST_SUCCESS,
            setup,
            feeParams
        );
    }

    function testRevertForwardNoDeliveryInProgress(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.expectRevert(abi.encodeWithSignature("NoDeliveryInProgress()"));
        setup.source.coreRelayer.forwardPayloadToEvm(
            setup.targetChain,
            address(forwardTester),
            bytes(""),
            Wei.wrap(0),
            Gas.wrap(TOO_LOW_GAS_LIMIT)
        );
    }

    function testRevertForwardForwardRequestFromWrongAddress(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        executeForwardTest(
            ForwardTester.Action.ForwardRequestFromWrongAddress,
            IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE,
            setup,
            feeParams
        );
    }

    function testRevertDeliveryReentrantCall(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
        executeForwardTest(
            ForwardTester.Action.ReentrantCall,
            IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE,
            setup,
            feeParams
        );
    }

    function testRevertForwardProviderNotSupported(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        executeForwardTest(
            ForwardTester.Action.ProviderNotSupported,
            IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE,
            setup,
            feeParams
        );
    }

    /*
    

    
    


    

    function testFundsCorrectForASendRevertsCrossChainRefund(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        (Contracts memory source, Contracts memory target, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 400000);

        vm.recordLogs();

        // Transaction fee not enough
        test.transactionFee = source.coreRelayer.quoteGas(2, 150000, address(source.relayProvider));

        // Receiver value (which will be refunded -> used to pay for cross chain refund) enough to pay for  wormhole fee on target, and target->source (1 gas)
        test.receiverValueSource += (
            (
                target.coreRelayer.quoteGas(1, 1, address(target.relayProvider))
                    + target.wormhole.messageFee()
            ) * 105 / 100 + 1
        ) * feeParams.targetNativePrice / feeParams.sourceNativePrice + 1;

        // Calculate how much receiver value was requested
        uint256 receiverValueTargetActual = (
            test.receiverValueSource * feeParams.sourceNativePrice * 100
                / (uint256(1) * feeParams.targetNativePrice * 105)
        );

        uint256 actualGasLimit = (
            test.transactionFee - source.relayProvider.quoteDeliveryOverhead(2).unwrap()
        ) / source.relayProvider.quoteGasPrice(2).unwrap();
        vm.assume(actualGasLimit <= type(uint32).max);
        if (actualGasLimit > type(uint32).max) {
            actualGasLimit = type(uint32).max;
        }
        test.payment = test.transactionFee + uint256(3) * source.wormhole.messageFee()
            + test.receiverValueSource;

        uint256 refundRewardAddressBalance = target.rewardAddress.balance;
        uint256 refundAddressBalance = source.refundAddress.balance;

        source.integration.sendMessageGeneral{value: test.payment}(
            bytes("Hello!"),
            2,
            address(target.integration),
            1,
            address(source.refundAddress),
            test.receiverValueSource,
            bytes("")
        );

        genericRelayer.relay(1);

        genericRelayer.relay(2);

        assertTrue(keccak256(target.integration.getMessage()) != keccak256(bytes("Hello!")));

        assertTrue(
            test.transactionFee + test.receiverValueSource
                == source.rewardAddress.balance - test.rewardAddressBalance,
            "The source to target relayer's reward address was paid appropriately"
        );
        uint256 amountToGetInRefundTarget =
            (target.rewardAddress.balance - refundRewardAddressBalance);

        // Calculate maximum refund for source->target delivery
        uint256 maximumRefund = (test.transactionFee - test.overhead) * feeParams.sourceNativePrice
            * 100 / (uint256(1) * feeParams.targetNativePrice * 105);

        // Calculate amount that must have been spent on gas, by reverse engineering from the amount that was paid to the provider's reward address on the target chain
        test.gasAmount = uint32(
            actualGasLimit
                - (
                    (
                        amountToGetInRefundTarget + feeParams.wormholeFeeOnTarget
                            - receiverValueTargetActual
                    )
                ) * actualGasLimit / maximumRefund
        );
        uint256 refundSource = 0;
        if (amountToGetInRefundTarget > target.relayProvider.quoteDeliveryOverhead(1).unwrap()) {
            refundSource = (
                amountToGetInRefundTarget - target.relayProvider.quoteDeliveryOverhead(1).unwrap()
            ) * feeParams.targetNativePrice * 100 / (uint256(1) * feeParams.sourceNativePrice * 105);
        }

        assertTrue(
            refundSource == source.refundAddress.balance - refundAddressBalance,
            "Refund wasn't the correct amount"
        );
        assertTrue(test.gasAmount == actualGasLimit, "Gas amount is as expected");
    }

    function testXNoFundsLostForASendIfReceiveWormholeMessagesReverts(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        uint256 refundAddressBalance = setup.target.refundAddress.balance;
        uint256 relayerBalance = setup.target.relayer.balance;
        uint256 rewardAddressBalance = setup.source.rewardAddress.balance;
        uint256 receiverValueSource = setup.source.coreRelayer.quoteReceiverValue(
            setup.targetChain, feeParams.receiverValueTarget, address(setup.source.relayProvider)
        );

        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChain, 21000, address(setup.source.relayProvider)
        ) + uint256(3) * setup.source.wormhole.messageFee() + receiverValueSource;

        setup.source.integration.sendMessageGeneral{value: payment}(
            message,
            setup.targetChain,
            address(setup.target.integration),
            setup.targetChain,
            address(setup.target.refundAddress),
            receiverValueSource,
            bytes("")
        );

        genericRelayer.relay(setup.sourceChain);

        uint256 USDcost = uint256(
            payment - uint256(3) * map[setup.sourceChain].wormhole.messageFee()
        ) * feeParams.sourceNativePrice
            - (setup.target.refundAddress.balance - refundAddressBalance) * feeParams.targetNativePrice;
        uint256 relayerProfit = uint256(feeParams.sourceNativePrice)
            * (setup.source.rewardAddress.balance - rewardAddressBalance)
            - feeParams.targetNativePrice * (relayerBalance - setup.target.relayer.balance);
        assertTrue(USDcost == relayerProfit, "We paid the exact amount");
    }

    function assumeAndGetForwardPayment(
        uint32 gasFirst,
        uint32 gasSecond,
        StandardSetupTwoChains memory setup,
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) internal returns (uint256) {
        vm.assume(uint256(1) * gasParams.targetGasPrice > uint256(1) * gasParams.sourceGasPrice);

        vm.assume(
            uint256(1) * feeParams.targetNativePrice > uint256(1) * feeParams.sourceNativePrice
        );

        vm.assume(
            setup.source.coreRelayer.quoteGas(
                setup.targetChain, gasFirst, address(setup.source.relayProvider)
            ) < uint256(2) ** 221
        );
        vm.assume(
            setup.target.coreRelayer.quoteGas(
                setup.sourceChain, gasSecond, address(setup.target.relayProvider)
            ) < uint256(2) ** 221 / feeParams.targetNativePrice
        );

        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChain, gasFirst, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        uint256 payment2 = (
            setup.target.coreRelayer.quoteGas(
                setup.sourceChain, gasSecond, address(setup.target.relayProvider)
            ) + uint256(2) * setup.target.wormhole.messageFee()
        ) * feeParams.targetNativePrice / feeParams.sourceNativePrice + 1;

        vm.assume((payment + payment2 * 105 / 100 + 1) < (uint256(2) ** 222));

        return (payment + payment2 * 105 / 100 + 1);
    }

    struct ForwardRequestFailStack {
        uint256 payment;
        uint256 wormholeFee;
        uint16[] chains;
        uint32[] gasLimits;
        bytes[] newMessages;
        uint64 sequence1;
        uint64 sequence2;
        bytes32 targetAddress;
        bytes encodedFurtherInstructions;
        bytes payload;
    }

    function testForwardRequestFail(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
        ForwardRequestFailStack memory stack;
        vm.assume(
            uint256(1) * feeParams.targetNativePrice * gasParams.targetGasPrice * 10
                < uint256(1) * feeParams.sourceNativePrice * gasParams.sourceGasPrice
        );
        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChain, 1000000, address(setup.source.relayProvider)
        );

        vm.recordLogs();

        IWormhole wormhole = setup.source.wormhole;
        stack.wormholeFee = wormhole.messageFee();
        vm.prank(address(setup.source.integration));
        stack.sequence1 = wormhole.publishMessage{value: stack.wormholeFee}(0, bytes("hello!"), 200);

        stack.chains = new uint16[](1);
        stack.chains[0] = wormhole.chainId();
        stack.gasLimits = new uint32[](1);
        stack.gasLimits[0] = 1000000;
        stack.newMessages = new bytes[](2);
        stack.newMessages[0] = bytes("received!");
        stack.newMessages[1] = abi.encodePacked(uint8(0));
        stack.payload = bytes("");
        MockRelayerIntegration.FurtherInstructions memory instructions = MockRelayerIntegration
            .FurtherInstructions({
            keepSending: true,
            newMessages: stack.newMessages,
            chains: stack.chains,
            gasLimits: stack.gasLimits
        });
        stack.encodedFurtherInstructions =
            setup.source.integration.encodeFurtherInstructions(instructions);
        vm.prank(address(setup.source.integration));
        stack.sequence2 = wormhole.publishMessage{value: stack.wormholeFee}(
            0, stack.encodedFurtherInstructions, 200
        );
        stack.targetAddress = toWormholeFormat(address(setup.target.integration));

        sendHelper(setup, stack);

        genericRelayer.relay(setup.sourceChain);

        Vm.Log[] memory logs = vm.getRecordedLogs();

        IWormholeRelayerDelivery.DeliveryStatus status = getDeliveryStatus(logs[logs.length - 1]);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(bytes("hello!")));

        assertTrue(status == IWormholeRelayerDelivery.DeliveryStatus.FORWARD_REQUEST_FAILURE);
    }

    function sendHelper(
        StandardSetupTwoChains memory setup,
        ForwardRequestFailStack memory stack
    ) public {
        VaaKey[] memory vaaKeys = vaaKeyArray(
            setup.sourceChain,
            stack.sequence1,
            address(setup.source.integration),
            stack.sequence2,
            address(setup.source.integration)
        );
        setup.source.coreRelayer.send{value: stack.payment + stack.wormholeFee}(
            setup.targetChain,
            stack.targetAddress,
            setup.targetChain,
            stack.targetAddress,
            stack.payment,
            0,
            stack.payload,
            vaaKeys,
            200
        );
    }

    function testAttackForwardRequestCache(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        // General idea:
        // 1. Attacker sets up a malicious integration contract in the target chain.
        // 2. Attacker requests a message send to `target` chain.
        //   The message destination and the refund address are both the malicious integration contract in the target chain.
        // 3. The delivery of the message triggers a refund to the malicious integration contract.
        // 4. During the refund, the integration contract activates the forwarding mechanism.
        //   This is allowed due to the integration contract also being the target of the delivery.
        // 5. The forward request is left as is in the `WormholeRelayer` state.
        // 6. The next message (i.e. the victim's message) delivery on `target` chain, from any relayer, using any `RelayProvider` and any integration contract,
        //   will see the forward request placed by the malicious integration contract and act on it.
        // Caveat: the delivery of the victim's message must not invoke the forwarding mechanism for the attack test to be meaningful.
        //
        // In essence, this tries to attack the shared forwarding request cache present in the contract state.
        // This attack doesn't work thanks to the check inside the `requestForward` function that only allows requesting a forward when there is a delivery being processed.

        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        // Collected funds from the attack are meant to be sent here.
        address attackerSourceAddress = address(
            uint160(
                uint256(keccak256(abi.encodePacked(bytes("attackerAddress"), setup.sourceChain)))
            )
        );
        assertTrue(attackerSourceAddress.balance == 0);

        // Borrowed assumes from testForward. They should help since this test is similar.
        vm.assume(
            uint256(1) * gasParams.targetGasPrice * feeParams.targetNativePrice
                > uint256(1) * gasParams.sourceGasPrice * feeParams.sourceNativePrice
        );

        vm.assume(
            setup.source.coreRelayer.quoteGas(
                setup.targetChain, gasParams.targetGasLimit, address(setup.source.relayProvider)
            ) < uint256(2) ** 222
        );
        vm.assume(
            setup.target.coreRelayer.quoteGas(
                setup.sourceChain, 500000, address(setup.target.relayProvider)
            ) < uint256(2) ** 222 / feeParams.targetNativePrice
        );

        // Estimate the cost based on the initialized values
        uint256 computeBudget = setup.source.coreRelayer.quoteGas(
            setup.targetChain, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        {
            AttackForwardIntegration attackerContract =
            new AttackForwardIntegration(setup.target.wormhole, setup.target.coreRelayer, setup.targetChain, attackerSourceAddress);
            bytes memory attackMsg = "attack";

            vm.recordLogs();

            // The attacker requests the message to be sent to the malicious contract.
            // It is critical that the refund and destination (aka integrator) addresses are the same.
            setup.source.integration.sendMessage{
                value: computeBudget + uint256(3) * setup.source.wormhole.messageFee()
            }(attackMsg, setup.targetChain, address(attackerContract));

            // The relayer triggers the call to the malicious contract.
            genericRelayer.relay(setup.sourceChain);

            // The message delivery should fail
            assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(attackMsg));
        }

        {
            // Now one victim sends their message. It doesn't need to be from the same source chain.
            // What's necessary is that a message is delivered to the chain targeted by the attacker.
            bytes memory victimMsg = "relay my message";

            uint256 victimBalancePreDelivery = setup.target.refundAddress.balance;

            // We will reutilize the compute budget estimated for the attacker to simplify the code here.
            // The victim requests their message to be sent.
            setup.source.integration.sendMessageWithRefundAddress{
                value: computeBudget + uint256(3) * setup.source.wormhole.messageFee()
            }(
                victimMsg,
                setup.targetChain,
                address(setup.target.integration),
                address(setup.target.refundAddress),
                bytes("")
            );

            // The relayer delivers the victim's message.
            // During the delivery process, the forward request injected by the malicious contract is acknowledged.
            // The victim's refund address is not called due to this.
            genericRelayer.relay(setup.sourceChain);

            // Ensures the message was received.
            assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(victimMsg));
            // Here we assert that the victim's refund is safe.
            assertTrue(victimBalancePreDelivery < setup.target.refundAddress.balance);
        }

        genericRelayer.relay(setup.targetChain);

        // Assert that the attack wasn't successful.
        assertTrue(attackerSourceAddress.balance == 0);
    }

    function testQuoteReceiverValueIsEnough(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
        vm.assume(feeParams.receiverValueTarget > 0);
        vm.recordLogs();

        // estimate the cost based on the intialized values
        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChain, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        uint256 oldBalance = address(setup.target.integration).balance;

        uint256 newReceiverValueSource = setup.source.coreRelayer.quoteReceiverValue(
            setup.targetChain, feeParams.receiverValueTarget, address(setup.source.relayProvider)
        );

        vm.deal(address(this), payment + newReceiverValueSource);

        setup.source.integration.sendMessageGeneral{value: payment + newReceiverValueSource}(
            message,
            setup.targetChain,
            address(setup.target.integration),
            setup.targetChain,
            address(0x0),
            newReceiverValueSource,
            bytes("")
        );

        genericRelayer.relay(setup.sourceChain);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        assertTrue(
            address(setup.target.integration).balance >= oldBalance + feeParams.receiverValueTarget
        );
    }

    

    

    

    function testEncodeAndDecodeDeliveryInstruction(
        ExecutionParameters memory executionParameters,
        bytes memory payload
    ) public {
        VaaKey[] memory vaaKeys = new VaaKey[](3);
        vaaKeys[0] = VaaKey({
            infoType: VaaKeyType.EMITTER_SEQUENCE,
            chainId: 1,
            emitterAddress: bytes32(""),
            sequence: 23,
            vaaHash: bytes32("")
        });
        vaaKeys[1] = vaaKeys[0];
        vaaKeys[2] = vaaKeys[0];

        DeliveryInstruction memory instruction = DeliveryInstruction({
            targetChain: 1,
            targetAddress: bytes32(""),
            refundAddress: bytes32(""),
            refundChain: 2,
            maximumRefundTarget: Wei.wrap(123),
            receiverValueTarget: Wei.wrap(456),
            sourceRelayProvider: bytes32(""),
            targetRelayProvider: bytes32(""),
            senderAddress: bytes32(""),
            vaaKeys: vaaKeys,
            consistencyLevel: 200,
            executionParameters: executionParameters,
            payload: payload
        });

        DeliveryInstruction memory newInstruction =
            WormholeRelayerSerde.decodeDeliveryInstruction(WormholeRelayerSerde.encode(instruction));

        assertTrue(newInstruction.maximumRefundTarget == instruction.maximumRefundTarget);
        assertTrue(newInstruction.receiverValueTarget == instruction.receiverValueTarget);

        assertTrue(keccak256(newInstruction.payload) == keccak256(instruction.payload));
    }

    function testDeliveryData(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChain, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + setup.source.wormhole.messageFee()*3;

        uint256 maxTransactionFee = payment - 3*setup.source.wormhole.messageFee();

        bytes memory payload = abi.encodePacked(uint256(6));

        setup.source.integration.sendMessageWithPayload{value: payment}(
            bytes(""), setup.targetChain, address(setup.target.integration), payload
        );

        genericRelayer.relay(setup.sourceChain);

        bytes32 deliveryVaaHash = getDeliveryVAAHash(vm.getRecordedLogs());

        DeliveryData memory deliveryData = setup.target.integration.getDeliveryData();

        uint256 calculatedRefund = 0;
        if (
            maxTransactionFee
                > setup.source.relayProvider.quoteDeliveryOverhead(setup.targetChain).unwrap()
        ) {
            calculatedRefund = (
                maxTransactionFee
                    - setup.source.relayProvider.quoteDeliveryOverhead(setup.targetChain).unwrap()
            ) * feeParams.sourceNativePrice * 100 / (uint256(feeParams.targetNativePrice) * 105);
        }
        assertTrue(
            fromWormholeFormat(deliveryData.sourceAddress) == address(setup.source.integration)
        );
        assertTrue(deliveryData.sourceChain == setup.sourceChain);
        assertTrue(deliveryData.maximumRefund == calculatedRefund);
        assertTrue(deliveryData.deliveryHash == deliveryVaaHash);
        assertTrue(keccak256(deliveryData.payload) == keccak256(payload));
    }

    function testInvalidRemoteRefundDoesNotRevert(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        setup.target.relayProvider.updateSupportedChain(setup.sourceChain, false);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChain, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        bytes memory payload = abi.encodePacked(uint256(6));

        setup.source.integration.sendMessageGeneral{value: stack.payment}(
            message,
            setup.targetChain,
            address(setup.target.integration),
            setup.sourceChain,
            address(setup.source.integration),
            0,
            payload
        );

        prepareDeliveryStack(stack, setup);

        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    



    
    */
}
