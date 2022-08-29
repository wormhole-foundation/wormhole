// contracts/mock/MockBatchedVAASender.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";
import "../interfaces/IWormhole.sol";

contract MockBatchedVAASender {
    using BytesLib for bytes;

    address wormholeCoreAddress;
    mapping(bytes32 => bytes) verifiedPayloads;

    function sendMultipleMessages(
        uint32 nonce,
        bytes[] calldata payload,
        uint8[] calldata consistencyLevel
    ) public payable returns (uint256[] memory) {
        require(payload.length == consistencyLevel.length, "invalid input parameters");

        // cache wormhole instance and payload length to save on gas
        IWormhole wormhole = wormholeCore();
        uint256 wormholeFee = wormhole.messageFee();
        uint256 numPayloads = payload.length;

        // confirm msg.value can cover messaging fees
        require(msg.value == wormholeFee * numPayloads, "insufficient value");

        // send each wormhole message and save the message sequence
        uint256[] memory messageSequences = new uint256[](numPayloads);
        for (uint256 i = 0; i < numPayloads; ++i) {
            messageSequences[i] = wormhole.publishMessage{value: wormholeFee}(
                nonce,
                payload[i],
                consistencyLevel[i]
            );
        }
        return messageSequences;
    }

    function consumeBatchVAA(bytes memory encodedVm2) public {
        // parse and verify a batch VAA
        IWormhole.VM2 memory vm = parseAndVerifyBatchVM(encodedVm2, true);

        // save each hash in the batch
        bytes32[] memory cachedHashes = new bytes32[](vm.indexedObservations.length);

        // consume individual VAAs in the batch
        uint256 observationsLen = vm.indexedObservations.length;
        for (uint256 i = 0; i < observationsLen; i++) {
            consumeSingleVAA(vm.indexedObservations[i].observation);
            cachedHashes[i] = vm.hashes[vm.indexedObservations[i].index];
        }

        // clear the batch cache
        wormholeCore().clearBatchCache(cachedHashes);
    }

    function consumeSingleVAA(bytes memory encodedVm) public {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormholeCore().parseAndVerifyVM(encodedVm);
        require(valid, reason);

        // encode the observation
        bytes memory encodedObservation = abi.encodePacked(
            vm.timestamp,
            vm.nonce,
            vm.emitterChainId,
            vm.emitterAddress,
            vm.sequence,
            vm.consistencyLevel,
            vm.payload
        );

        // save each payload in the verifiedPayloads map
        bytes32 observationHash = keccak256(abi.encodePacked(keccak256(encodedObservation)));
        verifiedPayloads[observationHash] = vm.payload;
    }

    function getPayload(bytes32 hash) public view returns (bytes memory) {
        return verifiedPayloads[hash];
    }

    function clearPayload(bytes32 hash) public {
        delete verifiedPayloads[hash];
    }

    function parseAndVerifyBatchVM(bytes memory encodedVm, bool cacheObservations) public returns (IWormhole.VM2 memory) {
        // parse and verify a batch VAA
        (IWormhole.VM2 memory vm, bool valid, string memory reason) = wormholeCore().parseAndVerifyBatchVM(encodedVm, cacheObservations);
        require(valid, reason);
        require(vm.version == 2, "wrong version type");
        return vm;
    }

    function parseBatchVM(bytes memory encodedVm) public view returns (IWormhole.VM2 memory) {
        return wormholeCore().parseBatchVM(encodedVm);
    }

    function parseVM(bytes memory encodedVm) public view returns (IWormhole.VM memory) {
        return wormholeCore().parseVM(encodedVm);
    }

    function wormholeCore() private view returns (IWormhole) {
        return IWormhole(wormholeCoreAddress);
    }

    function setup(address _wormholeCore) public {
        wormholeCoreAddress = _wormholeCore;
    }
}
