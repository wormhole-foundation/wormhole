// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {IRelayProvider} from "../contracts/interfaces/IRelayProvider.sol";
import {RelayProvider} from "../contracts/relayProvider/RelayProvider.sol";
import {RelayProviderSetup} from "../contracts/relayProvider/RelayProviderSetup.sol";
import {RelayProviderImplementation} from "../contracts/relayProvider/RelayProviderImplementation.sol";
import {RelayProviderProxy} from "../contracts/relayProvider/RelayProviderProxy.sol";
import {RelayProviderMessages} from "../contracts/relayProvider/RelayProviderMessages.sol";
import {RelayProviderStructs} from "../contracts/relayProvider/RelayProviderStructs.sol";
import {IWormholeRelayer} from "../contracts/interfaces/IWormholeRelayer.sol";
import {CoreRelayer} from "../contracts/coreRelayer/CoreRelayer.sol";
import {CoreRelayerStructs} from "../contracts/coreRelayer/CoreRelayerStructs.sol";
import {CoreRelayerSetup} from "../contracts/coreRelayer/CoreRelayerSetup.sol";
import {CoreRelayerImplementation} from "../contracts/coreRelayer/CoreRelayerImplementation.sol";
import {CoreRelayerProxy} from "../contracts/coreRelayer/CoreRelayerProxy.sol";
import {CoreRelayerMessages} from "../contracts/coreRelayer/CoreRelayerMessages.sol";
import {CoreRelayerStructs} from "../contracts/coreRelayer/CoreRelayerStructs.sol";
import {MockWormhole} from "../contracts/mock/MockWormhole.sol";
import {IWormhole} from "../contracts/interfaces/IWormhole.sol";
import {WormholeSimulator, FakeWormholeSimulator} from "./WormholeSimulator.sol";
import {IWormholeReceiver} from "../contracts/interfaces/IWormholeReceiver.sol";
import {AttackForwardIntegration} from "../contracts/mock/AttackForwardIntegration.sol";
import {MockRelayerIntegration} from "../contracts/mock/MockRelayerIntegration.sol";
import "../contracts/libraries/external/BytesLib.sol";

import "forge-std/Test.sol";
import "forge-std/console.sol";

