/// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "forge-std/Test.sol";
import "./utils/WormholeTestUtils.t.sol";
import "../contracts/interfaces/IWormhole.sol";

contract MessagesGasBenchmark is Test, WormholeTestUtils {
    // 19, current mainnet number of guardians, is used to have gas estimates
    // close to our mainnet transactions.
    uint8 constant NUM_GUARDIANS = 19;
    // 2/3 of the guardians should sign a message for a VAA which is 13 out of 19 guardians.
    // It is possible to have more signers but the median seems to be 13.
    uint8 constant NUM_GUARDIAN_SIGNERS = 13;

    uint constant PYTH_PAYLOAD_SIZE = 24; //approximately assuming there are 5 attestations in a batch

    uint constant TOKEN_BRIDGE_PAYLOAD_SIZE = 4; //approximately
    
    IWormhole public wormhole;
    
    bytes VAA;
    bytes pythPayload;
    bytes tokenBridgePayload;

    uint64 sequence;
    uint randSeed;

    function setUp() public {
        wormhole = IWormhole(setUpWormhole(NUM_GUARDIANS));

        for(uint i=0; i< PYTH_PAYLOAD_SIZE; i++){
            pythPayload = abi.encodePacked(pythPayload, bytes32(getRand()));
        }

        for(uint i=0; i< TOKEN_BRIDGE_PAYLOAD_SIZE; i++){
            tokenBridgePayload = abi.encodePacked(tokenBridgePayload, bytes32(getRand()));
        }
    }

    function getRand() internal returns (uint val) {
        ++randSeed;
        val = uint(keccak256(abi.encode(randSeed)));
    }

    function testBenchmarkParseAndVerifyVMPythPayload(uint32 timestamp, uint16 emitterChainId, bytes32 emitterAddress, uint64 sequence) public {
        bytes memory vaa = generateVaa(
                timestamp,
                emitterChainId,
                emitterAddress,
                sequence,
                pythPayload,
                NUM_GUARDIAN_SIGNERS
            );
        wormhole.parseAndVerifyVM(vaa);
    }

    function testBenchmarkParseAndVerifyVMTokenBridgePayload(uint32 timestamp, uint16 emitterChainId, bytes32 emitterAddress, uint64 sequence) public {
        bytes memory vaa = generateVaa(
                timestamp,
                emitterChainId,
                emitterAddress,
                sequence,
                tokenBridgePayload,
                NUM_GUARDIAN_SIGNERS
            );
        wormhole.parseAndVerifyVM(vaa);
    }

    function testBenchmarkParseAndVerifyVMRandomPayload(uint32 timestamp, uint16 emitterChainId, bytes32 emitterAddress, uint64 sequence, bytes memory payloadRandom) public {
        bytes memory vaa = generateVaa(
                timestamp,
                emitterChainId,
                emitterAddress,
                sequence,
                payloadRandom,
                NUM_GUARDIAN_SIGNERS
            );
        wormhole.parseAndVerifyVM(vaa);
    }

    function testBenchmarkMessageFee() public view {
        wormhole.messageFee();
    }

    function testPublishMessage(bytes memory payload) public {
        wormhole.publishMessage(0, payload, 0);
    }
}
