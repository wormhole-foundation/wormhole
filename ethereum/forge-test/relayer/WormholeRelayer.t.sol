// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {IRelayProvider} from "../../contracts/interfaces/relayer/IRelayProvider.sol";
import {RelayProvider} from "../../contracts/relayer/relayProvider/RelayProvider.sol";
import {RelayProviderSetup} from "../../contracts/relayer/relayProvider/RelayProviderSetup.sol";
import {RelayProviderImplementation} from "../../contracts/relayer/relayProvider/RelayProviderImplementation.sol";
import {RelayProviderProxy} from "../../contracts/relayer/relayProvider/RelayProviderProxy.sol";
import {RelayProviderMessages} from "../../contracts/relayer/relayProvider/RelayProviderMessages.sol";
import {RelayProviderStructs} from "../../contracts/relayer/relayProvider/RelayProviderStructs.sol";
import {IWormholeRelayer} from "../../contracts/interfaces/relayer/IWormholeRelayer.sol";
import {IDelivery} from "../../contracts/interfaces/relayer/IDelivery.sol";
import {CoreRelayer} from "../../contracts/relayer/coreRelayer/CoreRelayer.sol";
import {IWormholeRelayerInternalStructs} from "../../contracts/interfaces/relayer/IWormholeRelayerInternalStructs.sol";
import {CoreRelayerSetup} from "../../contracts/relayer/coreRelayer/CoreRelayerSetup.sol";
import {CoreRelayerImplementation} from "../../contracts/relayer/coreRelayer/CoreRelayerImplementation.sol";
import {CoreRelayerProxy} from "../../contracts/relayer/coreRelayer/CoreRelayerProxy.sol";
import {CoreRelayerMessages} from "../../contracts/relayer/coreRelayer/CoreRelayerMessages.sol";
import {CoreRelayerGovernance} from "../../contracts/relayer/coreRelayer/CoreRelayerGovernance.sol";
import {MockGenericRelayer} from "./MockGenericRelayer.sol";
import {MockWormhole} from "./MockWormhole.sol";
import {IWormhole} from "../../contracts/interfaces/IWormhole.sol";
import {WormholeSimulator, FakeWormholeSimulator} from "./WormholeSimulator.sol";
import {IWormholeReceiver} from "../../contracts/interfaces/relayer/IWormholeReceiver.sol";
import {AttackForwardIntegration} from "./AttackForwardIntegration.sol";
import {MockRelayerIntegration, Structs} from "../../contracts/mock/MockRelayerIntegration.sol";
import {ForwardTester} from "./ForwardTester.sol";
import {TestHelpers} from "./TestHelpers.sol";
import "../../contracts/libraries/external/BytesLib.sol";

import "forge-std/Test.sol";
import "forge-std/console.sol";
import "forge-std/Vm.sol";

