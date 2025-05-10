// SPDX-License-Identifier: AGPLv3
pragma solidity >=0.8.4;

import { Implementation } from "../contracts/Implementation.sol";
import "./WormholeSigner.sol";
import "./FuzzingHelpers.sol";

contract DiffFuzzUpgrades is WormholeSigner, FuzzingHelpers {
    address implementationV1;
    address implementationV2;
    IWormhole wormhole;
    uint256 fork1;
    uint256 fork2;

    event SwitchedFork(uint256 forkId);

    struct ContractUpgrade {
        bytes32 module;
        uint8 action;
        uint16 chain;
        address newContract;
    }
    struct GuardianSetUpgrade {
        bytes32 module;
        uint8 action;
        uint16 chain;
        GuardianSet newGuardianSet;
        uint32 newGuardianSetIndex;
    }
    struct SetMessageFee {
        bytes32 module;
        uint8 action;
        uint16 chain;
        uint256 messageFee;
    }
    struct TransferFees {
        bytes32 module;
        uint8 action;
        uint16 chain;
        uint256 amount;
        bytes32 recipient;
    }
    struct RecoverChainId {
        bytes32 module;
        uint8 action;
        uint256 evmChainId;
        uint16 newChainId;
    }
    struct GuardianSet {
        address[] keys;
        uint32 expirationTime;
    }

    mapping(bytes32 => bool) public governanceMessageIsConsumed;
    bytes32[] public governanceMessagesConsumed;
    mapping(address => bool) public initializedImplementation;


    constructor() {
        hevm.roll(20286167);
        hevm.warp(1720735919);
        fork1 = hevm.createFork("1");
        fork2 = hevm.createFork("2");
        fork1 = 1;
        fork2 = 2;
        implementationV1 = 0x3c3d457f1522D3540AB3325Aa5f1864E34cBA9D0;
        wormhole = IWormhole(0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B);
        
        hevm.selectFork(fork1);
        hevm.deal(address(this), type(uint256).max);

        overrideToTestGuardian(IWormhole(wormhole), hevm.addr(testGuardianKey));

        // Perform a "fake" upgrade on fork2
        // We set the implementation address in the proxy and then call initialize, which
        // usually is called during the proper upgrade flow. We test the proper upgrade
        // flow later
        hevm.selectFork(fork2);
        hevm.deal(address(this), type(uint256).max);

        implementationV2 = address(new Implementation());

        hevm.store(
            address(wormhole),
            bytes32(uint(24440054405305269366569402256811496959409073762505157381672968839269610695612)),
            bytes32(uint256(uint160(address(implementationV2))))
        );

        (bool success, bytes memory output) = address(wormhole).call(
            abi.encodeWithSignature(
                'initialize()'
            )
        );
        assert(success == true);
        initializedImplementation[implementationV2] = true;
        
        overrideToTestGuardian(IWormhole(wormhole), hevm.addr(testGuardianKey));
    }


    /*** Modified Functions ***/ 

    function Implementation_submitNewGuardianSet(bool validVAA, bytes32 seed, bytes memory a) public virtual {
        bytes32 messageHash;
        
        if (validVAA) {
            uint32 newIndex = rollNewGuardianSet(wormhole, seed);

            uint256 newGuardianSetLength = pendingGuardianSetAddresses.length;
            a = abi.encodePacked(MODULE, uint8(2), CHAINID, newIndex, uint8(newGuardianSetLength));

            for (uint256 i = 0; i < newGuardianSetLength; i++) {
                a = abi.encodePacked(a, pendingGuardianSetAddresses[i]);
            }

            (a, messageHash) = encodeAndSignGovernanceMessage(a, wormhole);

            // After rolling a new guardian set but before sending the message we need to make sure the forks are in sync
            // because we've been calling one fork for the index, but not the other. Guardian set expiry time matters
            // when comparing the results of the two forks
            // Keep the forks in sync
            hevm.selectFork(fork2);
            emit SwitchedFork(fork2);
            uint blockNo = block.number;
            uint blockTime = block.timestamp;
            hevm.selectFork(fork1);
            emit SwitchedFork(fork1);
            hevm.roll(blockNo);
            hevm.warp(blockTime);
        }

        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'submitNewGuardianSet(bytes)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'submitNewGuardianSet(bytes)', a
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            assert(validVAA == true);
            assert(keccak256(outputV1) == keccak256(outputV2));
            
            // We need to commit the new guardian set so it can start signing future messages
            commitNewGuardianSet();
            governanceMessageIsConsumed[messageHash] = true;
            governanceMessagesConsumed.push(messageHash);
        }
        else {
            assert(validVAA == false);
        }
    }

    function Implementation_parseAndVerifyVM(bool validVAA, bytes memory a, uint16 emitterChainId, bytes32 emitterAddress) public virtual {
        bytes memory originalPayload = a;
        if (validVAA) {
            a = encodeAndSignMessage(a, emitterChainId, emitterAddress, wormhole);
        }

        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'parseAndVerifyVM(bytes)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'parseAndVerifyVM(bytes)', a
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
            assert(validVAA);

            IWormhole.VM memory vaa = abi.decode(outputV2, (IWormhole.VM));
            assert(vaa.emitterAddress == emitterAddress);
            assert(vaa.emitterChainId == emitterChainId);
            assert(keccak256(vaa.payload) == keccak256(originalPayload));
        }
        else {
            assert(validVAA == false);
        }
    }

    function Implementation_verifyVM(IWormhole.VM memory a) public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'verifyVM(Structs.VM)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'verifyVM(Structs.VM)', a
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_verifySignatures(bytes32 a, IWormhole.Signature[] memory b, GuardianSet memory c) public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'verifySignatures(bytes32,Structs.Signature[],Structs.GuardianSet)', a, b, c
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'verifySignatures(bytes32,Structs.Signature[],Structs.GuardianSet)', a, b, c
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_parseVM(bool validVAA, bytes memory a, uint16 emitterChainId, bytes32 emitterAddress) public virtual {
        if (validVAA) {
            a = encodeAndSignMessage(a, emitterChainId, emitterAddress, wormhole);
        }
        
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'parseVM(bytes)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'parseVM(bytes)', a
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_initialize() public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'initialize()'
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'initialize()'
            )
        );
        assert(successV1 == successV2);
        // We don't want to be able to re-initialize 
        assert(successV1 == false);
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }


    /*** Tainted Functions ***/ 

    function Implementation_submitContractUpgrade(bool validVAA, bytes memory a) public virtual {
        bytes32 messageHash;

        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);

        address newImplementation = address(new Implementation());

        if (validVAA) {
           a = abi.encodePacked(MODULE, uint8(1), CHAINID, uint256(uint160(newImplementation)));
           (a, messageHash) = encodeAndSignGovernanceMessage(a, wormhole);
        }

        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'submitContractUpgrade(bytes)', a
            )
        );

        if (validVAA) {
            assert(successV2 == true);
            initializedImplementation[newImplementation] = true;

            // We're explicitly not setting these hashes as consumed becuase we only
            // submit the contract upgrade to V2

            // governanceMessageIsConsumed[messageHash] = true;
            // governanceMessagesConsumed.push(messageHash);
        }
        else{
            assert(successV2 == false);
        }

        // Keep the forks in sync
        uint blockNo = block.number;
        uint blockTime = block.timestamp;
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.roll(blockNo);
        hevm.warp(blockTime);
    }

    function Implementation_submitSetMessageFee(bool validVAA, bool fee, bytes memory a) public virtual {
        uint256 messageFee = fee ? 1 ether : 0;
        bytes32 messageHash;
        
        if (validVAA) {
           a = abi.encodePacked(MODULE, uint8(3), CHAINID, messageFee);
           (a, messageHash) = encodeAndSignGovernanceMessage(a, wormhole);
        }

        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'submitSetMessageFee(bytes)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'submitSetMessageFee(bytes)', a
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            // It must have been a valid VAA
            assert(validVAA == true);
            assert(keccak256(outputV1) == keccak256(outputV2));

            governanceMessageIsConsumed[messageHash] = true;
            governanceMessagesConsumed.push(messageHash);
        }
        else {
            assert(validVAA == false);
        }
    }

    function Implementation_submitTransferFees(bool validVAA, uint256 amount, bytes32 receiver, bytes memory a) public virtual {
        bytes32 messageHash;
        uint256 wormholeBalanceBefore = address(wormhole).balance;
        
        if (validVAA) {
            amount = clampBetween(amount, 0, wormholeBalanceBefore);
            a = abi.encodePacked(MODULE, uint8(4), CHAINID, amount, receiver);
            (a, messageHash) = encodeAndSignGovernanceMessage(a, wormhole);
        }

        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'submitTransferFees(bytes)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'submitTransferFees(bytes)', a
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
            assert(validVAA == true);

            governanceMessageIsConsumed[messageHash] = true;
            governanceMessagesConsumed.push(messageHash);

            assert(address(wormhole).balance == wormholeBalanceBefore - amount);

            hevm.selectFork(fork1);
            emit SwitchedFork(fork1);
            assert(address(wormhole).balance == wormholeBalanceBefore - amount);
        }
    }

    function Implementation_submitRecoverChainId(bytes memory a) public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'submitRecoverChainId(bytes)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'submitRecoverChainId(bytes)', a
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_getGuardianSet(uint32 a) public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'getGuardianSet(uint32)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'getGuardianSet(uint32)', a
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_getCurrentGuardianSetIndex() public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'getCurrentGuardianSetIndex()'
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'getCurrentGuardianSetIndex()'
            )
        );
        assert(successV1 == successV2);
        // This getter should always succeed 
        assert(successV1 == true);
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_getGuardianSetExpiry() public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'getGuardianSetExpiry()'
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'getGuardianSetExpiry()'
            )
        );
        assert(successV1 == successV2); 
        // This getter should always succeed 
        assert(successV1 == true);
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_governanceActionIsConsumed(bool consumedMessage, uint256 messageIndex, bytes32 a) public virtual {
        if (consumedMessage && governanceMessagesConsumed.length != 0) {
            messageIndex = clampBetween(messageIndex, 0, governanceMessagesConsumed.length - 1);
            a = governanceMessagesConsumed[messageIndex];
        }
        
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'governanceActionIsConsumed(bytes32)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'governanceActionIsConsumed(bytes32)', a
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));

            if (consumedMessage && governanceMessagesConsumed.length != 0) {
                assert(abi.decode(outputV2, (bool)) == true);
            }
        }
    }

    function Implementation_isInitialized(address a) public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'isInitialized(address)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'isInitialized(address)', a
            )
        );
        assert(successV1 == successV2);
        // This getter should always succeed 
        assert(successV1 == true);
        if(successV1 && successV2) {
            // We should always get the same result on both forks, apart from with new implementations
            if (initializedImplementation[a] == false) {
                assert(keccak256(outputV1) == keccak256(outputV2));

                // This implementation should be initialised on both forks
                if (a == implementationV1) {
                    assert(abi.decode(outputV2, (bool)) == true);
                }
            }
            else {
                // The v2 implementation will only be initialised on fork 2
                assert(abi.decode(outputV2, (bool)) == true);
                assert(abi.decode(outputV1, (bool)) == false);
            }
        }
    }

    function Implementation_chainId() public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'chainId()'
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'chainId()'
            )
        );
        assert(successV1 == successV2); 
        // This getter should always succeed 
        assert(successV1 == true);
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
            assert(abi.decode(outputV2, (uint16)) == 2);
        }
    }

    function Implementation_evmChainId() public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'evmChainId()'
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'evmChainId()'
            )
        );
        assert(successV1 == successV2); 
        // This getter should always succeed 
        assert(successV1 == true);
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
            assert(abi.decode(outputV2, (uint256)) == 1);
        }
    }

    function Implementation_isFork() public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'isFork()'
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'isFork()'
            )
        );
        assert(successV1 == successV2); 
        // This getter should always succeed 
        assert(successV1 == true);
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_governanceChainId() public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'governanceChainId()'
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'governanceChainId()'
            )
        );
        assert(successV1 == successV2); 
        // This getter should always succeed 
        assert(successV1 == true);
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
            assert(abi.decode(outputV2, (uint16)) == 1);
        }
    }

    function Implementation_governanceContract() public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'governanceContract()'
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'governanceContract()'
            )
        );
        assert(successV1 == successV2); 
        // This getter should always succeed 
        assert(successV1 == true);
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
            assert(abi.decode(outputV2, (bytes32)) == governanceContract);
        }
    }

    function Implementation_messageFee() public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'messageFee()'
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'messageFee()'
            )
        );
        assert(successV1 == successV2); 
        // This getter should always succeed 
        assert(successV1 == true);
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_nextSequence(address a) public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'nextSequence(address)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'nextSequence(address)', a
            )
        );
        assert(successV1 == successV2); 
        // This getter should always succeed 
        assert(successV1 == true);
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_publishMessage(uint32 a, bytes memory b, uint8 c) public virtual {
        
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);

        uint256 messageFee = IWormhole(wormhole).messageFee();
        uint256 balanceBefore = address(this).balance;

        (bool successV1, bytes memory outputV1) = address(wormhole).call{value: messageFee}(
            abi.encodeWithSignature(
                'publishMessage(uint32,bytes,uint8)', a, b, c
            )
        );

        assert(address(this).balance == balanceBefore - messageFee);

        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);

        balanceBefore = address(this).balance;

        (bool successV2, bytes memory outputV2) = address(wormhole).call{value: messageFee}(
            abi.encodeWithSignature(
                'publishMessage(uint32,bytes,uint8)', a, b, c
            )
        );
        assert(address(this).balance == balanceBefore - messageFee);

        assert(successV1 == successV2); 
        // This should always succeed
        assert(successV1 == true);
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }
}