contract TestCoreRelayer is Test {
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

        setUpChains(5);
    }

    function setUpWormhole(uint16 chainId)
        internal
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

    function setUpRelayProvider(uint16 chainId) internal returns (RelayProvider relayProvider) {
        RelayProviderSetup relayProviderSetup = new RelayProviderSetup();
        RelayProviderImplementation relayProviderImplementation = new RelayProviderImplementation();
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
        internal
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

    struct StandardSetupTwoChains {
        uint16 sourceChainId;
        uint16 targetChainId;
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
                < (uint256(2) ** 255)
                    / (
                        uint256(1) * gasParams.targetGasPrice
                            * (uint256(0) + gasParams.targetGasLimit + gasParams.evmGasOverhead) + feeParams.wormholeFeeOnTarget
                    )
        );
        vm.assume(
            feeParams.sourceNativePrice
                < (uint256(2) ** 255)
                    / (
                        uint256(1) * gasParams.sourceGasPrice
                            * (uint256(0) + gasParams.targetGasLimit + gasParams.evmGasOverhead) + feeParams.wormholeFeeOnSource
                    )
        );
        vm.assume(gasParams.sourceGasPrice < (uint256(2) ** 255) / feeParams.sourceNativePrice);
        vm.assume(gasParams.targetGasLimit >= minTargetGasLimit);
        vm.assume(
            1
                < (uint256(2) ** 255) / gasParams.targetGasLimit / gasParams.targetGasPrice
                    / (uint256(0) + feeParams.sourceNativePrice / feeParams.targetNativePrice + 2) / gasParams.targetGasLimit
        );
        vm.assume(
            1
                < (uint256(2) ** 255) / gasParams.targetGasLimit / gasParams.sourceGasPrice
                    / (uint256(0) + feeParams.targetNativePrice / feeParams.sourceNativePrice + 2) / gasParams.targetGasLimit
        );
        vm.assume(feeParams.receiverValueTarget < uint256(1) * (uint256(2) ** 239) / feeParams.targetNativePrice);

        s.sourceChainId = 1;
        s.targetChainId = 2;
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
        uint32 wormholeFeeOnTargetInSourceCurrency = uint32(
            feeParams.wormholeFeeOnTarget * s.source.relayProvider.quoteAssetPrice(s.targetChainId)
                / s.source.relayProvider.quoteAssetPrice(s.sourceChainId) + 1
        );
        s.source.relayProvider.updateWormholeFee(s.targetChainId, wormholeFeeOnTargetInSourceCurrency);
        uint32 wormholeFeeOnSourceInTargetCurrency = uint32(
            feeParams.wormholeFeeOnSource * s.target.relayProvider.quoteAssetPrice(s.sourceChainId)
                / s.target.relayProvider.quoteAssetPrice(s.targetChainId) + 1
        );
        s.target.relayProvider.updateWormholeFee(s.sourceChainId, wormholeFeeOnSourceInTargetCurrency);
    }

    /**
     * SENDING TESTS
     */

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
            (mapEntry.wormhole, mapEntry.wormholeSimulator) = setUpWormhole(i);
            mapEntry.relayProvider = setUpRelayProvider(i);
            mapEntry.coreRelayer = setUpCoreRelayer(i, mapEntry.wormhole, address(mapEntry.relayProvider));
            mapEntry.coreRelayerFull = CoreRelayer(address(mapEntry.coreRelayer));
            mapEntry.integration = new MockRelayerIntegration(address(mapEntry.wormhole), address(mapEntry.coreRelayer));
            mapEntry.relayer = address(uint160(uint256(keccak256(abi.encodePacked(bytes("relayer"), i)))));
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
                map[i].relayProvider.updateDeliveryAddress(j, bytes32(uint256(uint160(map[j].relayer))));
                map[i].relayProvider.updateAssetConversionBuffer(j, 500, 10000);
                map[i].relayProvider.updateRewardAddress(map[i].rewardAddress);
                registerCoreRelayerContract(
                    map[i].coreRelayerFull, i, j, bytes32(uint256(uint160(address(map[j].coreRelayer))))
                );
                map[i].relayProvider.updateMaximumBudget(j, maxBudget);
                map[i].integration.registerEmitter(j, bytes32(uint256(uint160(address(map[j].integration)))));
            }
        }
    }

    function registerCoreRelayerContract(
        CoreRelayer governance,
        uint16 currentChainId,
        uint16 chainId,
        bytes32 coreRelayerContractAddress
    ) internal {
        bytes32 coreRelayerModule = 0x000000000000000000000000000000000000000000436f726552656c61796572;
        bytes memory message =
            abi.encodePacked(coreRelayerModule, uint8(2), currentChainId, chainId, coreRelayerContractAddress);
        IWormhole.VM memory preSignedMessage = IWormhole.VM({
            version: 1,
            timestamp: uint32(block.timestamp),
            nonce: 0,
            emitterChainId: relayerWormhole.governanceChainId(),
            emitterAddress: relayerWormhole.governanceContract(),
            sequence: 0,
            consistencyLevel: 200,
            payload: message,
            guardianSetIndex: 0,
            signatures: new IWormhole.Signature[](0),
            hash: bytes32("")
        });

        bytes memory signed = relayerWormholeSimulator.encodeAndSignMessage(preSignedMessage);
        governance.registerCoreRelayerContract(signed);
    }

    function within(uint256 a, uint256 b, uint256 c) internal view returns (bool) {
        return (a / b <= c && b / a <= c);
    }

    function toWormholeFormat(address addr) public pure returns (bytes32 whFormat) {
        return bytes32(uint256(uint160(addr)));
    }

    function fromWormholeFormat(bytes32 whFormatAddress) public pure returns (address addr) {
        return address(uint160(uint256(whFormatAddress)));
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
        }(message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress));

        genericRelayer(setup.sourceChainId, 3);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    function testFundsCorrect(GasParameters memory gasParams, FeeParameters memory feeParams, bytes memory message)
        public
    {
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
            address(setup.target.refundAddress),
            receiverValueSource,
            1
        );

        genericRelayer(setup.sourceChainId, 3);

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

    function testFundsCorrectIfApplicationCallReverts(
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
            address(setup.target.refundAddress),
            receiverValueSource,
            1
        );

        genericRelayer(setup.sourceChainId, 3);

        uint256 USDcost = uint256(payment - uint256(3) * map[setup.sourceChainId].wormhole.messageFee())
            * feeParams.sourceNativePrice
            - (setup.target.refundAddress.balance - refundAddressBalance) * feeParams.targetNativePrice;
        uint256 relayerProfit = uint256(feeParams.sourceNativePrice)
            * (setup.source.rewardAddress.balance - rewardAddressBalance)
            - feeParams.targetNativePrice * (relayerBalance - setup.target.relayer.balance);
        assertTrue(USDcost == relayerProfit, "We paid the exact amount");
    }

    function testForward(GasParameters memory gasParams, FeeParameters memory feeParams, bytes memory message) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.assume(
            uint256(1) * gasParams.targetGasPrice * feeParams.targetNativePrice
                > uint256(1) * gasParams.sourceGasPrice * feeParams.sourceNativePrice
        );

        vm.recordLogs();
        vm.assume(
            setup.source.coreRelayer.quoteGas(
                setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
            ) < uint256(2) ** 221
        );
        vm.assume(
            setup.target.coreRelayer.quoteGas(setup.sourceChainId, 500000, address(setup.target.relayProvider))
                < uint256(2) ** 221 / feeParams.targetNativePrice
        );
        // estimate the cost based on the intialized values
        uint256 payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + uint256(3) * setup.source.wormhole.messageFee();

        uint256 payment2 = (
            setup.target.coreRelayer.quoteGas(setup.sourceChainId, 500000, address(setup.target.relayProvider))
                + uint256(2) * setup.target.wormhole.messageFee()
        ) * feeParams.targetNativePrice / feeParams.sourceNativePrice + 1;

        vm.assume((payment + payment2) < (uint256(2) ** 222));

        setup.source.integration.sendMessageWithForwardedResponse{value: payment + payment2}(
            message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress)
        );

        genericRelayer(setup.sourceChainId, 3);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        genericRelayer(setup.targetChainId, 3);

        assertTrue(keccak256(setup.source.integration.getMessage()) == keccak256(bytes("received!")));
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
            genericRelayer(setup.sourceChainId, 3);

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
            }(victimMsg, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress));

            // The relayer delivers the victim's message.
            // During the delivery process, the forward request injected by the malicious contract is acknowledged.
            // The victim's refund address is not called due to this.
            genericRelayer(setup.sourceChainId, 3);

            // Ensures the message was received.
            assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(victimMsg));
            // Here we assert that the victim's refund is safe.
            assertTrue(victimBalancePreDelivery < setup.target.refundAddress.balance);
        }

        Vm.Log[] memory entries = relayerWormholeSimulator.fetchWormholeMessageFromLog(vm.getRecordedLogs());
        if (entries.length > 0) {
            // There was a wormhole message produced.
            // If the attack is successful this is a forward.
            // We'll invoke the relay simulation here and later assert that the attack wasn't successful.
            // Relay from target chain to source chain.
            genericRelayerProcessLogs(setup.targetChainId, entries);
        }
        // Assert that the attack wasn't successful.
        assertTrue(attackerSourceAddress.balance == 0);
    }

    function testRedelivery(GasParameters memory gasParams, FeeParameters memory feeParams, bytes memory message)
        public
    {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        // estimate the cost based on the intialized values
        uint256 payment = setup.source.coreRelayer.quoteGasResend(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + setup.source.wormhole.messageFee();
        uint256 paymentNotEnough = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, 10, address(setup.source.relayProvider)
        ) + setup.source.wormhole.messageFee() * 3;

        uint256 oldBalance = address(setup.target.integration).balance;

        vm.deal(address(this), paymentNotEnough);
        setup.source.integration.sendMessageWithRefundAddress{value: paymentNotEnough}(
            message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress)
        );

        genericRelayer(setup.sourceChainId, 3);

        assertTrue(
            (keccak256(setup.target.integration.getMessage()) != keccak256(message))
                || (keccak256(message) == keccak256(bytes("")))
        );
        Vm.Log[] memory logs = vm.getRecordedLogs();
        bytes32 deliveryVaaHash = logs[0].data.toBytes32(0);

        uint256 newMaxTransactionFeeFee = setup.source.coreRelayer.quoteReceiverValue(
            setup.targetChainId, feeParams.receiverValueTarget, address(setup.source.relayProvider)
        );

        IWormholeRelayer.ResendByTx memory redeliveryRequest = IWormholeRelayer.ResendByTx({
            sourceChain: setup.sourceChainId,
            sourceTxHash: deliveryVaaHash,
            sourceNonce: 1,
            targetChain: setup.targetChainId,
            deliveryIndex: uint8(2),
            multisendIndex: uint8(0),
            newMaxTransactionFee: payment - setup.source.wormhole.messageFee(),
            newReceiverValue: newMaxTransactionFeeFee,
            newRelayParameters: setup.source.coreRelayer.getDefaultRelayParams()
        });

        vm.deal(address(this), payment + newMaxTransactionFeeFee);

        setup.source.coreRelayer.resend{value: payment + newMaxTransactionFeeFee}(
            redeliveryRequest, 1, address(setup.source.relayProvider)
        );

        genericRelayer(setup.sourceChainId, 1);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    function testApplicationBudgetFeeWithRedelivery(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);
        vm.assume(feeParams.receiverValueTarget > 0);
        vm.recordLogs();

        // estimate the cost based on the intialized values
        uint256 payment = setup.source.coreRelayer.quoteGasResend(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + setup.source.wormhole.messageFee();
        uint256 paymentNotEnough = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, 10, address(setup.source.relayProvider)
        ) + setup.source.wormhole.messageFee() * 3;

        uint256 oldBalance = address(setup.target.integration).balance;

        setup.source.integration.sendMessageWithRefundAddress{value: paymentNotEnough}(
            message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress)
        );

        genericRelayer(setup.sourceChainId, 3);

        assertTrue(
            (keccak256(setup.target.integration.getMessage()) != keccak256(message))
                || (keccak256(message) == keccak256(bytes("")))
        );

        bytes32 deliveryVaaHash = vm.getRecordedLogs()[0].data.toBytes32(0);

        uint256 newMaxTransactionFeeFee = setup.source.coreRelayer.quoteReceiverValue(
            setup.targetChainId, feeParams.receiverValueTarget, address(setup.source.relayProvider)
        );

        IWormholeRelayer.ResendByTx memory redeliveryRequest = IWormholeRelayer.ResendByTx({
            sourceChain: setup.sourceChainId,
            sourceTxHash: deliveryVaaHash,
            sourceNonce: 1,
            targetChain: setup.targetChainId,
            deliveryIndex: 2,
            multisendIndex: 0,
            newMaxTransactionFee: payment - setup.source.wormhole.messageFee(),
            newReceiverValue: newMaxTransactionFeeFee,
            newRelayParameters: setup.source.coreRelayer.getDefaultRelayParams()
        });

        vm.deal(address(this), payment + newMaxTransactionFeeFee);

        setup.source.coreRelayer.resend{value: payment + newMaxTransactionFeeFee}(
            redeliveryRequest, 1, address(setup.source.relayProvider)
        );

        genericRelayer(setup.sourceChainId, 1);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        assertTrue(address(setup.target.integration).balance >= oldBalance + feeParams.receiverValueTarget);

        oldBalance = address(setup.target.integration).balance;
        vm.getRecordedLogs();
        redeliveryRequest = IWormholeRelayer.ResendByTx({
            sourceChain: setup.sourceChainId,
            sourceTxHash: deliveryVaaHash,
            sourceNonce: 1,
            targetChain: setup.targetChainId,
            deliveryIndex: 2,
            multisendIndex: 0,
            newMaxTransactionFee: payment - setup.source.wormhole.messageFee(),
            newReceiverValue: newMaxTransactionFeeFee - 1,
            newRelayParameters: setup.source.coreRelayer.getDefaultRelayParams()
        });

        vm.deal(address(this), payment + newMaxTransactionFeeFee - 1);

        setup.source.coreRelayer.resend{value: payment + newMaxTransactionFeeFee - 1}(
            redeliveryRequest, 1, address(setup.source.relayProvider)
        );

        genericRelayer(setup.sourceChainId, 1);

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
            message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress)
        );

        genericRelayer(setup.sourceChainId, 3);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));

        vm.getRecordedLogs();

        vm.deal(address(this), payment);

        setup.source.integration.sendMessageWithRefundAddress{value: payment}(
            secondMessage, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress)
        );

        genericRelayer(setup.sourceChainId, 3);

        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(secondMessage));
    }

    function testRevertNonceZero(GasParameters memory gasParams, FeeParameters memory feeParams, bytes memory message)
        public
    {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        uint256 wormholeFee = setup.source.wormhole.messageFee();
        // estimate the cost based on the intialized values
        uint256 maxTransactionFee = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        );

        vm.expectRevert(abi.encodeWithSignature("NonceIsZero()"));

        setup.source.integration.sendMessageGeneral{value: maxTransactionFee + 3 * wormholeFee}(
            message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress), 0, 0
        );
    }

    /**
     * Forwarding tests 2, 3-7.. need to think about how to test this.. some sort of way to control the forwarding request? Or maybe make a different relayerintegration for testing?
     */

    /**
     * Reentrancy test for execute delivery 8
     */

    /**
     * Redelivery  9-17
     */
    struct RedeliveryStackTooDeep {
        bytes32 deliveryVaaHash;
        uint256 payment;
        Vm.Log[] entries;
        bytes redeliveryVM;
        IWormhole.VM parsed;
        uint256 budget;
        IWormholeRelayer.ResendByTx redeliveryRequest;
        CoreRelayer.TargetDeliveryParametersSingle originalDelivery;
        CoreRelayerStructs.TargetRedeliveryByTxHashParamsSingle package;
        CoreRelayer.RedeliveryByTxHashInstruction instruction;
    }

    function invalidateVM(bytes memory message, WormholeSimulator simulator) internal {
        change(message, message.length - 1);
        simulator.invalidateVM(message);
    }

    function change(bytes memory message, uint256 index) internal {
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
        FORWARD_REQUEST_SUCCESS,
        INVALID_REDELIVERY
    }

    function testRevertRedeliveryErrors(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        RedeliveryStackTooDeep memory stack;

        setup.source.integration.sendMessageWithRefundAddress{
            value: setup.source.coreRelayer.quoteGas(setup.targetChainId, 21000, address(setup.source.relayProvider))
                + uint256(3) * setup.source.wormhole.messageFee()
        }(message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress));

        genericRelayer(setup.sourceChainId, 3);

        assertTrue(
            (keccak256(setup.target.integration.getMessage()) != keccak256(message))
                || (keccak256(message) == keccak256(bytes("")))
        );

        stack.deliveryVaaHash = vm.getRecordedLogs()[0].data.toBytes32(0);

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + setup.source.wormhole.messageFee();

        stack.redeliveryRequest = IWormholeRelayer.ResendByTx({
            sourceChain: setup.sourceChainId,
            sourceTxHash: stack.deliveryVaaHash,
            sourceNonce: 1,
            targetChain: setup.targetChainId,
            deliveryIndex: 2,
            multisendIndex: 0,
            newMaxTransactionFee: stack.payment - setup.source.wormhole.messageFee(),
            newReceiverValue: 0,
            newRelayParameters: setup.source.coreRelayer.getDefaultRelayParams()
        });
        vm.deal(address(this), stack.payment);
        vm.expectRevert(abi.encodeWithSignature("MsgValueTooLow()"));
        setup.source.coreRelayer.resend{value: stack.payment - 1}(
            stack.redeliveryRequest, 1, address(setup.source.relayProvider)
        );

        setup.source.coreRelayer.resend{value: stack.payment}(
            stack.redeliveryRequest, 1, address(setup.source.relayProvider)
        );

        stack.entries = vm.getRecordedLogs();

        stack.redeliveryVM = relayerWormholeSimulator.fetchSignedMessageFromLogs(
            stack.entries[0], setup.sourceChainId, address(setup.source.coreRelayer)
        );

        stack.originalDelivery = pastDeliveries[keccak256(abi.encodePacked(stack.deliveryVaaHash, uint8(0)))];

        bytes memory fakeVM = abi.encodePacked(stack.originalDelivery.encodedVMs[2]);
        bytes memory correctVM = abi.encodePacked(stack.originalDelivery.encodedVMs[2]);
        invalidateVM(fakeVM, setup.target.wormholeSimulator);
        stack.originalDelivery.encodedVMs[2] = fakeVM;

        stack.package = CoreRelayerStructs.TargetRedeliveryByTxHashParamsSingle(
            stack.redeliveryVM, stack.originalDelivery.encodedVMs, payable(setup.target.relayer)
        );

        stack.parsed = relayerWormhole.parseVM(stack.redeliveryVM);
        stack.instruction = setup.target.coreRelayerFull.getRedeliveryByTxHashInstruction(stack.parsed.payload);

        stack.budget = stack.instruction.newMaximumRefundTarget + stack.instruction.newReceiverValueTarget
            + setup.target.wormhole.messageFee();

        vm.deal(setup.target.relayer, stack.budget);

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("InvalidVaa(uint8)", 2));
        setup.target.coreRelayerFull.redeliverSingle{value: stack.budget}(stack.package);

        stack.originalDelivery.encodedVMs[2] = stack.originalDelivery.encodedVMs[0];

        stack.package = CoreRelayerStructs.TargetRedeliveryByTxHashParamsSingle(
            stack.redeliveryVM, stack.originalDelivery.encodedVMs, payable(setup.target.relayer)
        );

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("InvalidEmitterInOriginalDeliveryVM(uint8)", 2));
        setup.target.coreRelayerFull.redeliverSingle{value: stack.budget}(stack.package);

        stack.originalDelivery.encodedVMs[2] = correctVM;

        stack.package = CoreRelayerStructs.TargetRedeliveryByTxHashParamsSingle(
            stack.redeliveryVM, stack.originalDelivery.encodedVMs, payable(setup.target.relayer)
        );

        correctVM = abi.encodePacked(stack.redeliveryVM);
        fakeVM = abi.encodePacked(correctVM);
        invalidateVM(fakeVM, setup.target.wormholeSimulator);

        stack.package = CoreRelayerStructs.TargetRedeliveryByTxHashParamsSingle(
            fakeVM, stack.originalDelivery.encodedVMs, payable(setup.target.relayer)
        );

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("InvalidRedeliveryVM(string)", ""));
        setup.target.coreRelayerFull.redeliverSingle{value: stack.budget}(stack.package);

        fakeVM = relayerWormholeSimulator.fetchSignedMessageFromLogs(
            stack.entries[0], setup.sourceChainId, address(setup.source.integration)
        );
        stack.package = CoreRelayerStructs.TargetRedeliveryByTxHashParamsSingle(
            fakeVM, stack.originalDelivery.encodedVMs, payable(setup.target.relayer)
        );

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("InvalidEmitterInRedeliveryVM()"));
        setup.target.coreRelayerFull.redeliverSingle{value: stack.budget}(stack.package);

        setup.source.relayProvider.updateDeliveryAddress(setup.targetChainId, bytes32(uint256(uint160(address(this)))));
        vm.deal(address(this), stack.payment);
        vm.getRecordedLogs();
        setup.source.coreRelayer.resend{value: stack.payment}(
            stack.redeliveryRequest, 1, address(setup.source.relayProvider)
        );
        stack.entries = vm.getRecordedLogs();
        setup.source.relayProvider.updateDeliveryAddress(
            setup.targetChainId, bytes32(uint256(uint160(address(setup.target.relayer))))
        );

        fakeVM = relayerWormholeSimulator.fetchSignedMessageFromLogs(
            stack.entries[0], setup.sourceChainId, address(setup.source.coreRelayer)
        );
        stack.package = CoreRelayerStructs.TargetRedeliveryByTxHashParamsSingle(
            fakeVM, stack.originalDelivery.encodedVMs, payable(setup.target.relayer)
        );

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("MismatchingRelayProvidersInRedelivery()"));
        setup.target.coreRelayerFull.redeliverSingle{value: stack.budget}(stack.package);

        stack.package = CoreRelayerStructs.TargetRedeliveryByTxHashParamsSingle(
            stack.redeliveryVM, stack.originalDelivery.encodedVMs, payable(msg.sender)
        );

        vm.deal(address(this), stack.budget);
        bytes32 redeliveryVmHash = relayerWormhole.parseVM(stack.redeliveryVM).hash;
        vm.expectEmit(true, true, true, true, address(setup.target.coreRelayer));
        emit Delivery({
            recipientContract: address(setup.target.integration),
            sourceChain: setup.sourceChainId,
            sequence: 1,
            deliveryVaaHash: redeliveryVmHash,
            status: DeliveryStatus.INVALID_REDELIVERY
        });
        setup.target.coreRelayerFull.redeliverSingle{value: stack.budget}(stack.package);

        uint16 differentChainId = 2;
        if (setup.targetChainId == 2) {
            differentChainId = 3;
        }
        vm.deal(setup.target.relayer, stack.budget);
        vm.expectEmit(true, true, true, true, address(map[differentChainId].coreRelayer));
        emit Delivery({
            recipientContract: address(setup.target.integration),
            sourceChain: setup.sourceChainId,
            sequence: 1,
            deliveryVaaHash: redeliveryVmHash,
            status: DeliveryStatus.INVALID_REDELIVERY
        });
        vm.prank(setup.target.relayer);
        map[differentChainId].coreRelayerFull.redeliverSingle{value: stack.budget}(stack.package);

        stack.redeliveryRequest = IWormholeRelayer.ResendByTx({
            sourceChain: setup.sourceChainId,
            sourceTxHash: stack.deliveryVaaHash,
            sourceNonce: 1,
            targetChain: differentChainId,
            deliveryIndex: stack.originalDelivery.deliveryIndex,
            multisendIndex: stack.originalDelivery.multisendIndex,
            newMaxTransactionFee: stack.payment - setup.source.wormhole.messageFee(),
            newReceiverValue: 0,
            newRelayParameters: setup.source.coreRelayer.getDefaultRelayParams()
        });
        setup.source.relayProvider.updatePrice(differentChainId, gasParams.targetGasPrice, feeParams.targetNativePrice);
        setup.source.relayProvider.updatePrice(differentChainId, gasParams.sourceGasPrice, feeParams.sourceNativePrice);
        setup.source.relayProvider.updateDeliveryAddress(
            differentChainId, bytes32(uint256(uint160(address(setup.target.relayer))))
        );
        vm.deal(address(this), stack.payment);
        vm.getRecordedLogs();
        setup.source.coreRelayer.resend{value: stack.payment}(
            stack.redeliveryRequest, 1, address(setup.source.relayProvider)
        );
        stack.entries = vm.getRecordedLogs();
        setup.source.relayProvider.updateDeliveryAddress(
            differentChainId, bytes32(uint256(uint160(address(map[differentChainId].relayer))))
        );

        fakeVM = relayerWormholeSimulator.fetchSignedMessageFromLogs(
            stack.entries[0], setup.sourceChainId, address(setup.source.coreRelayer)
        );
        stack.package = CoreRelayerStructs.TargetRedeliveryByTxHashParamsSingle(
            fakeVM, stack.originalDelivery.encodedVMs, payable(setup.target.relayer)
        );

        redeliveryVmHash = relayerWormhole.parseVM(fakeVM).hash;
        uint256 txValue = stack.payment + map[differentChainId].wormhole.messageFee();
        vm.deal(setup.target.relayer, txValue);

        vm.expectEmit(true, true, true, true, address(map[differentChainId].coreRelayer));
        emit Delivery({
            recipientContract: address(setup.target.integration),
            sourceChain: setup.sourceChainId,
            sequence: 3,
            deliveryVaaHash: redeliveryVmHash,
            status: DeliveryStatus.INVALID_REDELIVERY
        });
        vm.prank(setup.target.relayer);
        map[differentChainId].coreRelayerFull.redeliverSingle{value: txValue}(stack.package);

        stack.package = CoreRelayerStructs.TargetRedeliveryByTxHashParamsSingle({
            redeliveryVM: correctVM,
            sourceEncodedVMs: stack.originalDelivery.encodedVMs,
            relayerRefundAddress: payable(setup.target.relayer)
        });
        vm.deal(setup.target.relayer, stack.budget - 1);
        vm.expectRevert(abi.encodeWithSignature("InsufficientRelayerFunds()"));
        vm.prank(setup.target.relayer);
        setup.target.coreRelayerFull.redeliverSingle{value: stack.budget - 1}(stack.package);

        vm.deal(setup.target.relayer, stack.budget);

        assertTrue(
            (keccak256(setup.target.integration.getMessage()) != keccak256(message))
                || (keccak256(message) == keccak256(bytes("")))
        );
        vm.prank(setup.target.relayer);
        setup.target.coreRelayerFull.redeliverSingle{value: stack.budget}(stack.package);
        assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
    }

    /**
     * Delivery 18-24
     */
    struct DeliveryStackTooDeep {
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
        CoreRelayer.TargetDeliveryParametersSingle package;
        CoreRelayer.DeliveryInstruction instruction;
    }

    function testRevertDeliveryErrors(
        GasParameters memory gasParams,
        FeeParameters memory feeParams,
        bytes memory message
    ) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        DeliveryStackTooDeep memory stack;

        if (
            uint256(1) * feeParams.targetNativePrice * gasParams.targetGasPrice * 10
                < uint256(1) * feeParams.sourceNativePrice * gasParams.sourceGasPrice
        ) {
            stack.paymentNotEnough =
                setup.source.coreRelayer.quoteGas(setup.targetChainId, 600000, address(setup.source.relayProvider));

            setup.source.integration.sendMessageWithForwardedResponse{
                value: stack.paymentNotEnough + 3 * setup.source.wormhole.messageFee()
            }(message, setup.targetChainId, address(setup.target.integration), address(setup.target.refundAddress));

            genericRelayer(setup.sourceChainId, 3);

            assertTrue(keccak256(setup.target.integration.getMessage()) == keccak256(message));
            stack.entries = vm.getRecordedLogs();

            stack.actualVM1 = relayerWormholeSimulator.fetchSignedMessageFromLogs(
                stack.entries[0], setup.targetChainId, address(setup.target.integration)
            );

            stack.actualVM2 = relayerWormholeSimulator.fetchSignedMessageFromLogs(
                stack.entries[1], setup.targetChainId, address(setup.target.integration)
            );

            stack.deliveryVM = relayerWormholeSimulator.fetchSignedMessageFromLogs(
                stack.entries[2], setup.targetChainId, address(setup.target.coreRelayer)
            );

            stack.encodedVMs = new bytes[](3);
            stack.encodedVMs[0] = stack.actualVM1;
            stack.encodedVMs[1] = stack.actualVM2;
            stack.encodedVMs[2] = stack.deliveryVM;

            stack.package = CoreRelayerStructs.TargetDeliveryParametersSingle({
                encodedVMs: stack.encodedVMs,
                deliveryIndex: 2,
                multisendIndex: 0,
                relayerRefundAddress: payable(setup.target.relayer)
            });

            stack.parsed = relayerWormhole.parseVM(stack.deliveryVM);
            stack.instruction =
                setup.target.coreRelayerFull.getDeliveryInstructionsContainer(stack.parsed.payload).instructions[0];
            stack.budget = stack.instruction.maximumRefundTarget + stack.instruction.receiverValueTarget
                + setup.source.wormhole.messageFee();

            vm.deal(setup.source.relayer, stack.budget);
            vm.prank(setup.source.relayer);
            vm.expectRevert(abi.encodeWithSignature("SendNotSufficientlyFunded()"));
            setup.source.coreRelayerFull.deliverSingle{value: stack.budget}(stack.package);
        }

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + setup.source.wormhole.messageFee();

        vm.prank(address(setup.source.integration));
        setup.source.wormhole.publishMessage{value: setup.source.wormhole.messageFee()}(
            1, abi.encodePacked(uint8(0), bytes("hi!")), 200
        );

        vm.prank(address(setup.source.integration));
        setup.source.wormhole.publishMessage{value: setup.source.wormhole.messageFee()}(
            1, abi.encodePacked(uint8(0)), 200
        );

        IWormholeRelayer.Send memory deliveryRequest = IWormholeRelayer.Send({
            targetChain: setup.targetChainId,
            targetAddress: setup.source.coreRelayer.toWormholeFormat(address(setup.target.integration)),
            refundAddress: setup.source.coreRelayer.toWormholeFormat(address(setup.target.refundAddress)),
            maxTransactionFee: stack.payment - setup.source.wormhole.messageFee(),
            receiverValue: 0,
            relayParameters: setup.source.coreRelayer.getDefaultRelayParams()
        });

        setup.source.coreRelayer.send{value: stack.payment}(deliveryRequest, 1, address(setup.source.relayProvider));

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

        bytes memory fakeVM = abi.encodePacked(stack.deliveryVM);

        invalidateVM(fakeVM, setup.target.wormholeSimulator);

        stack.encodedVMs = new bytes[](3);
        stack.encodedVMs[0] = stack.actualVM1;
        stack.encodedVMs[1] = stack.actualVM2;
        stack.encodedVMs[2] = fakeVM;

        stack.package = CoreRelayerStructs.TargetDeliveryParametersSingle({
            encodedVMs: stack.encodedVMs,
            deliveryIndex: 2,
            multisendIndex: 0,
            relayerRefundAddress: payable(setup.target.relayer)
        });

        stack.parsed = relayerWormhole.parseVM(stack.deliveryVM);
        stack.instruction =
            setup.target.coreRelayerFull.getDeliveryInstructionsContainer(stack.parsed.payload).instructions[0];

        stack.budget = stack.instruction.maximumRefundTarget + stack.instruction.receiverValueTarget
            + setup.target.wormhole.messageFee();

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("InvalidVaa(uint8)", 2));
        setup.target.coreRelayerFull.deliverSingle{value: stack.budget}(stack.package);

        stack.encodedVMs[2] = stack.encodedVMs[0];

        stack.package = CoreRelayerStructs.TargetDeliveryParametersSingle({
            encodedVMs: stack.encodedVMs,
            deliveryIndex: 2,
            multisendIndex: 0,
            relayerRefundAddress: payable(setup.target.relayer)
        });

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("InvalidEmitter()"));
        setup.target.coreRelayerFull.deliverSingle{value: stack.budget}(stack.package);

        stack.encodedVMs[2] = stack.deliveryVM;

        stack.package = CoreRelayerStructs.TargetDeliveryParametersSingle({
            encodedVMs: stack.encodedVMs,
            deliveryIndex: 2,
            multisendIndex: 0,
            relayerRefundAddress: payable(setup.target.relayer)
        });

        vm.expectRevert(abi.encodeWithSignature("UnexpectedRelayer()"));
        setup.target.coreRelayerFull.deliverSingle{value: stack.budget}(stack.package);

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("InsufficientRelayerFunds()"));
        setup.target.coreRelayerFull.deliverSingle{value: stack.budget - 1}(stack.package);

        uint16 differentChainId = 2;
        if (setup.targetChainId == 2) {
            differentChainId = 3;
        }

        vm.prank(setup.target.relayer);
        vm.expectRevert(abi.encodeWithSignature("TargetChainIsNotThisChain(uint16)", 2));
        map[differentChainId].coreRelayerFull.deliverSingle{value: stack.budget}(stack.package);

        vm.prank(setup.target.relayer);
        setup.target.coreRelayerFull.deliverSingle{value: stack.budget}(stack.package);
    }

    struct SendStackTooDeep {
        uint256 payment;
        IWormholeRelayer.Send deliveryRequest;
        uint256 deliveryOverhead;
        IWormholeRelayer.Send badSend;
    }
    /**
     * Request delivery 25-27
     */

    function testRevertSendErrors(GasParameters memory gasParams, FeeParameters memory feeParams) public {
        StandardSetupTwoChains memory setup = standardAssumeAndSetupTwoChains(gasParams, feeParams, 1000000);

        vm.recordLogs();

        SendStackTooDeep memory stack;

        stack.payment = setup.source.coreRelayer.quoteGas(
            setup.targetChainId, gasParams.targetGasLimit, address(setup.source.relayProvider)
        ) + setup.source.wormhole.messageFee();

        setup.source.wormhole.publishMessage{value: setup.source.wormhole.messageFee()}(
            1, abi.encodePacked(uint8(0), bytes("hi!")), 200
        );

        stack.deliveryRequest = IWormholeRelayer.Send({
            targetChain: setup.targetChainId,
            targetAddress: setup.source.coreRelayer.toWormholeFormat(address(setup.target.integration)),
            refundAddress: setup.source.coreRelayer.toWormholeFormat(address(setup.target.refundAddress)),
            maxTransactionFee: stack.payment - setup.source.wormhole.messageFee(),
            receiverValue: 0,
            relayParameters: setup.source.coreRelayer.getDefaultRelayParams()
        });

        vm.expectRevert(abi.encodeWithSignature("MsgValueTooLow()"));
        setup.source.coreRelayer.send{value: stack.payment - 1}(
            stack.deliveryRequest, 1, address(setup.source.relayProvider)
        );

        setup.source.relayProvider.updateDeliverGasOverhead(setup.targetChainId, gasParams.evmGasOverhead);

        stack.deliveryOverhead = setup.source.relayProvider.quoteDeliveryOverhead(setup.targetChainId);
        vm.assume(stack.deliveryOverhead > 0);

        stack.badSend = IWormholeRelayer.Send({
            targetChain: setup.targetChainId,
            targetAddress: setup.source.coreRelayer.toWormholeFormat(address(setup.target.integration)),
            refundAddress: setup.source.coreRelayer.toWormholeFormat(address(setup.target.refundAddress)),
            maxTransactionFee: stack.deliveryOverhead - 1,
            receiverValue: 0,
            relayParameters: setup.source.coreRelayer.getDefaultRelayParams()
        });

        vm.expectRevert(abi.encodeWithSignature("MaxTransactionFeeNotEnough()"));
        setup.source.coreRelayer.send{value: stack.deliveryOverhead - 1}(
            stack.badSend, 1, address(setup.source.relayProvider)
        );

        //setup.source.relayProvider.updateDeliverGasOverhead(setup.targetChainId, 0);

        setup.source.relayProvider.updateMaximumBudget(
            setup.targetChainId, uint256(gasParams.targetGasLimit - 1) * gasParams.targetGasPrice
        );

        vm.expectRevert(abi.encodeWithSignature("FundsTooMuch()"));
        setup.source.coreRelayer.send{value: stack.payment}(
            stack.deliveryRequest, 1, address(setup.source.relayProvider)
        );
    }

    /**
     *
     *
     * GENERIC RELAYER CODE
     *
     *
     */

    mapping(uint256 => bool) nonceCompleted;

    mapping(bytes32 => CoreRelayer.TargetDeliveryParametersSingle) pastDeliveries;

    function genericRelayer(uint16 chainId, uint8 num) internal {
        Vm.Log[] memory entries = truncateRecordedLogs(chainId, num);
        genericRelayerProcessLogs(chainId, entries);
    }

    /**
     * Discards wormhole events beyond `num` events.
     * Expects at least `num` wormhole events.
     */
    function truncateRecordedLogs(uint16 chainId, uint8 num) internal returns (Vm.Log[] memory) {
        // Filters all events to just the wormhole messages.
        Vm.Log[] memory entries = relayerWormholeSimulator.fetchWormholeMessageFromLog(vm.getRecordedLogs());
        // We expect at least `num` events.
        assertTrue(entries.length >= num);

        Vm.Log[] memory firstEntries = new Vm.Log[](num);
        for (uint256 i = 0; i < num; i++) {
            firstEntries[i] = entries[i];
        }
        return firstEntries;
    }

    function genericRelayerProcessLogs(uint16 chainId, Vm.Log[] memory entries) internal {
        bytes[] memory encodedVMs = new bytes[](entries.length);
        for (uint256 i = 0; i < encodedVMs.length; i++) {
            encodedVMs[i] = relayerWormholeSimulator.fetchSignedMessageFromLogs(
                entries[i], chainId, address(uint160(uint256(bytes32(entries[i].topics[1]))))
            );
        }
        IWormhole.VM[] memory parsed = new IWormhole.VM[](encodedVMs.length);
        for (uint16 i = 0; i < encodedVMs.length; i++) {
            parsed[i] = relayerWormhole.parseVM(encodedVMs[i]);
        }
        //uint16 chainId = parsed[parsed.length - 1].emitterChainId;
        Contracts memory contracts = map[chainId];

        for (uint16 i = 0; i < encodedVMs.length; i++) {
            if (!nonceCompleted[parsed[i].nonce]) {
                nonceCompleted[parsed[i].nonce] = true;
                uint8 length = 1;
                for (uint16 j = i + 1; j < encodedVMs.length; j++) {
                    if (parsed[i].nonce == parsed[j].nonce) {
                        length++;
                    }
                }
                bytes[] memory encodedVMsToBeDelivered = new bytes[](length);
                uint8 counter = 0;
                for (uint16 j = i; j < encodedVMs.length; j++) {
                    if (parsed[i].nonce == parsed[j].nonce) {
                        encodedVMsToBeDelivered[counter] = encodedVMs[j];
                        counter++;
                    }
                }
                counter = 0;
                for (uint16 j = i; j < encodedVMs.length; j++) {
                    if (parsed[i].nonce == parsed[j].nonce) {
                        if (
                            parsed[j].emitterAddress == toWormholeFormat(address(contracts.coreRelayer))
                                && (parsed[j].emitterChainId == chainId)
                        ) {
                            genericRelay(contracts, counter, encodedVMs[j], encodedVMsToBeDelivered, parsed[j]);
                        }
                        counter += 1;
                    }
                }
            }
        }
        for (uint8 i = 0; i < encodedVMs.length; i++) {
            nonceCompleted[parsed[i].nonce] = false;
        }
    }

    function genericRelay(
        Contracts memory contracts,
        uint8 counter,
        bytes memory encodedDeliveryInstructionContainer,
        bytes[] memory encodedVMsToBeDelivered,
        IWormhole.VM memory parsedInstruction
    ) internal {
        uint8 payloadId = parsedInstruction.payload.toUint8(0);
        if (payloadId == 1) {
            CoreRelayer.DeliveryInstructionsContainer memory container =
                contracts.coreRelayerFull.getDeliveryInstructionsContainer(parsedInstruction.payload);
            for (uint8 k = 0; k < container.instructions.length; k++) {
                uint256 budget =
                    container.instructions[k].maximumRefundTarget + container.instructions[k].receiverValueTarget;
                uint16 targetChain = container.instructions[k].targetChain;
                CoreRelayer.TargetDeliveryParametersSingle memory package = CoreRelayerStructs
                    .TargetDeliveryParametersSingle({
                    encodedVMs: encodedVMsToBeDelivered,
                    deliveryIndex: counter,
                    multisendIndex: k,
                    relayerRefundAddress: payable(map[targetChain].relayer)
                });
                uint256 wormholeFee = map[targetChain].wormhole.messageFee();
                vm.prank(map[targetChain].relayer);
                map[targetChain].coreRelayerFull.deliverSingle{value: (budget + wormholeFee)}(package);
                pastDeliveries[keccak256(abi.encodePacked(parsedInstruction.hash, k))] = package;
            }
        } else if (payloadId == 2) {
            CoreRelayer.RedeliveryByTxHashInstruction memory instruction =
                contracts.coreRelayerFull.getRedeliveryByTxHashInstruction(parsedInstruction.payload);
            CoreRelayer.TargetDeliveryParametersSingle memory originalDelivery =
                pastDeliveries[keccak256(abi.encodePacked(instruction.sourceTxHash, instruction.multisendIndex))];
            uint16 targetChain = instruction.targetChain;
            uint256 budget = instruction.newMaximumRefundTarget + instruction.newReceiverValueTarget
                + map[targetChain].wormhole.messageFee();
            CoreRelayerStructs.TargetRedeliveryByTxHashParamsSingle memory package = CoreRelayerStructs
                .TargetRedeliveryByTxHashParamsSingle({
                redeliveryVM: encodedDeliveryInstructionContainer,
                sourceEncodedVMs: originalDelivery.encodedVMs,
                relayerRefundAddress: payable(map[targetChain].relayer)
            });
            vm.prank(map[targetChain].relayer);
            map[targetChain].coreRelayerFull.redeliverSingle{value: budget}(package);
        }
    }
}