contract WormholeRelayerTests is Test {
    using BytesLib for bytes;

    uint16 MAX_UINT16_VALUE = 65535;
    uint96 MAX_UINT96_VALUE = 79228162514264337593543950335;

    struct GasParameters {
        uint32 evmGasOverhead;
        uint32 targetGasLimit;
        uint128 targetGasPrice;
        uint128 sourceGasPrice;
    }

    struct FeeParameters {
        uint128 targetNativePrice;
        uint128 sourceNativePrice;
        uint128 wormholeFeeOnSource;
        uint128 wormholeFeeOnTarget;
        uint256 receiverValueTarget;
    }

    IWormhole relayerWormhole;
    WormholeSimulator relayerWormholeSimulator;
    MockGenericRelayer genericRelayer;
    TestHelpers helpers;
    CoreRelayer utilityCoreRelayer;

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
        new MockGenericRelayer(address(wormhole), address(relayerWormholeSimulator), address(helpers.setUpCoreRelayer(2, wormhole, address(0x1))));

        setUpChains(5);

        utilityCoreRelayer = map[1].coreRelayerFull;

        //
    }

    struct StandardSetupTwoChains {
        uint16 sourceChainId;
        uint16 targetChainId;
        uint16 differentChainId;
        Contracts source;
        Contracts target;
    }

    function standardAssumeAndSetupTwoChains(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        uint32 minTargetGasLimit
    ) public returns (StandardSetupTwoChains memory s) {
        vm.assume(gasParams.evmGasOverhead > 0);
        vm.assume(gasParams.targetGasLimit > 0);
        vm.assume(feeParams.targetNativePrice > 0);
        vm.assume(gasParams.targetGasPrice > 0);
        vm.assume(gasParams.sourceGasPrice > 0);
        vm.assume(feeParams.sourceNativePrice > 0);
        vm.assume(
            feeParams.targetNativePrice
                < (uint256(2) ** 239)
                    / (
                        uint256(1) * gasParams.targetGasPrice
                            * (uint256(0) + gasParams.targetGasLimit + gasParams.evmGasOverhead) + feeParams.wormholeFeeOnTarget
                    )
        );
        vm.assume(
            feeParams.sourceNativePrice
                < (uint256(2) ** 239)
                    / (
                        uint256(1) * gasParams.sourceGasPrice
                            * (uint256(0) + gasParams.targetGasLimit + gasParams.evmGasOverhead) + feeParams.wormholeFeeOnSource
                    )
        );
        vm.assume(gasParams.sourceGasPrice < (uint256(2) ** 239) / feeParams.sourceNativePrice);
        vm.assume(gasParams.targetGasLimit >= minTargetGasLimit);
        vm.assume(
            1
                < (uint256(2) ** 239) / gasParams.targetGasLimit / gasParams.targetGasPrice
                    / (uint256(0) + feeParams.sourceNativePrice / feeParams.targetNativePrice + 2) / gasParams.targetGasLimit
        );
        vm.assume(
            1
                < (uint256(2) ** 239) / gasParams.targetGasLimit / gasParams.sourceGasPrice
                    / (uint256(0) + feeParams.targetNativePrice / feeParams.sourceNativePrice + 2) / gasParams.targetGasLimit
        );
        vm.assume(feeParams.receiverValueTarget < uint256(1) * (uint256(2) ** 239) / feeParams.targetNativePrice);

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
        s.source.relayProvider.updatePrice(s.targetChainId, gasParams.targetGasPrice, feeParams.targetNativePrice);
        s.source.relayProvider.updatePrice(s.sourceChainId, gasParams.sourceGasPrice, feeParams.sourceNativePrice);
        s.target.relayProvider.updatePrice(s.targetChainId, gasParams.targetGasPrice, feeParams.targetNativePrice);
        s.target.relayProvider.updatePrice(s.sourceChainId, gasParams.sourceGasPrice, feeParams.sourceNativePrice);

        s.source.relayProvider.updateDeliverGasOverhead(s.targetChainId, gasParams.evmGasOverhead);
        s.target.relayProvider.updateDeliverGasOverhead(s.sourceChainId, gasParams.evmGasOverhead);

        s.source.wormholeSimulator.setMessageFee(feeParams.wormholeFeeOnSource);
        s.target.wormholeSimulator.setMessageFee(feeParams.wormholeFeeOnTarget);
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
            mapEntry.coreRelayer = helpers.setUpCoreRelayer(i, mapEntry.wormhole, address(mapEntry.relayProvider));
            mapEntry.coreRelayerFull = CoreRelayer(payable(address(mapEntry.coreRelayer)));
            genericRelayer.setWormholeRelayerContract(i, address(mapEntry.coreRelayer));
            mapEntry.integration = new MockRelayerIntegration(address(mapEntry.wormhole), address(mapEntry.coreRelayer));
            mapEntry.relayer = address(uint160(uint256(keccak256(abi.encodePacked(bytes("relayer"), i)))));
            genericRelayer.setProviderDeliveryAddress(i, mapEntry.relayer);
            mapEntry.refundAddress =
                payable(address(uint160(uint256(keccak256(abi.encodePacked(bytes("refundAddress"), i))))));
            mapEntry.rewardAddress =
                payable(address(uint160(uint256(keccak256(abi.encodePacked(bytes("rewardAddress"), i))))));
            mapEntry.chainId = i;
            map[i] = mapEntry;
        }

        uint256 maxBudget = type(uint256).max;
        for (uint16 i = 1; i <= numChains; i++) {
            for (uint16 j = 1; j <= numChains; j++) {
                map[i].relayProvider.updateSupportedChain(j, true);
                map[i].relayProvider.updateAssetConversionBuffer(j, 500, 10000);
                map[i].relayProvider.updateTargetChainAddress(j, bytes32(uint256(uint160(address(map[j].relayProvider)))));
                map[i].relayProvider.updateRewardAddress(map[i].rewardAddress);
                helpers.registerCoreRelayerContract(
                    map[i].coreRelayerFull,
                    map[i].wormhole,
                    i,
                    j,
                    bytes32(uint256(uint160(address(map[j].coreRelayer))))
                );
                map[i].relayProvider.updateMaximumBudget(j, maxBudget);
                map[i].integration.registerEmitter(j, bytes32(uint256(uint160(address(map[j].integration)))));
                Structs.XAddress[] memory addresses = new Structs.XAddress[](1);
                addresses[0] = Structs.XAddress(j, bytes32(uint256(uint160(address(map[j].integration)))));
                map[i].integration.registerEmitters(addresses);
            }
        }
    }

    function getDeliveryVAAHash(Vm.Log[] memory logs) internal pure returns (bytes32 vaaHash) {
        vaaHash = logs[0].data.toBytes32(0);
    }

    function getDeliveryStatus(Vm.Log memory log) internal pure returns (DeliveryStatus status) {
        status = DeliveryStatus(log.data.toUint256(32));
    }

    function getDeliveryStatus() internal returns (DeliveryStatus status) {
        Vm.Log[] memory logs = vm.getRecordedLogs();
        status = getDeliveryStatus(logs[logs.length - 1]);
    }

    function vaaKeyArray(uint16 chainId, uint64 sequence, address emitterAddress)
        internal view
        returns (IWormholeRelayer.VaaKey[] memory vaaKeys)
    {
        vaaKeys = new IWormholeRelayer.VaaKey[](1);
        vaaKeys[0] = IWormholeRelayer.VaaKey(
            IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE,
            chainId,
            map[1].coreRelayer.toWormholeFormat(emitterAddress),
            sequence,
            bytes32(0x0)
        );
    }

    function vaaKeyArray(uint16 chainId, uint64 sequence1, address emitterAddress1, uint64 sequence2, address emitterAddress2)
        internal view
        returns (IWormholeRelayer.VaaKey[] memory vaaKeys)
    {
        vaaKeys = new IWormholeRelayer.VaaKey[](2);
        vaaKeys[0] = IWormholeRelayer.VaaKey(
            IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE,
            chainId,
            map[1].coreRelayer.toWormholeFormat(emitterAddress1),
            sequence1,
            bytes32(0x0)
        );
        vaaKeys[1] = IWormholeRelayer.VaaKey(
            IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE,
            chainId,
            map[1].coreRelayer.toWormholeFormat(emitterAddress2),
            sequence2,
            bytes32(0x0)
        );
    }

    function testSend(GasParameters memory gasParams, FeeParameters memory feeParams, bytes memory message) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        // estimate the cost based on the intialized values
        uint256 maxTransactionFee = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        setup.source.integration.sendMessageWithRefundAddress{
            value: maxTransactionFee + uint256(3) * setup.source.wormhole.messageFee()
        }(message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress), bytes(""));

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    function testMultipleForwards(GasParameters memory gasParams, FeeParameters memory feeParams, bytes memory message) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 2000000);

        uint256 payment = assumeAndGetForwardPayment(gasParams.targetGasLimit, 500000, setup, gasParams, feeParams) * 2;

        uint16[] memory chains = new uint16[](2);
        bytes[] memory newMessages = new bytes[](2);
        uint32[] memory gasLimits = new uint32[](2);
        newMessages[0] = message;
        newMessages[1] = message;
        chains[0] = setup.sourceChainId;
        chains[1] = setup.targetChainId;
        gasLimits[0] = 500000;
        gasLimits[1] = 500000;

        MockRelayerIntegration.FurtherInstructions memory furtherInstructions =   MockRelayerIntegration.FurtherInstructions({
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

    function testFundsCorrectForASend(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        uint256 refundAddressBalance = setup.target.refundAddress.balance;
        uint256 relayerBalance = setup.target.relayer.balance;
        uint256 rewardAddressBalance = setup.source.rewardAddress.balance;
        uint256 receiverValueSource = setup.source.coreRelayer.quoteReceiverValue(
            setup.targetChainId, feeParams.receiverValueTarget, address(setup.source.relayProvider)
        );
        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
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

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        uint256 USDcost = (uint256(payment) - uint256(3) * map[setup.sourceChainId].wormhole.messageFee())
            * feeParams.sourceNativePrice
            - (setup.target.refundAddress.balance - refundAddressBalance) * feeParams.targetNativePrice;
        uint256 relayerProfit = uint256(feeParams.sourceNativePrice)
            * (setup.source.rewardAddress.balance - rewardAddressBalance)
            - feeParams.targetNativePrice * (relayerBalance - setup.target.relayer.balance);

        uint256 howMuchGasRelayerCouldHavePaidForAndStillProfited =
            relayerProfit / gasParams.targetGasPrice / feeParams.targetNativePrice;
        assertTrue(howMuchGasRelayerCouldHavePaidForAndStillProfited >= 30000); // takes around this much gas (seems to go from 36k-200k?!?)
        assertTrue(
            USDcost - (relayerProfit + (uint256(1) * feeParams.receiverValueTarget * feeParams.targetNativePrice)) >= 0,
            "We paid enough"
        );
        assertTrue(
            USDcost - (relayerProfit + (uint256(1) * feeParams.receiverValueTarget * feeParams.targetNativePrice))
                < feeParams.sourceNativePrice,
            "We paid the least amount necessary"
        );
    }

    function testFundsCorrectForASendCrossChainRefund(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

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

        uint256 USDcost = (uint256(payment) - uint256(3) * map[setup.sourceChainId].wormhole.messageFee())
            * feeParams.sourceNativePrice
            - (setup.source.refundAddress.balance - refundAddressBalance) * feeParams.sourceNativePrice;

        uint256 relayerProfit = uint256(feeParams.sourceNativePrice)
            * (setup.source.rewardAddress.balance - rewardAddressBalance)
            - feeParams.targetNativePrice * (relayerBalance - setup.target.relayer.balance);
        uint256 refundRelayerProfit = uint256(feeParams.targetNativePrice)
            * (setup.target.rewardAddress.balance - refundRewardAddressBalance)
            - feeParams.sourceNativePrice * (refundRelayerBalance - setup.source.relayer.balance);

        if(refundRelayerProfit > 0) {
            USDcost -= map[setup.targetChainId].wormhole.messageFee() * feeParams.targetNativePrice;
        }

        if(refundRelayerProfit > 0) {
            assertTrue(setup.source.refundAddress.balance > refundAddressBalance, "The cross chain refund went through");
        }
        assertTrue(USDcost - (relayerProfit + refundRelayerProfit) >= 0, "We paid enough");
        assertTrue(
            USDcost - (relayerProfit + refundRelayerProfit) < uint256(0) + feeParams.targetNativePrice + feeParams.sourceNativePrice,
            "We paid the least amount necessary"
        );
    }

    function testFundsCorrectForASendIfReceiveWormholeMessagesReverts(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

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

        uint256 USDcost = uint256(payment - uint256(3) * map[setup.sourceChainId].wormhole.messageFee())
            * feeParams.sourceNativePrice
            - (setup.target.refundAddress.balance - refundAddressBalance) * feeParams.targetNativePrice;
        uint256 relayerProfit = uint256(feeParams.sourceNativePrice)
            * (setup.source.rewardAddress.balance - rewardAddressBalance)
            - feeParams.targetNativePrice * (relayerBalance - setup.target.relayer.balance);
        console.log(USDcost);
        console.log(relayerProfit);
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

        vm.assume(uint256(1) * feeParams.targetNativePrice > uint256(1) * feeParams.sourceNativePrice);

        vm.assume(
            setup.source.coreRelayer.quoteGas(setup.targetChainId, gasFirst, address(setup.source.relayProvider))
                < uint256(2) ** 221
        );
        vm.assume(
            setup.target.coreRelayer.quoteGas(setup.sourceChainId, gasSecond, address(setup.target.relayProvider))
                < uint256(2) ** 221 / feeParams.targetNativePrice
        );

        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasFirst, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        uint256 payment2 = (
            setup.target.coreRelayer.quoteGas(setup.sourceChainId, gasSecond, address(setup.target.relayProvider))
                + uint256(2) * setup.target.wormhole.messageFee()
        ) * feeParams.targetNativePrice / feeParams.sourceNativePrice + 1;

        vm.assume((payment + payment2) < (uint256(2) ** 222));

        return (payment + payment2 * 105 / 100 + 1);
    }

    function testForward(GasParameters memory gasParams, FeeParameters memory feeParams, bytes memory message) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        uint256 payment = assumeAndGetForwardPayment(gasParams.targetGasLimit, 500000, setup, gasParams, feeParams);

        vm.recordLogs();

        setup.source.integration.sendMessageWithForwardedResponse{value: payment}(
            message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress)
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        genericRelayer.relay(setup.targetChainId);

        assertTrue(keccak256(setup.source.integration.getMessage()) == keccak256(bytes("received!")));
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

    function testForwardRequestFail(GasParameters memory gasParams, FeeParameters memory feeParams) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
        ForwardRequestFailStack memory stack;
        vm.assume(
            uint256(1) * feeParams.targetNativePrice * gasParams.targetGasPrice * 10
                < uint256(1) * feeParams.sourceNativePrice * gasParams.sourceGasPrice
        );
        stack.payment =
            setup.source.coreRelayer.quoteGas(setup.targetChainId, 1000000, address(setup.source.relayProvider));

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
        MockRelayerIntegration.FurtherInstructions memory instructions = MockRelayerIntegration.FurtherInstructions({
            keepSending: true,
            newMessages: stack.newMessages,
            chains: stack.chains,
            gasLimits: stack.gasLimits
        });
        stack.encodedFurtherInstructions = setup.source.integration.encodeFurtherInstructions(instructions);
        vm.prank(address(setup.source.integration));
        stack.sequence2 = wormhole.publishMessage{value: stack.wormholeFee}(0, stack.encodedFurtherInstructions, 200);
        stack.targetAddress = setup.source.coreRelayer.toWormholeFormat(address(setup.target.integration));

        sendHelper(setup, stack);

        genericRelayer.relay(setup.sourceChainId);

        Vm.Log[] memory logs = vm.getRecordedLogs();

        DeliveryStatus status = getDeliveryStatus(logs[logs.length - 1]);

        assertTrue(keccak256(setup.target.integration.getMessage()) != keccak256(bytes("hello!")));

        assertTrue(status == DeliveryStatus.FORWARD_REQUEST_FAILURE);
    }

    function sendHelper(StandardSetupTwoChains memory setup, ForwardRequestFailStack memory stack) public {
        IWormholeRelayer.VaaKey[] memory vaaKeys = vaaKeyArray(
                setup.sourceChainId, stack.sequence1, address(setup.source.integration), stack.sequence2, address(setup.source.integration)
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

    function testAttackForwardRequestCache(GasParameters memory gasParams, FeeParameters memory feeParams) public {
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

        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        // Collected funds from the attack are meant to be sent here.
        address attackerSourceAddress =
            address(uint160(uint256(keccak256(abi.encodePacked(bytes("attackerAddress"), setup.sourceChainId)))));
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
            setup.target.coreRelayer.quoteGas(setup.sourceChainId, 500000, address(setup.target.relayProvider))
                < uint256(2) ** 222 / feeParams.targetNativePrice
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
            setup.source.integration.sendMessage{value: computeBudget + uint256(3) * setup.source.wormhole.messageFee()}(
                attackMsg, setup.targetChainId, address(attackerContract)
            );

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
            }(victimMsg, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress), bytes(""));

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
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
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

        assertTrue(address(setup.target.integration).balance >= oldBalance + feeParams.receiverValueTarget);
    }

    function testQuoteReceiverValueIsNotMoreThanNecessary(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
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

        vm.deal(address(this), payment + newReceiverValueSource - 1);

        setup.source.integration.sendMessageGeneral{value: payment + newReceiverValueSource - 1}(
            message,
            setup.targetChainId,
            address(setup.target.integration),
            setup.targetChainId,
            address(0x0),
            newReceiverValueSource - 1,
            bytes("")
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        assertTrue(address(setup.target.integration).balance < oldBalance + feeParams.receiverValueTarget);
    }

    function testTwoSends(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message,
        bytes memory secondMessage
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        // estimate the cost based on the intialized values
        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + uint256(3) * setup.source.wormhole.messageFee();

        // This avoids payment overflow in the target.
        vm.assume(payment <= type(uint256).max >> 1);

        // start listening to events
        vm.recordLogs();

        setup.source.integration.sendMessageWithRefundAddress{value: payment}(
            message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress), bytes("")
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        vm.getRecordedLogs();

        vm.deal(address(this), payment);

        setup.source.integration.sendMessageWithRefundAddress{value: payment}(
            secondMessage, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress), bytes("")
        );

        genericRelayer.relay(setup.sourceChainId);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(secondMessage));
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

    event Delivery(
        address indexed recipientContract,
        uint16 indexed sourceChain,
        uint64 indexed sequence,
        bytes32 deliveryVaaHash,
        DeliveryStatus status
    );

    enum DeliveryStatus {
        SUCCESS,
        RECEIVER_FAILURE,
        FORWARD_REQUEST_FAILURE,
        FORWARD_REQUEST_SUCCESS
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
        IDelivery.TargetDeliveryParameters package;
        IWormholeRelayerInternalStructs.DeliveryInstruction instruction;
    }

    function prepareDeliveryStack(DeliveryStack memory stack, StandardSetupTwoChains memory setup) internal {
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

        stack.package = IDelivery.TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides:bytes("")
        });

        stack.parsed = relayerWormhole.parseVM(stack.deliveryVM);
        stack.instruction =
            setup.target.coreRelayerFull.decodeDeliveryInstruction(stack.parsed.payload);

        stack.budget = stack.instruction.maximumRefundTarget + stack.instruction.receiverValueTarget;
    }

    function testRevertDeliveryInvalidDeliveryVAA(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            message, setup.targetChainId, address(setup.target.integration), setup.target.refundAddress, bytes("")
        );

        prepareDeliveryStack(stack, setup);

        bytes memory fakeVM = abi.encodePacked(stack.deliveryVM);

        invalidateVM(fakeVM, setup.target.wormholeSimulator);

        stack.deliveryVM = fakeVM;

        stack.package = IDelivery.TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides:bytes("")
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
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            message, setup.targetChainId, address(setup.target.integration), setup.target.refundAddress, bytes("")
        );

        prepareDeliveryStack(stack, setup);

        stack.deliveryVM = stack.encodedVMs[0];

        stack.package = IDelivery.TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides: bytes("")
        });

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("InvalidEmitter()"));
        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);
    }

    function testRevertDeliveryInsufficientRelayerFunds(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            message, setup.targetChainId, address(setup.target.integration), setup.target.refundAddress, bytes("")
        );

        prepareDeliveryStack(stack, setup);

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("InsufficientRelayerFunds()"));
        setup.target.coreRelayerFull.deliver{value: stack.budget - 1}(stack.package);
    }

    function testRevertDeliveryTargetChainIsNotThisChain(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            message, setup.targetChainId, address(setup.target.integration), setup.target.refundAddress, bytes("")
        );

        prepareDeliveryStack(stack, setup);

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("TargetChainIsNotThisChain(uint16)", 2));
        map[setup.differentChainId].coreRelayerFull.deliver{value: stack.budget}(stack.package);
    }

    struct SendStackTooDeep {
        uint256 payment;
        IWormholeRelayer.Send deliveryRequest;
        uint256 deliveryOverhead;
        IWormholeRelayer.Send badSend;
    }

    function testRevertSendMsgValueTooLow(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        uint256 maxTransactionFee = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        uint64 sequence =
            setup.source.wormhole.publishMessage{value: setup.source.wormhole.messageFee()}(1, message, 200);

        bytes memory emptyArray;

        IWormholeRelayer.VaaKey[] memory vaaKeys = vaaKeyArray(setup.sourceChainId, sequence, address(this));

        IWormholeRelayer.Send memory deliveryRequest = IWormholeRelayer.Send({
            targetChain: setup.targetChainId,
            targetAddress: setup.source.coreRelayer.toWormholeFormat(address(setup.target.integration)),
            refundChain: setup.targetChainId,
            refundAddress: setup.source.coreRelayer.toWormholeFormat(address(setup.target.refundAddress)),
            maxTransactionFee: maxTransactionFee,
            receiverValue: 0,
            relayProviderAddress: address(setup.source.relayProvider),
            vaaKeys: vaaKeys, 
            consistencyLevel: 200,
            payload: emptyArray,
            relayParameters: setup.source.coreRelayer.getDefaultRelayParams()
        });

        uint256 wormholeFee = setup.source.wormhole.messageFee();

        vm.expectRevert(abi.encodeWithSignature("MsgValueTooLow()"));
        setup.source.coreRelayer.send{value: maxTransactionFee + wormholeFee - 1}(
            deliveryRequest
        );
    }

    function testRevertSendMaxTransactionFeeNotEnough(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        setup.source.relayProvider.updateDeliverGasOverhead(setup.targetChainId, gasParams.evmGasOverhead);

        uint256 maxTransactionFee = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, 1, setup.source.coreRelayer.getDefaultRelayProvider()
        ) - 1;

        uint256 wormholeFee = setup.source.wormhole.messageFee();

        vm.expectRevert(abi.encodeWithSignature("MaxTransactionFeeNotEnough()"));
        setup.source.integration.sendMessageWithRefundAddress{value: maxTransactionFee + 3 * wormholeFee}(
            message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress), bytes("")
        );
    }

    function testRevertSendFundsTooMuch(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        uint256 maxTransactionFee = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        setup.source.relayProvider.updateMaximumBudget(
            setup.targetChainId, uint256(gasParams.targetGasLimit - 1) * gasParams.targetGasPrice
        );

        uint256 wormholeFee = setup.source.wormhole.messageFee();

        vm.expectRevert(abi.encodeWithSignature("FundsTooMuch()"));
        setup.source.integration.sendMessageWithRefundAddress{
            value: maxTransactionFee * 105 / 100 + 1 + 3 * wormholeFee
        }(message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress), bytes(""));
    }

    ForwardTester forwardTester;

    struct ForwardStack {
        bytes32 targetAddress;
        uint256 payment;
        uint256 wormholeFee;
    }

    function executeForwardTest(
        ForwardTester.Action test,
        DeliveryStatus desiredOutcome,
        StandardSetupTwoChains memory setup,
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) internal {
        ForwardStack memory stack;
        vm.recordLogs();
        forwardTester =
        new ForwardTester(address(setup.target.wormhole), address(setup.target.coreRelayer), address(setup.target.wormholeSimulator));
        vm.deal(address(forwardTester), type(uint256).max / 2);
        stack.targetAddress = setup.source.coreRelayer.toWormholeFormat(address(forwardTester));
        stack.payment = assumeAndGetForwardPayment(gasParams.targetGasLimit, 500000, setup, gasParams, feeParams);
        stack.wormholeFee = setup.source.wormhole.messageFee();
        uint64 sequence =
            setup.source.wormhole.publishMessage{value: stack.wormholeFee}(1, abi.encodePacked(uint8(test)), 200);
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
        DeliveryStatus status = getDeliveryStatus();
        assertTrue(status == desiredOutcome);
    }

    function testForwardTester(GasParameters memory gasParams, FeeParameters memory feeParams) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
        executeForwardTest(
            ForwardTester.Action.WorksCorrectly, DeliveryStatus.FORWARD_REQUEST_SUCCESS, setup, gasParams, feeParams
        );
    }

    function testRevertForwardNoDeliveryInProgress(GasParameters memory gasParams, FeeParameters memory feeParams)
        public
    {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        bytes32 targetAddress = setup.source.coreRelayer.toWormholeFormat(address(forwardTester));

        IWormholeRelayer.VaaKey[] memory msgInfoArray = vaaKeyArray(0, 0, address(this));
        vm.expectRevert(abi.encodeWithSignature("NoDeliveryInProgress()"));
        setup.source.coreRelayer.forward(
            setup.targetChainId, targetAddress, setup.targetChainId,  targetAddress, 0, 0, bytes(""), msgInfoArray, 200
        );
    }

    function testRevertForwardForwardRequestFromWrongAddress(
        GasParameters memory gasParams,
        FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        executeForwardTest(
            ForwardTester.Action.ForwardRequestFromWrongAddress,
            DeliveryStatus.RECEIVER_FAILURE,
            setup,
            gasParams,
            feeParams
        );
    }

    function testRevertDeliveryReentrantCall(GasParameters memory gasParams, FeeParameters memory feeParams) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
        executeForwardTest(
            ForwardTester.Action.ReentrantCall, DeliveryStatus.RECEIVER_FAILURE, setup, gasParams, feeParams
        );
    }

    function testRevertForwardMaxTransactionFeeNotEnough(GasParameters memory gasParams, FeeParameters memory feeParams)
        public
    {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        executeForwardTest(
            ForwardTester.Action.MaxTransactionFeeNotEnough,
            DeliveryStatus.RECEIVER_FAILURE,
            setup,
            gasParams,
            feeParams
        );
    }

    function testRevertForwardFundsTooMuch(GasParameters memory gasParams, FeeParameters memory feeParams) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        setup.target.relayProvider.updateMaximumBudget(
            setup.sourceChainId, uint256(10000 - 1) * gasParams.sourceGasPrice
        );

        executeForwardTest(
            ForwardTester.Action.FundsTooMuch, DeliveryStatus.RECEIVER_FAILURE, setup, gasParams, feeParams
        );
    }

    function testRevertTargetNotSupported(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        // estimate the cost based on the intialized values
        uint256 maxTransactionFee = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        uint256 wormholeFee = setup.source.wormhole.messageFee();

        vm.expectRevert(abi.encodeWithSignature("RelayProviderDoesNotSupportTargetChain()"));
        setup.source.integration.sendMessageWithRefundAddress{value: maxTransactionFee + uint256(3) * wormholeFee}(
            message, 32, address(setup.target.integration), address(setup.target.refundAddress), bytes("")
        );
    }

    function testToAndFromWormholeFormat(bytes32 msg2, address msg1) public {
        assertTrue(map[1].coreRelayer.fromWormholeFormat(msg2) == address(uint160(uint256(msg2))));
        assertTrue(map[1].coreRelayer.toWormholeFormat(msg1) == bytes32(uint256(uint160(msg1))));
        assertTrue(map[1].coreRelayer.fromWormholeFormat(map[1].coreRelayer.toWormholeFormat(msg1)) == msg1);
    }

   

    function testEncodeAndDecodeDeliveryInstruction(
        IWormholeRelayerInternalStructs.ExecutionParameters memory executionParameters,
        bytes memory payload
    ) public {
        IWormholeRelayer.VaaKey[] memory vaaKeys = new IWormholeRelayer.VaaKey[](3);
        vaaKeys[0] =  IWormholeRelayer.VaaKey({infoType: IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE, chainId: 1, emitterAddress: bytes32(""), sequence: 23, vaaHash: bytes32("")});
        vaaKeys[1] = vaaKeys[0];
        vaaKeys[2] = vaaKeys[0];

        IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction = IWormholeRelayerInternalStructs.DeliveryInstruction({
            targetChain: 1,
            targetAddress: bytes32(""),
            refundAddress: bytes32(""),
            refundChain: 2,
            maximumRefundTarget: 123,
            receiverValueTarget: 456,
            sourceRelayProvider: bytes32(""),
            targetRelayProvider: bytes32(""),
            senderAddress: bytes32(""),
            vaaKeys: vaaKeys, 
            consistencyLevel: 200,
            executionParameters: executionParameters,
            payload: payload
        });
     
        IWormholeRelayerInternalStructs.DeliveryInstruction memory newInstruction = utilityCoreRelayer
            .decodeDeliveryInstruction(utilityCoreRelayer.encodeDeliveryInstruction(instruction));

        assertTrue(newInstruction.maximumRefundTarget == instruction.maximumRefundTarget);
        assertTrue(newInstruction.receiverValueTarget == instruction.receiverValueTarget);

        assertTrue(keccak256(newInstruction.payload) == keccak256(instruction.payload));
        
    }

    function testDeliveryData(
        GasParameters memory gasParams, FeeParameters memory feeParams
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        )  + setup.source.wormhole.messageFee();

        uint256 maxTransactionFee = payment - setup.source.wormhole.messageFee();

        bytes memory payload = abi.encodePacked(uint256(6));

        setup.source.integration.sendOnlyPayload{value: payment}(
            payload, setup.targetChainId, address(setup.target.integration)
        );

        genericRelayer.relay(setup.sourceChainId);

         bytes32 deliveryVaaHash = getDeliveryVAAHash(vm.getRecordedLogs());

        IWormholeReceiver.DeliveryData memory deliveryData = setup.target.integration.getDeliveryData();

        uint256 calculatedRefund = 0;
        if(maxTransactionFee > setup.source.relayProvider.quoteDeliveryOverhead(setup.targetChainId)) {
            calculatedRefund = (maxTransactionFee - setup.source.relayProvider.quoteDeliveryOverhead(setup.targetChainId)) * feeParams.sourceNativePrice * 100 / (uint256(feeParams.targetNativePrice) * 105);
        }
        assertTrue(setup.target.coreRelayer.fromWormholeFormat(deliveryData.sourceAddress) == address(setup.source.integration));
        assertTrue(deliveryData.sourceChain == setup.sourceChainId);
        assertTrue(deliveryData.maximumRefund == calculatedRefund);
        assertTrue(deliveryData.deliveryHash == deliveryVaaHash);
        assertTrue(keccak256(deliveryData.payload) == keccak256(payload));
    }

    function testInvalidRemoteRefundDoesNotRevert(
        GasParameters memory gasParams, FeeParameters memory feeParams, bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        setup.target.relayProvider.updateSupportedChain(setup.sourceChainId, false);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        bytes memory payload = abi.encodePacked(uint256(6));

        setup.source.integration.sendMessageGeneral{value: stack.payment}(
            message, setup.targetChainId, address(setup.target.integration), setup.sourceChainId, address(setup.source.integration), 0, payload
        );

        prepareDeliveryStack(stack, setup);

        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    function testEmitRedelivery(GasParameters memory gasParams, FeeParameters memory feeParams) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
        vm.recordLogs();
        DeliveryStack memory stack;

        uint256 maxTransactionSource = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        uint256 receiverValueSource = setup.source.coreRelayer.quoteReceiverValue(
            setup.targetChainId, feeParams.receiverValueTarget, address(setup.source.relayProvider));

        uint256 quote = maxTransactionSource +  receiverValueSource + 1 * setup.source.wormhole.messageFee();

        //The key isn't read, so just instantiate dummy values
        IWormholeRelayer.VaaKey memory junkKey = IWormholeRelayer.VaaKey(
            IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE,
            setup.sourceChainId,
            0x0,
            1,
            bytes32(0x0)
        );

        // console.log("LOGGING");
        // console.log(quote);
        // console.log(maxTransactionSource);
        // console.log(receiverValueSource);
        // console.log(setup.source.wormhole.messageFee());

        setup.source.coreRelayer.resend{value: quote}(
            junkKey,
            maxTransactionSource, //newMaxTransactionFee
            receiverValueSource, //new receiver
            setup.targetChainId,
            address(setup.source.relayProvider)
        );

        stack.entries = vm.getRecordedLogs();

        bytes memory redeliveryVM = relayerWormholeSimulator.fetchSignedMessageFromLogs(
            stack.entries[0], setup.sourceChainId, address(setup.source.coreRelayer)
        );

        IWormhole.VM memory vm = setup.source.wormhole.parseVM(redeliveryVM);
        IWormholeRelayerInternalStructs.RedeliveryInstruction memory ins = decodeRedeliveryInstruction(vm.payload);


        assertTrue(ins.key.chainId == setup.sourceChainId, "VAA key has correct chainID");
        assertTrue(ins.key.infoType == IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE, "VAA key type matches");
        assertTrue(ins.newReceiverValue >= feeParams.receiverValueTarget, "new receiver value greater than the old value");
        assertTrue(ins.sourceRelayProvider == setup.source.coreRelayer.toWormholeFormat(address(setup.source.relayProvider)), "specified relay provider is listed");
        assertTrue(ins.executionParameters.gasLimit >= gasParams.targetGasLimit, "new gaslimit was recorded");

    }


    //TODO put this elsewhere
    function decodeRedeliveryInstruction(bytes memory encoded) public view returns (IWormholeRelayerInternalStructs.RedeliveryInstruction memory output) {
        uint256 index = 0;
        
        encoded.toUint8(index); //not actually on the object
        index += 1;

        (output.key, index) = utilityCoreRelayer.decodeVaaKey(encoded, index);

        output.newMaxRefundTarget = encoded.toUint256(index);
        index+=32;

        output.newReceiverValue = encoded.toUint256(index);
        index+=32;

        output.sourceRelayProvider = encoded.toBytes32(index);
        index+=32;

        output.executionParameters.version = 1;
        index+=1;

        output.executionParameters.gasLimit = encoded.toUint32(index);
        index+=4;
    }

    //TODO put this elsewhere
    function encodeDeliveryOverride(IDelivery.DeliveryOverride memory request) public pure returns (bytes memory encoded){
        encoded = abi.encodePacked(
            uint8(1),
            request.gasLimit,
            request.maximumRefund,
            request.receiverValue,
            request.redeliveryHash);
    }

    function testDeliverWithOverrides(GasParameters memory gasParams, FeeParameters memory feeParams, bytes memory message) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            message, setup.targetChainId, address(setup.target.integration), setup.target.refundAddress, bytes("")
        );

        prepareDeliveryStack(stack, setup);

        IDelivery.DeliveryOverride memory deliveryOverride = IDelivery.DeliveryOverride(
            stack.instruction.executionParameters.gasLimit,
            stack.instruction.maximumRefundTarget,
            stack.instruction.receiverValueTarget,
            stack.deliveryVaaHash //really redeliveryHash
            );

        stack.package = IDelivery.TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides: encodeDeliveryOverride(deliveryOverride)
        });

        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);
        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    function testDeliverWithOverrideRevert(GasParameters memory gasParams, FeeParameters memory feeParams, bytes memory message) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStack memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + 3 * setup.source.wormhole.messageFee();

        setup.source.integration.sendMessageWithRefundAddress{value: stack.payment}(
            message, setup.targetChainId, address(setup.target.integration), setup.target.refundAddress, bytes("")
        );

        prepareDeliveryStack(stack, setup);

        IDelivery.DeliveryOverride memory deliveryOverride = IDelivery.DeliveryOverride(
            stack.instruction.executionParameters.gasLimit,
            stack.instruction.maximumRefundTarget -1,
            stack.instruction.receiverValueTarget,
            stack.deliveryVaaHash //really redeliveryHash
            );

        stack.package = IDelivery.TargetDeliveryParameters({
            encodedVMs: stack.encodedVMs,
            encodedDeliveryVAA: stack.deliveryVM,
            relayerRefundAddress: payable(setup.target.relayer),
            overrides: encodeDeliveryOverride(deliveryOverride)
        });

        vm.expectRevert();
        setup.target.coreRelayerFull.deliver{value: stack.budget}(stack.package);
    }
}
