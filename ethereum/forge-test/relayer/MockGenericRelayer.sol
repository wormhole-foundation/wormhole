// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {IWormholeRelayer} from "../../contracts/interfaces/relayer/IWormholeRelayer.sol";
import {IDelivery} from "../../contracts/interfaces/relayer/IDelivery.sol";
import {IWormholeRelayerInstructionParser} from "../../contracts/interfaces/relayer/IWormholeRelayerInstructionParser.sol";
import {IWormhole} from "../../contracts/interfaces/IWormhole.sol";
import {WormholeSimulator} from "./WormholeSimulator.sol";

import "../../contracts/libraries/external/BytesLib.sol";
import "forge-std/Vm.sol";

contract MockGenericRelayer {
    using BytesLib for bytes;

    IWormhole relayerWormhole;
    WormholeSimulator relayerWormholeSimulator;
    IWormholeRelayerInstructionParser parser;
    uint256 transactionIndex;

    address private constant VM_ADDRESS = address(bytes20(uint160(uint256(keccak256("hevm cheat code")))));

    Vm public constant vm = Vm(VM_ADDRESS);

    mapping(uint16 => address) wormholeRelayerContracts;

    mapping(uint16 => address) relayers;

    mapping(bytes32 => bytes[]) pastEncodedVMs;

    mapping(bytes32 => bytes) pastEncodedDeliveryVAA;

    constructor(address _wormhole, address _wormholeSimulator, address wormholeRelayer) {
        // deploy Wormhole

        relayerWormhole = IWormhole(_wormhole);
        relayerWormholeSimulator = WormholeSimulator(_wormholeSimulator);
        parser = IWormholeRelayerInstructionParser(wormholeRelayer);
        transactionIndex = 0;
    }

    function getPastEncodedVMs(uint16 chainId, uint64 deliveryVAASequence) public view returns (bytes[] memory) {
        return pastEncodedVMs[keccak256(abi.encodePacked(chainId, deliveryVAASequence))];
    }

    function getPastDeliveryVAA(uint16 chainId, uint64 deliveryVAASequence) public view returns (bytes memory) {
        return pastEncodedDeliveryVAA[keccak256(abi.encodePacked(chainId, deliveryVAASequence))];
    }

    function setWormholeRelayerContract(uint16 chainId, address contractAddress) public {
        wormholeRelayerContracts[chainId] = contractAddress;
    }

    function setProviderDeliveryAddress(uint16 chainId, address deliveryAddress) public {
        relayers[chainId] = deliveryAddress;
    }

    function relay(uint16 chainId) public {
        relay(vm.getRecordedLogs(), chainId);
    }

    function vaaKeyMatchesVAA(IWormholeRelayer.VaaKey memory vaaKey, bytes memory signedVaa)
        internal
        view
        returns (bool)
    {
        IWormhole.VM memory parsedVaa = relayerWormhole.parseVM(signedVaa);
        if (vaaKey.infoType == IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE) {
            return
                (vaaKey.chainId == parsedVaa.emitterChainId) && (vaaKey.emitterAddress == parsedVaa.emitterAddress) && (vaaKey.sequence == parsedVaa.sequence);
        } else if (vaaKey.infoType == IWormholeRelayer.VaaKeyType.VAAHASH) {
            return (vaaKey.vaaHash == parsedVaa.hash);
        } else {
            return false;
        }
    }

    function relay(Vm.Log[] memory logs, uint16 chainId) public {
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
                parsed[i].emitterAddress == parser.toWormholeFormat(wormholeRelayerContracts[chainId])
                    && (parsed[i].emitterChainId == chainId)
            ) {
                genericRelay(encodedVMs[i], encodedVMs, parsed[i]);
            }
        }
    }

    function genericRelay(
        bytes memory encodedDeliveryVAA,
        bytes[] memory encodedVMs,
        IWormhole.VM memory parsedDeliveryVAA
    ) internal {
        uint8 payloadId = parsedDeliveryVAA.payload.toUint8(0);
        if (payloadId == 1) {
            IWormholeRelayerInstructionParser.DeliveryInstructionsContainer memory container =
                parser.decodeDeliveryInstructionsContainer(parsedDeliveryVAA.payload);

            bytes[] memory encodedVMsToBeDelivered = new bytes[](container.messages.length);

            for (uint8 i = 0; i < container.messages.length; i++) {
                for (uint8 j = 0; j < encodedVMs.length; j++) {
                    if (vaaKeyMatchesVAA(container.messages[i], encodedVMs[j])) {
                        encodedVMsToBeDelivered[i] = encodedVMs[j];
                        break;
                    }
                }
            }

            for (uint8 k = 0; k < container.instructions.length; k++) {
                uint256 budget =
                    container.instructions[k].maximumRefundTarget + container.instructions[k].receiverValueTarget;

                uint16 targetChain = container.instructions[k].targetChain;
                IDelivery.TargetDeliveryParameters memory package = IDelivery.TargetDeliveryParameters({
                    encodedVMs: encodedVMsToBeDelivered,
                    encodedDeliveryVAA: encodedDeliveryVAA,
                    multisendIndex: k,
                    relayerRefundAddress: payable(relayers[targetChain])
                });

                vm.prank(relayers[targetChain]);
                IDelivery(wormholeRelayerContracts[targetChain]).deliver{value: budget}(package);
            }
            bytes32 key = keccak256(abi.encodePacked(parsedDeliveryVAA.emitterChainId, parsedDeliveryVAA.sequence));
            pastEncodedVMs[key] = encodedVMsToBeDelivered;
            pastEncodedDeliveryVAA[key] = encodedDeliveryVAA;
        }
    }
}
