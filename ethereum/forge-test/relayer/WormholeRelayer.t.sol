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
import {CoreRelayer} from "../../contracts/relayer/coreRelayer/CoreRelayer.sol";
import {MockGenericRelayer} from "./MockGenericRelayer.sol";
import {MockWormhole} from "./MockWormhole.sol";
import {IWormhole} from "../../contracts/interfaces/IWormhole.sol";
import {WormholeSimulator, FakeWormholeSimulator} from "./WormholeSimulator.sol";
import {
    DeliveryData,
    IWormholeReceiver
} from "../../contracts/interfaces/relayer/IWormholeReceiver.sol";
import {AttackForwardIntegration} from "./AttackForwardIntegration.sol";
import {MockRelayerIntegration, XAddress} from "../../contracts/mock/relayer/MockRelayerIntegration.sol";
import {ForwardTester} from "./ForwardTester.sol";
import {TestHelpers} from "./TestHelpers.sol";
import {CoreRelayerSerde} from "../../contracts/relayer/coreRelayer/CoreRelayerSerde.sol";
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

    uint16 MAX_UINT16_VALUE = 65535;
    uint96 MAX_UINT96_VALUE = 79228162514264337593543950335;

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

    function toFeeParamtersTyped(FeeParameters memory feeParams)
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
        uint16 sourceChainId;
        uint16 targetChainId;
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
        FeeParametersTyped memory feeParams = toFeeParamtersTyped(feeParams_);

        s.sourceChainId = 1;
        s.targetChainId = 2;
        s.differentChainId = 3;
        s.source = map[s.sourceChainId];
        s.target = map[s.targetChainId];

        vm.deal(s.source.relayer, type(uint256).max / 2);
        vm.deal(s.target.relayer, type(uint256).max / 2);
        vm.deal(address(this), type(uint256).max / 2);
        vm.deal(address(s.target.integration), type(uint256).max / 2);
        vm.deal(address(s.source.integration), type(uint256).max / 2);

        // set relayProvider prices
        s.source.relayProvider.updatePrice(
            s.targetChainId, gasParams.targetGasPrice, feeParams.targetNativePrice
        );
        s.source.relayProvider.updatePrice(
            s.sourceChainId, gasParams.sourceGasPrice, feeParams.sourceNativePrice
        );
        s.target.relayProvider.updatePrice(
            s.targetChainId, gasParams.targetGasPrice, feeParams.targetNativePrice
        );
        s.target.relayProvider.updatePrice(
            s.sourceChainId, gasParams.sourceGasPrice, feeParams.sourceNativePrice
        );

        s.source.relayProvider.updateDeliverGasOverhead(s.targetChainId, gasParams.evmGasOverhead);
        s.target.relayProvider.updateDeliverGasOverhead(s.sourceChainId, gasParams.evmGasOverhead);

        s.source.wormholeSimulator.setMessageFee(feeParams.wormholeFeeOnSource.unwrap());
        s.target.wormholeSimulator.setMessageFee(feeParams.wormholeFeeOnTarget.unwrap());
    }

    struct Contracts {
        IWormhole wormhole;
        WormholeSimulator wormholeSimulator;
        RelayProvider relayProvider;
        IWormholeRelayer coreRelayer;
        CoreRelayer coreRelayerFull;
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
            mapEntry.relayProvider = helpers.setUpRelayProvider(i);
            mapEntry.coreRelayer =
                helpers.setUpCoreRelayer(mapEntry.wormhole, address(mapEntry.relayProvider));
            mapEntry.coreRelayerFull = CoreRelayer(payable(address(mapEntry.coreRelayer)));
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
                helpers.registerCoreRelayerContract(
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
        vaaKeys[0] = VaaKey(
            chainId,
            toWormholeFormat(emitterAddress),
            sequence
        );
    }

    function vaaKeyArray(
        uint16 chainId,
        uint64 sequence1,
        address emitterAddress1,
        uint64 sequence2,
        address emitterAddress2
    ) internal pure returns (VaaKey[] memory vaaKeys) {
        vaaKeys = new VaaKey[](2);
        vaaKeys[0] = VaaKey(
            chainId,
            toWormholeFormat(emitterAddress1),
            sequence1
        );
        vaaKeys[1] = VaaKey(
            chainId,
            toWormholeFormat(emitterAddress2),
            sequence2
        );
    }

    /**
     * Basic Functionality Tests: Send, Forward, and Resend
     * 
     */

    function testSend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        vm.recordLogs();

        (uint256 deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(setup.targetChainId, 0, gasParams.targetGasLimit);

        setup.source.integration.sendMessage{
            value: deliveryCost + setup.source.wormhole.messageFee()
        }(
            message,
            setup.targetChainId,
            gasParams.targetGasLimit,
            0
        );

        genericRelayer.relay(setup.sourceChainId);

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
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        (uint256 forwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(setup.sourceChainId, 0, REASONABLE_GAS_LIMIT);
        uint256 receiverValue = forwardDeliveryCost + setup.target.wormhole.messageFee();
        vm.assume(receiverValue <= type(uint128).max);

        (uint256 deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(setup.targetChainId, uint128(receiverValue), REASONABLE_GAS_LIMIT_FORWARDS);

        vm.recordLogs();

        setup.source.integration.sendMessageWithForwardedResponse{value: deliveryCost}(
            message,
            forwardedMessage,
            setup.targetChainId,
            REASONABLE_GAS_LIMIT_FORWARDS,
            uint128(receiverValue)
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        genericRelayer.relay(setup.targetChainId);

        assertTrue(
            keccak256(setup.source.integration.getMessage()) == keccak256(forwardedMessage)
        );
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

        (uint256 deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(setup.targetChainId, 0, TOO_LOW_GAS_LIMIT);

        uint64 sequence = setup.source.integration.sendMessage{
            value: deliveryCost + setup.source.wormhole.messageFee()
        }(
            message,
            setup.targetChainId,
            TOO_LOW_GAS_LIMIT,
            0
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(message));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE);

        
        (uint256 newDeliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(setup.targetChainId, 0, REASONABLE_GAS_LIMIT);

        setup.source.integration.resend{
            value: newDeliveryCost + setup.source.wormhole.messageFee()
        }(
            setup.sourceChainId,
            sequence,
            setup.targetChainId,
            REASONABLE_GAS_LIMIT,
            0
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.SUCCESS);
    }

    /**
     * More functionality tests
     */

    function testMultipleForwards(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message,
        bytes memory forwardedMessage
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, REASONABLE_GAS_LIMIT);

        (uint256 firstForwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(setup.sourceChainId, 0, REASONABLE_GAS_LIMIT);
        (uint256 secondForwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(setup.targetChainId, 0, REASONABLE_GAS_LIMIT);

        uint256 receiverValue = firstForwardDeliveryCost + secondForwardDeliveryCost + 2 * setup.target.wormhole.messageFee();
        vm.assume(receiverValue <= type(uint128).max);

        (uint256 deliveryCost,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(setup.targetChainId, uint128(receiverValue), REASONABLE_GAS_LIMIT_FORWARDS*2);

        vm.recordLogs();

        setup.source.integration.sendMessageWithMultiForwardedResponse{value: deliveryCost}(
            message,
            forwardedMessage,
            setup.targetChainId,
            REASONABLE_GAS_LIMIT_FORWARDS*2,
            uint128(receiverValue)
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        genericRelayer.relay(setup.targetChainId);

        assertTrue(keccak256(setup.source.integration.getMessage()) == keccak256(forwardedMessage));

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(forwardedMessage));
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
    )
        public
        returns (StandardSetupTwoChains memory s, FundsCorrectTest memory test)
    {
        s = standardAssumeAndSetupTwoChains(gasParams_, feeParams_, minGasLimit);

        test.refundAddressBalance = s.target.refundAddress.balance;
        test.relayerBalance = s.target.relayer.balance;
        test.rewardAddressBalance = s.source.rewardAddress.balance;
        test.destinationBalance = address(s.target.integration).balance;
        test.sourceContractBalance = address(s.source.coreRelayer).balance;
        test.targetContractBalance = address(s.target.coreRelayer).balance;
        test.receiverValue = feeParams_.receiverValueTarget;
        (test.deliveryPrice, test.targetChainRefundPerGasUnused) = s.source.coreRelayer.quoteEVMDeliveryPrice(s.targetChainId, test.receiverValue, gasParams_.targetGasLimit);
        vm.assume(test.targetChainRefundPerGasUnused > 0);
    }


    function testFundsCorrectForASend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 170000);

        setup.source.integration.sendMessageWithRefund{value: test.deliveryPrice + feeParams.wormholeFeeOnSource}(
            bytes("Hello!"),
            setup.targetChainId,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.targetChainId,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChainId);

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
            test.destinationAmount == test.receiverValue,
            "Receiver value was sent to the contract"
        );
        assertTrue(
            test.rewardAddressAmount == test.deliveryPrice,
            "Reward address was paid correctly"
        );
        assertTrue(test.targetChainRefundPerGasUnused == deliveryData.targetChainRefundPerGasUnused, "Correct value of targetChainRefundPerGasUnused is reported to receiver in deliveryData");
        
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

        setup.source.integration.sendMessageWithRefund{value: test.deliveryPrice + feeParams.wormholeFeeOnSource}(
            bytes("Hello!"),
            setup.targetChainId,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.targetChainId,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChainId);

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
            test.rewardAddressAmount == test.deliveryPrice,
            "Reward address was paid correctly"
        );
        assertTrue(
            test.refundAddressAmount
                == test.receiverValue,
            "Receiver value was refunded"
        );
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

        setup.source.integration.sendMessageWithForwardedResponse{value: test.deliveryPrice + feeParams.wormholeFeeOnSource}(
            bytes("Hello!"),
            bytes("Forwarded Message"),
            setup.targetChainId,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.targetChainId,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChainId);

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
            test.rewardAddressAmount == test.deliveryPrice,
            "Reward address was paid correctly"
        );
        test.gasAmount = uint32(
            gasParams.targetGasLimit - (test.refundAddressAmount - test.receiverValue) / test.targetChainRefundPerGasUnused
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

        (uint256 forwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(setup.sourceChainId, 0, REASONABLE_GAS_LIMIT);
        uint256 receiverValue = forwardDeliveryCost + setup.target.wormhole.messageFee();
        vm.assume(receiverValue <= type(uint128).max);
        vm.assume(feeParams.receiverValueTarget >= receiverValue);

        uint256 rewardAddressBalanceTarget = setup.target.rewardAddress.balance;

        setup.source.integration.sendMessageWithForwardedResponse{
            value: test.deliveryPrice + feeParams.wormholeFeeOnSource
        }(
            bytes("Hello!"),
            bytes("Forwarded Message!"),
            setup.targetChainId,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.targetChainId,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(bytes("Hello!")));
        DeliveryData memory deliveryData = setup.target.integration.getDeliveryData();

        genericRelayer.relay(setup.targetChainId);

        assertTrue(keccak256(setup.source.integration.getMessage()) == keccak256(bytes("Forwarded Message!")));

        test.refundAddressAmount = setup.target.refundAddress.balance - test.refundAddressBalance;

        test.rewardAddressAmount = setup.source.rewardAddress.balance - test.rewardAddressBalance;

        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;

        test.destinationAmount = test.destinationBalance - address(setup.target.integration).balance;

        assertTrue(test.sourceContractBalance == address(setup.source.coreRelayer).balance, "Source contract has extra balance");
        assertTrue(test.targetContractBalance == address(setup.target.coreRelayer).balance, "Target contract has extra balance");
        assertTrue(test.refundAddressAmount == 0, "All refund amount was forwarded");
        assertTrue(
            test.destinationAmount == 0,
            "All receiver amount was sent to forward"
        );
        assertTrue(
            test.rewardAddressAmount == test.deliveryPrice,
            "Source reward address was paid correctly"
        );

        uint256 refundIntermediate = (setup.target.rewardAddress.balance - rewardAddressBalanceTarget)
           + feeParams.wormholeFeeOnTarget - test.receiverValue;

        assertTrue(
            test.targetChainRefundPerGasUnused == deliveryData.targetChainRefundPerGasUnused,
            "Correct value of targetChainRefundPerGasUnused is reported to receiver in deliveryData"
        );
        test.gasAmount =
            uint32(gasParams.targetGasLimit - refundIntermediate / test.targetChainRefundPerGasUnused);

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

        (uint256 forwardDeliveryCost,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(setup.sourceChainId, 0, TOO_LOW_GAS_LIMIT);
        uint256 receiverValue = forwardDeliveryCost + setup.target.wormhole.messageFee();
        vm.assume(receiverValue <= type(uint128).max);
        vm.assume(feeParams.receiverValueTarget < receiverValue);

        setup.source.integration.sendMessageWithForwardedResponse{
            value: test.deliveryPrice + feeParams.wormholeFeeOnSource
        }(
            bytes("Hello!"),
            bytes("Forwarded Message!"),
            setup.targetChainId,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.targetChainId,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(bytes("Hello!")));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.FORWARD_REQUEST_FAILURE);

        test.refundAddressAmount = setup.target.refundAddress.balance - test.refundAddressBalance;

        test.rewardAddressAmount = setup.source.rewardAddress.balance - test.rewardAddressBalance;

        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;

        test.destinationAmount = test.destinationBalance - address(setup.target.integration).balance;

        assertTrue(test.sourceContractBalance == address(setup.source.coreRelayer).balance, "Source contract has extra balance");
        assertTrue(test.targetContractBalance == address(setup.target.coreRelayer).balance, "Target contract has extra balance");
        assertTrue(
            test.destinationAmount == 0,
            "No receiver value was sent to contract"
        );
        assertTrue(
            test.rewardAddressAmount == test.deliveryPrice,
            "Source reward address was paid correctly"
        );

        test.gasAmount =
            uint32(gasParams.targetGasLimit - (test.refundAddressAmount - test.receiverValue) / test.targetChainRefundPerGasUnused);

        console.log(test.gasAmount);

        assertTrue(
            test.relayerPayment == test.refundAddressAmount,
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

    function testFundsCorrectForAResend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        vm.recordLogs();
        (StandardSetupTwoChains memory setup, FundsCorrectTest memory test) =
            setupFundsCorrectTest(gasParams, feeParams, 170000);

        (uint256 notEnoughDeliveryPrice,) = setup.source.coreRelayer.quoteEVMDeliveryPrice(setup.targetChainId, 0, TOO_LOW_GAS_LIMIT);

        uint64 sequence = setup.source.integration.sendMessageWithRefund{value: notEnoughDeliveryPrice + feeParams.wormholeFeeOnSource}(
            bytes("Hello!"),
            setup.targetChainId,
            TOO_LOW_GAS_LIMIT,
            0,
            setup.targetChainId,
            setup.target.refundAddress
        );

        genericRelayer.relay(setup.sourceChainId);
        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(bytes("Hello!")));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE);

        (, test) =
            setupFundsCorrectTest(gasParams, feeParams, 170000);


        //call a resend for the orignal message
        setup.source.integration.resend{
            value: test.deliveryPrice + setup.source.wormhole.messageFee()
        }(
            setup.sourceChainId,
            sequence,
            setup.targetChainId,
            gasParams.targetGasLimit,
            test.receiverValue
        );


        genericRelayer.relay(setup.sourceChainId);


        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(bytes("Hello!")));
        assertTrue(getDeliveryStatus() == IWormholeRelayerDelivery.DeliveryStatus.SUCCESS);
        DeliveryData memory deliveryData = setup.target.integration.getDeliveryData();

        test.refundAddressAmount =  setup.target.refundAddress.balance - test.refundAddressBalance;
        test.rewardAddressAmount =  setup.source.rewardAddress.balance - test.rewardAddressBalance;
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
            test.rewardAddressAmount == test.deliveryPrice,
            "Reward address was paid correctly"
        );
        test.gasAmount = uint32(
            gasParams.targetGasLimit - test.refundAddressAmount / test.targetChainRefundPerGasUnused
        );
        assertTrue(test.targetChainRefundPerGasUnused == deliveryData.targetChainRefundPerGasUnused);
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

        setup.source.integration.sendMessageWithRefund{value: test.deliveryPrice + feeParams.wormholeFeeOnSource}(
            bytes("Hello!"),
            setup.targetChainId,
            gasParams.targetGasLimit,
            test.receiverValue,
            setup.sourceChainId,
            setup.source.refundAddress
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(bytes("Hello!")));
        DeliveryData memory deliveryData = setup.target.integration.getDeliveryData();

        genericRelayer.relay(setup.targetChainId);



        assertTrue(
            test.deliveryPrice
                == setup.source.rewardAddress.balance - test.rewardAddressBalance,
            "The source to target relayer's reward address was paid appropriately"
        );
        // Calculate maximum refund for source->target delivery, and check against Delivery Data
        assertTrue(
            test.targetChainRefundPerGasUnused == deliveryData.targetChainRefundPerGasUnused,
            "Correct value of targetChainRefundPerGasUnused is reported to receiver in deliveryData"
        );
        uint256 amountToGetInRefundTarget =
            (setup.target.rewardAddress.balance - refundRewardAddressBalance);
        
        
        uint256 refundSource = 0;
        (uint256 baseFee,) = setup.target.coreRelayer.quoteEVMDeliveryPrice(setup.sourceChainId, 0, 0);
        
        vm.assume(amountToGetInRefundTarget > baseFee );
        if (amountToGetInRefundTarget > baseFee ) {
            refundSource = setup.target.coreRelayer.quoteAssetConversion(setup.sourceChainId, 
                uint128(amountToGetInRefundTarget - baseFee), setup.target.coreRelayer.getDefaultRelayProvider()
            );
        }

        // Calculate amount that must have been spent on gas, by reverse engineering from the amount that was paid to the provider's reward address on the target chain
        test.gasAmount = uint32(
            gasParams.targetGasLimit
                - (amountToGetInRefundTarget + feeParams.wormholeFeeOnTarget)
                    / deliveryData.targetChainRefundPerGasUnused
        );
        test.relayerPayment = test.relayerBalance - setup.target.relayer.balance;
        test.destinationAmount = address(setup.target.integration).balance - test.destinationBalance;

        assertTrue(
            test.destinationAmount == feeParams.receiverValueTarget,
            "Receiver value was sent to the contract"
        );
        assertTrue(
            test.relayerPayment == amountToGetInRefundTarget + feeParams.wormholeFeeOnTarget + feeParams.receiverValueTarget,
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

    /*
    function testSendCheckConsistencyLevel(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        // estimate the cost based on the intialized values
        uint256 maxTransactionFee = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        uint64 sequence = setup.source.wormhole.publishMessage{value: feeParams.wormholeFeeOnSource}(
            0, bytes("Hi!"), 200
        );

        setup.source.coreRelayer.send{value: maxTransactionFee + feeParams.wormholeFeeOnSource}(
            setup.targetChainId,
            toWormholeFormat(address(setup.target.integration)),
            setup.targetChainId,
            toWormholeFormat(address(setup.target.integration)),
            maxTransactionFee,
            0,
            bytes(""),
            vaaKeyArray(setup.sourceChainId, sequence, address(this)),
            uint8(23)
        );

        Vm.Log memory log = vm.getRecordedLogs()[1];

        // Parse the consistency level from the published VAA
        (uint8 consistencyLevel,) = log.data.asUint8(32 + 32 + 32 + 32 - 1);

        assertTrue(consistencyLevel == 23);
    }

    function testSendUsingVaaHashAsVaaKey(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        // estimate the cost based on the intialized values
        uint256 maxTransactionFee = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        vm.prank(address(setup.source.integration));
        setup.source.wormhole.publishMessage{value: feeParams.wormholeFeeOnSource}(
            0, abi.encodePacked(uint8(0)), 200
        );
        vm.prank(address(setup.source.integration));
        setup.source.wormhole.publishMessage{value: feeParams.wormholeFeeOnSource}(
            0, bytes("Hi!"), 200
        );
        Vm.Log[] memory logs = vm.getRecordedLogs();

        VaaKey[] memory vaaKeyArr = new VaaKey[](2);
        vaaKeyArr[0] = VaaKey({
            infoType: VaaKeyType.VAAHASH,
            chainId: setup.sourceChainId,
            emitterAddress: bytes32(0x0),
            sequence: 0,
            vaaHash: relayerWormhole.parseVM(
                relayerWormholeSimulator.fetchSignedMessageFromLogs(
                    logs[1], setup.sourceChainId, address(setup.source.integration)
                )
                ).hash
        });
        vaaKeyArr[1] = VaaKey({
            infoType: VaaKeyType.VAAHASH,
            chainId: setup.sourceChainId,
            emitterAddress: bytes32(0x0),
            sequence: 0,
            vaaHash: relayerWormhole.parseVM(
                relayerWormholeSimulator.fetchSignedMessageFromLogs(
                    logs[0], setup.sourceChainId, address(setup.source.integration)
                )
                ).hash
        });

        setup.source.coreRelayer.send{value: maxTransactionFee + feeParams.wormholeFeeOnSource}(
            setup.targetChainId,
            toWormholeFormat(address(setup.target.integration)),
            setup.targetChainId,
            toWormholeFormat(address(setup.target.integration)),
            maxTransactionFee,
            0,
            bytes(""),
            vaaKeyArr,
            200
        );
        Vm.Log[] memory newLogs = new Vm.Log[](3);
        newLogs[0] = logs[0];
        newLogs[1] = logs[1];
        newLogs[2] = vm.getRecordedLogs()[0];

        genericRelayer.relay(newLogs, setup.sourceChainId, bytes(""));

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256("Hi!"));
    }

    function testMultipleForwards(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 2000000);

        uint256 payment = assumeAndGetForwardPayment(
            gasParams.targetGasLimit, 500000, setup, gasParams, feeParams
        ) * 2;

        uint16[] memory chains = new uint16[](2);
        bytes[] memory newMessages = new bytes[](2);
        uint32[] memory gasLimits = new uint32[](2);
        newMessages[0] = message;
        newMessages[1] = abi.encodePacked(uint8(0));
        chains[0] = setup.sourceChainId;
        chains[1] = setup.targetChainId;
        gasLimits[0] = 500000;
        gasLimits[1] = 500000;

        MockRelayerIntegration.FurtherInstructions memory furtherInstructions =
        MockRelayerIntegration.FurtherInstructions({
            keepSending: true,
            newMessages: newMessages,
            chains: chains,
            gasLimits: gasLimits
        });

        vm.recordLogs();

        uint16[] memory sendChains = new uint16[](1);
        sendChains[0] = setup.targetChainId;

        uint256[] memory computeBudgets = new uint256[](1);
        computeBudgets[0] = payment - setup.source.wormhole.messageFee();

        setup.source.integration.sendMessagesWithFurtherInstructions{value: payment}(
            new bytes[](0), furtherInstructions, sendChains, computeBudgets
        );

        genericRelayer.relay(setup.sourceChainId);

        genericRelayer.relay(setup.targetChainId);

        assertTrue(keccak256(setup.source.integration.getMessage()) == keccak256(message));

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    

    function testNoFundsLostForASendCrossChainRefund(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        uint256 refundAddressBalance = setup.source.refundAddress.balance;
        uint256 relayerBalance = setup.target.relayer.balance;
        uint256 rewardAddressBalance = setup.source.rewardAddress.balance;
        uint256 refundRewardAddressBalance = setup.target.rewardAddress.balance;
        uint256 refundRelayerBalance = setup.source.relayer.balance;
        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + uint256(3) * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageGeneral{value: payment}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            setup.sourceChainId,
            address(setup.source.refundAddress),
            0,
            bytes("")
        );

        genericRelayer.relay(setup.sourceChainId);

        genericRelayer.relay(setup.targetChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        uint256 USDcost = (
            uint256(payment) - uint256(3) * map[setup.sourceChainId].wormhole.messageFee()
        ) * feeParams.sourceNativePrice
            - (setup.source.refundAddress.balance - refundAddressBalance) * feeParams.sourceNativePrice;

        uint256 relayerProfit = uint256(feeParams.sourceNativePrice)
            * (setup.source.rewardAddress.balance - rewardAddressBalance)
            - feeParams.targetNativePrice * (relayerBalance - setup.target.relayer.balance);
        uint256 refundRelayerProfit = uint256(feeParams.targetNativePrice)
            * (setup.target.rewardAddress.balance - refundRewardAddressBalance)
            - feeParams.sourceNativePrice * (refundRelayerBalance - setup.source.relayer.balance);

        if (refundRelayerProfit > 0) {
            USDcost -= map[setup.targetChainId].wormhole.messageFee() * feeParams.targetNativePrice;
        }

        if (refundRelayerProfit > 0) {
            assertTrue(
                setup.source.refundAddress.balance > refundAddressBalance,
                "The cross chain refund went through"
            );
        }
        assertTrue(
            USDcost - (relayerProfit + refundRelayerProfit) == 0, "We did not lose any funds"
        );
    }

    

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
            setup.targetChainId, feeParams.receiverValueTarget, address(setup.source.relayProvider)
        );

        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, 21000, address(setup.source.relayProvider)
        ) + uint256(3) * setup.source.wormhole.messageFee() + receiverValueSource;

        setup.source.integration.sendMessageGeneral{value: payment}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            setup.targetChainId,
            address(setup.target.refundAddress),
            receiverValueSource,
            bytes("")
        );

        genericRelayer.relay(setup.sourceChainId);

        uint256 USDcost = uint256(
            payment - uint256(3) * map[setup.sourceChainId].wormhole.messageFee()
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
                setup.targetChainId, gasFirst, address(setup.source.relayProvider)
            ) < uint256(2) ** 221
        );
        vm.assume(
            setup.target.coreRelayer.quoteGas(
                setup.sourceChainId, gasSecond, address(setup.target.relayProvider)
            ) < uint256(2) ** 221 / feeParams.targetNativePrice
        );

        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasFirst, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        uint256 payment2 = (
            setup.target.coreRelayer.quoteGas(
                setup.sourceChainId, gasSecond, address(setup.target.relayProvider)
            ) + uint256(2) * setup.target.wormhole.messageFee()
        ) * feeParams.targetNativePrice / feeParams.sourceNativePrice + 1;

        vm.assume((payment + payment2 * 105 / 100 + 1) < (uint256(2) ** 222));

        return (payment + payment2 * 105 / 100 + 1);
    }


    function testNoFundsLostForAForward(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        uint256 payment = assumeAndGetForwardPayment(
            gasParams.targetGasLimit, 500000, setup, gasParams, feeParams
        );

        vm.recordLogs();

        uint256 sourceIntegrationBalance = address(setup.source.integration).balance;
        uint256 sourceRelayerBalance = address(setup.source.relayer).balance;
        uint256 targetRelayerBalance = address(setup.target.relayer).balance;

        setup.source.integration.sendMessageWithForwardedResponse{value: payment}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            address(setup.target.refundAddress),
            0
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
        assertTrue(address(setup.source.integration).balance == sourceIntegrationBalance);

        genericRelayer.relay(setup.targetChainId);

        assertTrue(
            keccak256(setup.source.integration.getMessage()) == keccak256(bytes("received!"))
        );

        uint256 USDCost = (payment - uint256(3) * feeParams.wormholeFeeOnSource)
            * feeParams.sourceNativePrice
            - uint256(1) * feeParams.wormholeFeeOnTarget * feeParams.targetNativePrice;
        USDCost -= (address(setup.source.integration).balance - sourceIntegrationBalance)
            * feeParams.sourceNativePrice;
        uint256 relayerProfit = (
            address(setup.source.rewardAddress).balance * feeParams.sourceNativePrice
                + address(setup.target.rewardAddress).balance * feeParams.targetNativePrice
        )
            - (sourceRelayerBalance - address(setup.source.relayer).balance)
                * feeParams.sourceNativePrice
            - (targetRelayerBalance - address(setup.target.relayer).balance)
                * feeParams.targetNativePrice;
        assertTrue(USDCost == relayerProfit, "We did not lose any funds along the way");
    }

    function testNoFundsLostForAForwardFailure(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
        vm.assume(keccak256(message) != keccak256(bytes("")));
        vm.assume(
            feeParams.receiverValueTarget
                < setup.target.coreRelayer.quoteGas(1, 500000, address(setup.target.relayProvider))
        );
        vm.assume(
            uint256(10) * feeParams.targetNativePrice * gasParams.targetGasPrice
                < uint256(1) * feeParams.sourceNativePrice * gasParams.sourceGasPrice
        );

        uint256 payment = setup.source.coreRelayer.quoteGas(
            2, 1000000, address(setup.source.relayProvider)
        ) + uint256(3) * feeParams.wormholeFeeOnSource;

        uint256 receiverValueSource = setup.source.coreRelayer.quoteReceiverValue(
            2, feeParams.receiverValueTarget, address(setup.source.relayProvider)
        );

        vm.assume(payment + receiverValueSource < uint256(2) ** 222);
        vm.assume(
            uint256(2) ** 255 / feeParams.sourceNativePrice > receiverValueSource * uint256(100)
        );

        vm.recordLogs();

        uint256 targetRefundBalance = address(setup.target.refundAddress).balance;
        uint256 sourceRelayerBalance = address(setup.source.relayer).balance;
        uint256 targetRelayerBalance = address(setup.target.relayer).balance;

        setup.source.integration.sendMessageWithForwardedResponse{value: payment}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            address(setup.target.refundAddress),
            0
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(message));

        uint256 USDCost = (payment - uint256(3) * feeParams.wormholeFeeOnSource)
            * feeParams.sourceNativePrice
            - uint256(0) * feeParams.wormholeFeeOnTarget * feeParams.targetNativePrice;
        USDCost -= (address(setup.target.refundAddress).balance - targetRefundBalance)
            * feeParams.targetNativePrice;
        uint256 relayerProfit = (
            address(setup.source.rewardAddress).balance * feeParams.sourceNativePrice
                + address(setup.target.rewardAddress).balance * feeParams.targetNativePrice
        )
            - (sourceRelayerBalance - address(setup.source.relayer).balance)
                * feeParams.sourceNativePrice
            - (targetRelayerBalance - address(setup.target.relayer).balance)
                * feeParams.targetNativePrice;
        assertTrue(USDCost == relayerProfit, "We did not lose any funds along the way");
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
            setup.targetChainId, 1000000, address(setup.source.relayProvider)
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

        genericRelayer.relay(setup.sourceChainId);

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
            setup.sourceChainId,
            stack.sequence1,
            address(setup.source.integration),
            stack.sequence2,
            address(setup.source.integration)
        );
        setup.source.coreRelayer.send{value: stack.payment + stack.wormholeFee}(
            setup.targetChainId,
            stack.targetAddress,
            setup.targetChainId,
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
        // 5. The forward request is left as is in the `CoreRelayer` state.
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
                uint256(keccak256(abi.encodePacked(bytes("attackerAddress"), setup.sourceChainId)))
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
                setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
            ) < uint256(2) ** 222
        );
        vm.assume(
            setup.target.coreRelayer.quoteGas(
                setup.sourceChainId, 500000, address(setup.target.relayProvider)
            ) < uint256(2) ** 222 / feeParams.targetNativePrice
        );

        // Estimate the cost based on the initialized values
        uint256 computeBudget = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        {
            AttackForwardIntegration attackerContract =
            new AttackForwardIntegration(setup.target.wormhole, setup.target.coreRelayer, setup.targetChainId, attackerSourceAddress);
            bytes memory attackMsg = "attack";

            vm.recordLogs();

            // The attacker requests the message to be sent to the malicious contract.
            // It is critical that the refund and destination (aka integrator) addresses are the same.
            setup.source.integration.sendMessage{
                value: computeBudget + uint256(3) * setup.source.wormhole.messageFee()
            }(attackMsg, setup.targetChainId, address(attackerContract));

            // The relayer triggers the call to the malicious contract.
            genericRelayer.relay(setup.sourceChainId);

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
                setup.targetChainId,
                address(setup.target.integration),
                address(setup.target.refundAddress),
                bytes("")
            );

            // The relayer delivers the victim's message.
            // During the delivery process, the forward request injected by the malicious contract is acknowledged.
            // The victim's refund address is not called due to this.
            genericRelayer.relay(setup.sourceChainId);

            // Ensures the message was received.
            assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(victimMsg));
            // Here we assert that the victim's refund is safe.
            assertTrue(victimBalancePreDelivery < setup.target.refundAddress.balance);
        }

        genericRelayer.relay(setup.targetChainId);

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
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        uint256 oldBalance = address(setup.target.integration).balance;

        uint256 newReceiverValueSource = setup.source.coreRelayer.quoteReceiverValue(
            setup.targetChainId, feeParams.receiverValueTarget, address(setup.source.relayProvider)
        );

        vm.deal(address(this), payment + newReceiverValueSource);

        setup.source.integration.sendMessageGeneral{value: payment + newReceiverValueSource}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            setup.targetChainId,
            address(0x0),
            newReceiverValueSource,
            bytes("")
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        assertTrue(
            address(setup.target.integration).balance >= oldBalance + feeParams.receiverValueTarget
        );
    }

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
        uint256 paymentNotEnough;
        Vm.Log[] entries;
        bytes actualVM1;
        bytes actualVM2;
        bytes deliveryVM;
        bytes[] encodedVMs;
        IWormhole.VM parsed;
        uint256 budget;
        TargetDeliveryParameters package;
        DeliveryInstruction instruction;
    }

    function prepareDeliveryStack(
        DeliveryStack memory stack,
        StandardSetupTwoChains memory setup
    ) internal {
        stack.entries = vm.getRecordedLogs();

        stack.actualVM1 = relayerWormholeSimulator.fetchSignedMessageFromLogs(
            stack.entries[0], setup.sourceChainId, address(setup.source.integration)
        );

        stack.actualVM2 = relayerWormholeSimulator.fetchSignedMessageFromLogs(
            stack.entries[1], setup.sourceChainId, address(setup.source.integration)
        );

        stack.deliveryVM = relayerWormholeSimulator.fetchSignedMessageFromLogs(
            stack.entries[2], setup.sourceChainId, address(setup.source.coreRelayer)
        );

        stack.encodedVMs = new bytes[](2);
        stack.encodedVMs[0] = stack.actualVM1;
        stack.encodedVMs[1] = stack.actualVM2;

        stack.package = TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides: bytes("")
        });

        stack.parsed = relayerWormhole.parseVM(stack.deliveryVM);
        stack.instruction = CoreRelayerSerde.decodeDeliveryInstruction(stack.parsed.payload);

        stack.budget = Wei.unwrap(
            stack.instruction.maximumRefundTarget + stack.instruction.receiverValueTarget
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

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            setup.target.refundAddress,
            bytes("")
        );

        prepareDeliveryStack(stack, setup);

        bytes memory fakeVM = abi.encodePacked(stack.deliveryVM);

        invalidateVM(fakeVM, setup.target.wormholeSimulator);

        stack.deliveryVM = fakeVM;

        stack.package = TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides: bytes("")
        });

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("InvalidDeliveryVaa(string)", ""));
        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);
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

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            setup.target.refundAddress,
            bytes("")
        );

        prepareDeliveryStack(stack, setup);

        stack.deliveryVM = stack.encodedVMs[0];

        stack.package = TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides: bytes("")
        });

        vm.prank(setup.target.relayer);
        vm.expectRevert(
            abi.encodeWithSelector(
                InvalidEmitter.selector,
                setup.source.integration,
                setup.source.coreRelayer,
                setup.source.chainId
            )
        );
        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);
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

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            setup.target.refundAddress,
            bytes("")
        );

        prepareDeliveryStack(stack, setup);

        vm.prank(setup.target.relayer);
        vm.expectRevert(
            abi.encodeWithSelector(
                InsufficientRelayerFunds.selector, stack.budget - 1, stack.budget
            )
        );
        setup.target.coreRelayerFull.deliver{value: stack.budget - 1}(stack.package);
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

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            setup.target.refundAddress,
            bytes("")
        );

        prepareDeliveryStack(stack, setup);

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("TargetChainIsNotThisChain(uint16)", 2));
        map[setup.differentChainId].coreRelayerFull.deliver{value: stack.budget}(stack.package);
    }

    struct SendStackTooDeep {
        uint256 payment;
        Send deliveryRequest;
        uint256 deliveryOverhead;
        Send badSend;
    }

    function testRevertSendMsgValueTooLow(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        uint256 maxTransactionFee = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        uint64 sequence = setup.source.wormhole.publishMessage{
            value: setup.source.wormhole.messageFee()
        }(1, message, 200);

        bytes memory emptyArray;

        VaaKey[] memory vaaKeys = vaaKeyArray(setup.sourceChainId, sequence, address(this));

        uint256 wormholeFee = setup.source.wormhole.messageFee();

        vm.expectRevert(
            abi.encodeWithSelector(
                InvalidMsgValue.selector,
                maxTransactionFee + wormholeFee - 1,
                maxTransactionFee + wormholeFee
            )
        );
        setup.source.coreRelayer.sendToEvm{value: maxTransactionFee + wormholeFee - 1}(
            setup.targetChainId,
            address(setup.target.integration),
            setup.targetChainId,
            address(setup.target.refundAddress),
            maxTransactionFee,
            0,
            emptyArray,
            vaaKeys,
            200
        );
    }

    function testRevertSendMsgValueTooMuch(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        uint256 maxTransactionFee = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        uint64 sequence = setup.source.wormhole.publishMessage{
            value: setup.source.wormhole.messageFee()
        }(1, message, 200);

        bytes memory emptyArray;

        VaaKey[] memory vaaKeys = vaaKeyArray(setup.sourceChainId, sequence, address(this));

        uint256 wormholeFee = setup.source.wormhole.messageFee();

        vm.expectRevert(
            abi.encodeWithSelector(
                InvalidMsgValue.selector,
                maxTransactionFee + wormholeFee + 1,
                maxTransactionFee + wormholeFee
            )
        );
        setup.source.coreRelayer.sendToEvm{value: maxTransactionFee + wormholeFee + 1}(
            setup.targetChainId,
            address(setup.target.integration),
            setup.targetChainId,
            address(setup.target.refundAddress),
            maxTransactionFee,
            0,
            emptyArray,
            vaaKeys,
            200
        );
    }

    ForwardTester forwardTester;

    struct ForwardStack {
        bytes32 targetAddress;
        uint256 payment;
        uint256 wormholeFee;
    }

    function executeForwardTest(
        ForwardTester.Action test,
        IWormholeRelayerDelivery.DeliveryStatus desiredOutcome,
        StandardSetupTwoChains memory setup,
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) internal {
        ForwardStack memory stack;
        vm.recordLogs();
        forwardTester =
        new ForwardTester(address(setup.target.wormhole), address(setup.target.coreRelayer), address(setup.target.wormholeSimulator));
        vm.deal(address(forwardTester), type(uint256).max / 2);
        stack.targetAddress = toWormholeFormat(address(forwardTester));
        stack.payment = assumeAndGetForwardPayment(
            gasParams.targetGasLimit, 500000, setup, gasParams, feeParams
        );
        stack.wormholeFee = setup.source.wormhole.messageFee();
        uint64 sequence = setup.source.wormhole.publishMessage{value: stack.wormholeFee}(
            1, abi.encodePacked(uint8(test)), 200
        );
        setup.source.coreRelayer.send{value: stack.payment}(
            setup.targetChainId,
            stack.targetAddress,
            setup.targetChainId,
            stack.targetAddress,
            stack.payment - stack.wormholeFee,
            0,
            bytes(""),
            vaaKeyArray(setup.sourceChainId, sequence, address(this)),
            200
        );
        genericRelayer.relay(setup.sourceChainId);
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
            gasParams,
            feeParams
        );
    }

    function testRevertForwardNoDeliveryInProgress(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        bytes32 targetAddress = toWormholeFormat(address(forwardTester));

        VaaKey[] memory msgInfoArray = vaaKeyArray(0, 0, address(this));
        vm.expectRevert(abi.encodeWithSignature("NoDeliveryInProgress()"));
        setup.source.coreRelayer.forward(
            setup.targetChainId,
            targetAddress,
            setup.targetChainId,
            targetAddress,
            0,
            0,
            bytes(""),
            msgInfoArray,
            200
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
            gasParams,
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
            gasParams,
            feeParams
        );
    }

    function testRevertForwardMsgValueTooMuch(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        setup.target.relayProvider.updateMaximumBudget(
            setup.sourceChainId, Wei.wrap(uint192(10000 - 1) * gasParams.sourceGasPrice)
        );

        executeForwardTest(
            ForwardTester.Action.MsgValueTooMuch,
            IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE,
            setup,
            gasParams,
            feeParams
        );
    }

    function testRevertTargetNotSupported(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        // estimate the cost based on the intialized values
        uint256 maxTransactionFee = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        uint256 wormholeFee = setup.source.wormhole.messageFee();

        vm.expectRevert(
            abi.encodeWithSelector(
                RelayProviderDoesNotSupportTargetChain.selector,
                address(setup.source.relayProvider),
                uint16(32)
            )
        );
        setup.source.integration.sendMessageWithRefundAddress{
            value: maxTransactionFee + uint256(3) * wormholeFee
        }(
            message,
            32,
            address(setup.target.integration),
            address(setup.target.refundAddress),
            bytes("")
        );
    }

    function testToAndFromWormholeFormat(address msg1) public {
        assertTrue(toWormholeFormat(msg1) == bytes32(uint256(uint160(msg1))));
        assertTrue(fromWormholeFormat(toWormholeFormat(msg1)) == msg1);
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
            targetChainId: 1,
            targetAddress: bytes32(""),
            refundAddress: bytes32(""),
            refundChainId: 2,
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
            CoreRelayerSerde.decodeDeliveryInstruction(CoreRelayerSerde.encode(instruction));

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
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + setup.source.wormhole.messageFee()*3;

        uint256 maxTransactionFee = payment - 3*setup.source.wormhole.messageFee();

        bytes memory payload = abi.encodePacked(uint256(6));

        setup.source.integration.sendMessageWithPayload{value: payment}(
            bytes(""), setup.targetChainId, address(setup.target.integration), payload
        );

        genericRelayer.relay(setup.sourceChainId);

        bytes32 deliveryVaaHash = getDeliveryVAAHash(vm.getRecordedLogs());

        DeliveryData memory deliveryData = setup.target.integration.getDeliveryData();

        uint256 calculatedRefund = 0;
        if (
            maxTransactionFee
                > setup.source.relayProvider.quoteDeliveryOverhead(setup.targetChainId).unwrap()
        ) {
            calculatedRefund = (
                maxTransactionFee
                    - setup.source.relayProvider.quoteDeliveryOverhead(setup.targetChainId).unwrap()
            ) * feeParams.sourceNativePrice * 100 / (uint256(feeParams.targetNativePrice) * 105);
        }
        assertTrue(
            fromWormholeFormat(deliveryData.sourceAddress) == address(setup.source.integration)
        );
        assertTrue(deliveryData.sourceChainId == setup.sourceChainId);
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

        setup.target.relayProvider.updateSupportedChain(setup.sourceChainId, false);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        bytes memory payload = abi.encodePacked(uint256(6));

        setup.source.integration.sendMessageGeneral{value: stack.payment}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            setup.sourceChainId,
            address(setup.source.integration),
            0,
            payload
        );

        prepareDeliveryStack(stack, setup);

        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    function testDeliverWithOverrides(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            setup.target.refundAddress,
            bytes("")
        );

        prepareDeliveryStack(stack, setup);

        DeliveryOverride memory deliveryOverride = DeliveryOverride(
            stack.instruction.executionParameters.gasLimit,
            stack.instruction.maximumRefundTarget,
            stack.instruction.receiverValueTarget,
            stack.deliveryVaaHash //really redeliveryHash
        );

        stack.package = TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides: CoreRelayerSerde.encode(deliveryOverride)
        });

        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);
        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    function testRevertDeliverWithOverrideGasLimit(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            bytes(""),
            setup.targetChainId,
            address(setup.target.integration),
            setup.target.refundAddress,
            bytes("")
        );

        prepareDeliveryStack(stack, setup);

        DeliveryOverride memory deliveryOverride = DeliveryOverride(
            stack.instruction.executionParameters.gasLimit - Gas.wrap(1),
            stack.instruction.maximumRefundTarget,
            stack.instruction.receiverValueTarget,
            stack.deliveryVaaHash //really redeliveryHash
        );

        stack.package = TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides: CoreRelayerSerde.encode(deliveryOverride)
        });

        vm.expectRevert(abi.encodeWithSignature("InvalidOverrideGasLimit()"));
        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);
    }

    function testRevertDeliverWithOverrideReceiverValue(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        vm.assume(feeParams.receiverValueTarget > 0);

        uint256 receiverValueSource = setup.source.coreRelayer.quoteReceiverValue(
            setup.targetChainId, feeParams.receiverValueTarget, address(setup.source.relayProvider)
        );

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee() + receiverValueSource;

        setup.source.integration.sendMessageGeneral{value: stack.payment}(
            bytes(""),
            setup.targetChainId,
            address(setup.target.integration),
            setup.targetChainId,
            setup.target.refundAddress,
            receiverValueSource,
            bytes("")
        );

        prepareDeliveryStack(stack, setup);

        DeliveryOverride memory deliveryOverride = DeliveryOverride(
            stack.instruction.executionParameters.gasLimit,
            stack.instruction.maximumRefundTarget,
            stack.instruction.receiverValueTarget - Wei.wrap(1),
            stack.deliveryVaaHash //really redeliveryHash
        );

        stack.package = TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides: CoreRelayerSerde.encode(deliveryOverride)
        });

        vm.expectRevert(abi.encodeWithSignature("InvalidOverrideReceiverValue()"));
        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);
    }

    function testRevertDeliverWithOverrideMaximumRefund(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            bytes(""),
            setup.targetChainId,
            address(setup.target.integration),
            setup.target.refundAddress,
            bytes("")
        );

        prepareDeliveryStack(stack, setup);

        DeliveryOverride memory deliveryOverride = DeliveryOverride(
            stack.instruction.executionParameters.gasLimit,
            stack.instruction.maximumRefundTarget - Wei.wrap(1),
            stack.instruction.receiverValueTarget,
            stack.deliveryVaaHash //really redeliveryHash
        );

        stack.package = TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides: CoreRelayerSerde.encode(deliveryOverride)
        });

        vm.expectRevert(abi.encodeWithSignature("InvalidOverrideMaximumRefund()"));
        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);
    }



    function testRedeliveryFailAndSucceed(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup =
            standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        vm.assume(keccak256(message) != keccak256(bytes("")));

        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, 10000, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        //send an original message
        uint64 sequence = setup.source.integration.sendMessageWithRefundAddress{value: payment}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            setup.target.refundAddress,
            bytes("")
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(message));

        //create a VAA key for the original message
        VaaKey memory vaaKey = VaaKey(
            VaaKeyType.EMITTER_SEQUENCE,
            setup.sourceChainId,
            bytes32(uint256(uint160(address(setup.source.integration)))),
            sequence,
            bytes32(0x0)
        );

        uint256 newPayment;
        for (uint8 i = 2; i < 10; i++) {
            newPayment = setup.source.coreRelayer.quoteGas(
                setup.targetChainId, uint32(i) * 10000, address(setup.source.relayProvider)
            ) + setup.source.wormhole.messageFee();

            //call a resend for the orignal message
            setup.source.coreRelayer.resend{value: newPayment}(
                vaaKey,
                newPayment - feeParams.wormholeFeeOnSource, //newMaxTransactionFee
                0, //new receiver
                setup.targetChainId,
                address(setup.source.relayProvider)
            );

            genericRelayer.relay(setup.sourceChainId);

            assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(message));

            Vm.Log[] memory logs = vm.getRecordedLogs();
            assertTrue(
                getDeliveryStatus(logs[logs.length - 1])
                    == IWormholeRelayerDelivery.DeliveryStatus.RECEIVER_FAILURE
            );
        }

        newPayment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + setup.source.wormhole.messageFee();

        //call a resend for the orignal message
        setup.source.coreRelayer.resend{value: newPayment}(
            vaaKey,
            newPayment - feeParams.wormholeFeeOnSource, //newMaxTransactionFee
            0, //new receiver
            setup.targetChainId,
            address(setup.source.relayProvider)
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    
    */
}
