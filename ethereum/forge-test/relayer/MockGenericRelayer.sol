// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../contracts/interfaces/relayer/IWormholeRelayer.sol";
import {IWormhole} from "../../contracts/interfaces/IWormhole.sol";
import {WormholeSimulator} from "./WormholeSimulator.sol";
import {toWormholeFormat} from "../../contracts/libraries/relayer/Utils.sol";
import {
    DeliveryInstruction,
    DeliveryOverride,
    RedeliveryInstruction
} from "../../contracts/libraries/relayer/RelayerInternalStructs.sol";
import {CoreRelayerSerde} from "../../contracts/relayer/coreRelayer/CoreRelayerSerde.sol";
import "../../contracts/libraries/external/BytesLib.sol";
import "forge-std/Vm.sol";
import "../../contracts/interfaces/relayer/TypedUnits.sol";
import "../../contracts/libraries/relayer/ExecutionParameters.sol";

contract MockGenericRelayer {
    using BytesLib for bytes;
    using WeiLib for Wei;
    using GasLib for Gas;

    IWormhole relayerWormhole;
    WormholeSimulator relayerWormholeSimulator;
    uint256 transactionIndex;

    address private constant VM_ADDRESS =
        address(bytes20(uint160(uint256(keccak256("hevm cheat code")))));

    Vm public constant vm = Vm(VM_ADDRESS);

    mapping(uint16 => address) wormholeRelayerContracts;

    mapping(uint16 => address) relayers;

    mapping(bytes32 => bytes[]) pastEncodedVMs;

    mapping(bytes32 => bytes) pastEncodedDeliveryVAA;

    constructor(address _wormhole, address _wormholeSimulator) {
        // deploy Wormhole

        relayerWormhole = IWormhole(_wormhole);
        relayerWormholeSimulator = WormholeSimulator(_wormholeSimulator);
        transactionIndex = 0;
    }

    function getPastEncodedVMs(
        uint16 chainId,
        uint64 deliveryVAASequence
    ) public view returns (bytes[] memory) {
        return pastEncodedVMs[keccak256(abi.encodePacked(chainId, deliveryVAASequence))];
    }

    function getPastDeliveryVAA(
        uint16 chainId,
        uint64 deliveryVAASequence
    ) public view returns (bytes memory) {
        return pastEncodedDeliveryVAA[keccak256(abi.encodePacked(chainId, deliveryVAASequence))];
    }

    function setInfo(
        uint16 chainId,
        uint64 deliveryVAASequence,
        bytes[] memory encodedVMs,
        bytes memory encodedDeliveryVAA
    ) internal {
        pastEncodedVMs[keccak256(abi.encodePacked(chainId, deliveryVAASequence))] = encodedVMs;
        pastEncodedDeliveryVAA[keccak256(abi.encodePacked(chainId, deliveryVAASequence))] =
            encodedDeliveryVAA;
    }

    function setWormholeRelayerContract(uint16 chainId, address contractAddress) public {
        wormholeRelayerContracts[chainId] = contractAddress;
    }

    function setProviderDeliveryAddress(uint16 chainId, address deliveryAddress) public {
        relayers[chainId] = deliveryAddress;
    }

    function relay(uint16 chainId) public {
        relay(vm.getRecordedLogs(), chainId, bytes(""));
    }

    function vaaKeyMatchesVAA(
        VaaKey memory vaaKey,
        bytes memory signedVaa
    ) internal view returns (bool) {
        IWormhole.VM memory parsedVaa = relayerWormhole.parseVM(signedVaa);
        return (vaaKey.chainId == parsedVaa.emitterChainId)
            && (vaaKey.emitterAddress == parsedVaa.emitterAddress)
            && (vaaKey.sequence == parsedVaa.sequence);
    }

    function relay(Vm.Log[] memory logs, uint16 chainId, bytes memory deliveryOverrides) public {
        Vm.Log[] memory entries = relayerWormholeSimulator.fetchWormholeMessageFromLog(logs);
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
        for (uint16 i = 0; i < encodedVMs.length; i++) {
            if (
                parsed[i].emitterAddress == toWormholeFormat(wormholeRelayerContracts[chainId])
                    && (parsed[i].emitterChainId == chainId)
            ) {
                genericRelay(encodedVMs[i], encodedVMs, parsed[i], deliveryOverrides);
            }
        }
    }

    function relay(uint16 chainId, bytes memory deliveryOverrides) public {
        relay(vm.getRecordedLogs(), chainId, deliveryOverrides);
    }

    function genericRelay(
        bytes memory encodedDeliveryVAA,
        bytes[] memory encodedVMs,
        IWormhole.VM memory parsedDeliveryVAA,
        bytes memory deliveryOverrides
    ) internal {
        uint8 payloadId = parsedDeliveryVAA.payload.toUint8(0);
        if (payloadId == 1) {
            DeliveryInstruction memory instruction =
                CoreRelayerSerde.decodeDeliveryInstruction(parsedDeliveryVAA.payload);

            bytes[] memory encodedVMsToBeDelivered = new bytes[](instruction.vaaKeys.length);

            for (uint8 i = 0; i < instruction.vaaKeys.length; i++) {
                for (uint8 j = 0; j < encodedVMs.length; j++) {
                    if (vaaKeyMatchesVAA(instruction.vaaKeys[i], encodedVMs[j])) {
                        encodedVMsToBeDelivered[i] = encodedVMs[j];
                        break;
                    }
                }
            }

            EvmExecutionInfoV1 memory executionInfo = decodeEvmExecutionInfoV1(instruction.encodedExecutionInfo);
            Wei budget = executionInfo.gasLimit.toWei(executionInfo.targetChainRefundPerGasUnused) + instruction.requestedReceiverValue + instruction.extraReceiverValue;

            uint16 targetChainId = instruction.targetChainId;

            vm.prank(relayers[targetChainId]);
            IWormholeRelayerDelivery(wormholeRelayerContracts[targetChainId]).deliver{
                value: budget.unwrap()
            }(
                encodedVMsToBeDelivered,
                encodedDeliveryVAA,
                payable(relayers[targetChainId]),
                deliveryOverrides
            );

            setInfo(
                parsedDeliveryVAA.emitterChainId,
                parsedDeliveryVAA.sequence,
                encodedVMsToBeDelivered,
                encodedDeliveryVAA
            );
        } else if (payloadId == 2) {
            RedeliveryInstruction memory instruction =
                CoreRelayerSerde.decodeRedeliveryInstruction(parsedDeliveryVAA.payload);

            

            DeliveryOverride memory deliveryOverride = DeliveryOverride({
                newExecutionInfo: instruction.newEncodedExecutionInfo,
                newReceiverValue: instruction.newRequestedReceiverValue,
                redeliveryHash: parsedDeliveryVAA.hash
            });

            EvmExecutionInfoV1 memory executionInfo = decodeEvmExecutionInfoV1(instruction.newEncodedExecutionInfo);
            Wei budget = executionInfo.gasLimit.toWei(executionInfo.targetChainRefundPerGasUnused) + instruction.newRequestedReceiverValue;

            bytes memory oldEncodedDeliveryVAA = getPastDeliveryVAA(
                instruction.deliveryVaaKey.chainId, instruction.deliveryVaaKey.sequence
            );
            bytes[] memory oldEncodedVMs = getPastEncodedVMs(
                instruction.deliveryVaaKey.chainId, instruction.deliveryVaaKey.sequence
            );

            uint16 targetChainId = CoreRelayerSerde.decodeDeliveryInstruction(
                relayerWormhole.parseVM(oldEncodedDeliveryVAA).payload
            ).targetChainId;

            vm.prank(relayers[targetChainId]);
            IWormholeRelayerDelivery(wormholeRelayerContracts[targetChainId]).deliver{
                value: budget.unwrap()
            }(
                oldEncodedVMs,
                oldEncodedDeliveryVAA,
                payable(relayers[targetChainId]),
                CoreRelayerSerde.encode(deliveryOverride)
            );
        }
    }
}
