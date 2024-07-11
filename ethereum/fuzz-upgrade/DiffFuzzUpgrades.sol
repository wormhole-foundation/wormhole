// SPDX-License-Identifier: AGPLv3
pragma solidity >=0.8.4;

import { Implementation } from "../contracts/Implementation.sol";
interface IImplementationV1 {
    enum GovernanceAction { UpgradeContract, UpgradeGuardianset }
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
    struct VM {
        uint8 version;
        uint32 timestamp;
        uint32 nonce;
        uint16 emitterChainId;
        bytes32 emitterAddress;
        uint64 sequence;
        uint8 consistencyLevel;
        bytes payload;
        uint32 guardianSetIndex;
        Signature[] signatures;
        bytes32 hash;
    }
    struct Signature {
        bytes32 r;
        bytes32 s;
        uint8 v;
        uint8 guardianIndex;
    }
    struct GuardianSet {
        address[] keys;
        uint32 expirationTime;
    }
    function submitContractUpgrade(bytes memory) external;
    function submitSetMessageFee(bytes memory) external;
    function submitNewGuardianSet(bytes memory) external;
    function submitTransferFees(bytes memory) external;
    function submitRecoverChainId(bytes memory) external;
    function parseAndVerifyVM(bytes calldata) external view returns (VM memory,bool,string memory);
    function verifyVM(VM memory) external view returns (bool,string memory);
    function verifySignatures(bytes32,Signature[] memory,GuardianSet memory) external pure returns (bool,string memory);
    function parseVM(bytes memory) external pure returns (VM memory);
    function quorum(uint256) external pure returns (uint256);
    function getGuardianSet(uint32) external view returns (GuardianSet memory);
    function getCurrentGuardianSetIndex() external view returns (uint32);
    function getGuardianSetExpiry() external view returns (uint32);
    function governanceActionIsConsumed(bytes32) external view returns (bool);
    function isInitialized(address) external view returns (bool);
    function chainId() external view returns (uint16);
    function evmChainId() external view returns (uint256);
    function isFork() external view returns (bool);
    function governanceChainId() external view returns (uint16);
    function governanceContract() external view returns (bytes32);
    function messageFee() external view returns (uint256);
    function nextSequence(address) external view returns (uint64);
    function parseContractUpgrade(bytes memory) external pure returns (ContractUpgrade memory);
    function parseGuardianSetUpgrade(bytes memory) external pure returns (GuardianSetUpgrade memory);
    function parseSetMessageFee(bytes memory) external pure returns (SetMessageFee memory);
    function parseTransferFees(bytes memory) external pure returns (TransferFees memory);
    function parseRecoverChainId(bytes memory) external pure returns (RecoverChainId memory);
    function publishMessage(uint32,bytes memory,uint8) external payable returns (uint64);
    function initialize() external;
}

interface IImplementationV2 {
    enum GovernanceAction { UpgradeContract, UpgradeGuardianset }
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
    struct VM {
        uint8 version;
        uint32 timestamp;
        uint32 nonce;
        uint16 emitterChainId;
        bytes32 emitterAddress;
        uint64 sequence;
        uint8 consistencyLevel;
        bytes payload;
        uint32 guardianSetIndex;
        Signature[] signatures;
        bytes32 hash;
    }
    struct GuardianSet {
        address[] keys;
        uint32 expirationTime;
    }
    struct Signature {
        bytes32 r;
        bytes32 s;
        uint8 v;
        uint8 guardianIndex;
    }
    function submitContractUpgrade(bytes memory) external;
    function submitSetMessageFee(bytes memory) external;
    function submitNewGuardianSet(bytes memory) external;
    function submitTransferFees(bytes memory) external;
    function submitRecoverChainId(bytes memory) external;
    function setGuardianSetHash(uint32) external;
    function parseAndVerifyVMOptimized(bytes calldata,bytes calldata,uint32) external view returns (VM memory,bool,string memory);
    function parseGuardianSet(bytes calldata) external pure returns (GuardianSet memory);
    function parseAndVerifyVM(bytes calldata) external view returns (VM memory,bool,string memory);
    function verifyVM(VM memory) external view returns (bool,string memory);
    function verifySignatures(bytes32,Signature[] memory,GuardianSet memory) external pure returns (bool,string memory);
    function verifyCurrentQuorum(bytes32,Signature[] memory) external view returns (bool,string memory);
    function parseVM(bytes memory) external view returns (VM memory);
    function quorum(uint256) external pure returns (uint256);
    function getGuardianSet(uint32) external view returns (GuardianSet memory);
    function getCurrentGuardianSetIndex() external view returns (uint32);
    function getGuardianSetExpiry() external view returns (uint32);
    function governanceActionIsConsumed(bytes32) external view returns (bool);
    function isInitialized(address) external view returns (bool);
    function chainId() external view returns (uint16);
    function evmChainId() external view returns (uint256);
    function isFork() external view returns (bool);
    function governanceChainId() external view returns (uint16);
    function governanceContract() external view returns (bytes32);
    function messageFee() external view returns (uint256);
    function nextSequence(address) external view returns (uint64);
    function getGuardianSetHash(uint32) external view returns (bytes32);
    function getEncodedGuardianSet(uint32) external view returns (bytes memory);
    function parseContractUpgrade(bytes memory) external pure returns (ContractUpgrade memory);
    function parseGuardianSetUpgrade(bytes memory) external pure returns (GuardianSetUpgrade memory);
    function parseSetMessageFee(bytes memory) external pure returns (SetMessageFee memory);
    function parseTransferFees(bytes memory) external pure returns (TransferFees memory);
    function parseRecoverChainId(bytes memory) external pure returns (RecoverChainId memory);
    function publishMessage(uint32,bytes memory,uint8) external payable returns (uint64);
    function initialize() external;
}

interface IWormhole {
}

interface IHevm {
    function warp(uint256 newTimestamp) external;
    function roll(uint256 newNumber) external;
    function load(address where, bytes32 slot) external returns (bytes32);
    function store(address where, bytes32 slot, bytes32 value) external;
    function sign(uint256 privateKey, bytes32 digest) external returns (uint8 r, bytes32 v, bytes32 s);
    function addr(uint256 privateKey) external returns (address add);
    function ffi(string[] calldata inputs) external returns (bytes memory result);
    function prank(address newSender) external;
    function createFork() external returns (uint256 forkId);
    function selectFork(uint256 forkId) external;
}

contract DiffFuzzUpgrades {
    IHevm hevm = IHevm(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    IImplementationV1 implementationV1;
    IImplementationV2 implementationV2;
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

    constructor() public {
        hevm.roll(20286330);
        hevm.warp(1720737875);
        fork1 = hevm.createFork();
        fork2 = hevm.createFork();
        fork1 = 1;
        fork2 = 2;
        implementationV1 = IImplementationV1(0x3c3d457f1522D3540AB3325Aa5f1864E34cBA9D0);
        implementationV2 = IImplementationV2(0x0102030405060708091011121314151617181920);
        wormhole = IWormhole(0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B);
        // Store the implementation addresses in the proxy.
        hevm.selectFork(fork1);
        hevm.store(
            address(wormhole),
            bytes32(uint(24440054405305269366569402256811496959409073762505157381672968839269610695612)),
            bytes32(uint256(uint160(address(implementationV1))))
        );
        hevm.selectFork(fork2);
        hevm.store(
            address(wormhole),
            bytes32(uint(24440054405305269366569402256811496959409073762505157381672968839269610695612)),
            bytes32(uint256(uint160(address(implementationV1))))
        );
    }

    /*** Upgrade Function ***/ 

    // TODO: Consider replacing this with the actual upgrade method
    function upgradeV2() external virtual {
        hevm.selectFork(fork2);
        hevm.store(
            address(wormhole),
            bytes32(uint(24440054405305269366569402256811496959409073762505157381672968839269610695612)),
            bytes32(uint256(uint160(address(implementationV2))))
        );
        hevm.selectFork(fork1);
        bytes32 impl1 = hevm.load(
            address(wormhole),
            bytes32(uint(24440054405305269366569402256811496959409073762505157381672968839269610695612))
        );
        bytes32 implV1 = bytes32(uint256(uint160(address(implementationV1))));
        assert(impl1 == implV1);
    }


    /*** Modified Functions ***/ 

    function Implementation_submitNewGuardianSet(bytes memory a) public virtual {
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
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_parseAndVerifyVM(bytes memory a) public virtual {
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
        }
    }

    function Implementation_verifyVM(VM memory a) public virtual {
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

    function Implementation_verifySignatures(bytes32 a, Signature[] memory b, GuardianSet memory c) public virtual {
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

    function Implementation_parseVM(bytes memory a) public virtual {
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
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }


    /*** Tainted Functions ***/ 

    function Implementation_submitContractUpgrade(bytes memory a) public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.prank(msg.sender);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'submitContractUpgrade(bytes)', a
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'submitContractUpgrade(bytes)', a
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_submitSetMessageFee(bytes memory a) public virtual {
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
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_submitTransferFees(bytes memory a) public virtual {
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
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_governanceActionIsConsumed(bytes32 a) public virtual {
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
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
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
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
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
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
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
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
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
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
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
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }

    function Implementation_publishMessage(uint32 a, bytes memory b, uint8 c) public virtual {
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        (bool successV1, bytes memory outputV1) = address(wormhole).call(
            abi.encodeWithSignature(
                'publishMessage(uint32,bytes,uint8)', a, b, c
            )
        );
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'publishMessage(uint32,bytes,uint8)', a, b, c
            )
        );
        assert(successV1 == successV2); 
        if(successV1 && successV2) {
            assert(keccak256(outputV1) == keccak256(outputV2));
        }
    }


    /*** New Functions ***/ 

    function Implementation_setGuardianSetHash(uint32 a) public virtual {
        // This function does nothing with the V1, since setGuardianSetHash is new in the V2
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        address impl = address(uint160(uint256(
            hevm.load(address(wormhole),0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc)
        )));
        require(impl == address(implementationV2));
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'setGuardianSetHash(uint32)', a
            )
        );
        // Keep the forks in sync
        uint blockNo = block.number;
        uint blockTime = block.timestamp;
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.roll(blockNo);
        hevm.warp(blockTime);
        // Never fail assertion, since there is nothing to compare
        assert(true);
    }

    function Implementation_parseAndVerifyVMOptimized(bytes memory a, bytes memory b, uint32 c) public virtual {
        // This function does nothing with the V1, since parseAndVerifyVMOptimized is new in the V2
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        address impl = address(uint160(uint256(
            hevm.load(address(wormhole),0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc)
        )));
        require(impl == address(implementationV2));
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'parseAndVerifyVMOptimized(bytes,bytes,uint32)', a, b, c
            )
        );
        // Keep the forks in sync
        uint blockNo = block.number;
        uint blockTime = block.timestamp;
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.roll(blockNo);
        hevm.warp(blockTime);
        // Never fail assertion, since there is nothing to compare
        assert(true);
    }

    function Implementation_parseGuardianSet(bytes memory a) public virtual {
        // This function does nothing with the V1, since parseGuardianSet is new in the V2
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        address impl = address(uint160(uint256(
            hevm.load(address(wormhole),0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc)
        )));
        require(impl == address(implementationV2));
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'parseGuardianSet(bytes)', a
            )
        );
        // Keep the forks in sync
        uint blockNo = block.number;
        uint blockTime = block.timestamp;
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.roll(blockNo);
        hevm.warp(blockTime);
        // Never fail assertion, since there is nothing to compare
        assert(true);
    }

    function Implementation_verifyCurrentQuorum(bytes32 a, Signature[] memory b) public virtual {
        // This function does nothing with the V1, since verifyCurrentQuorum is new in the V2
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        address impl = address(uint160(uint256(
            hevm.load(address(wormhole),0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc)
        )));
        require(impl == address(implementationV2));
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'verifyCurrentQuorum(bytes32,Structs.Signature[])', a, b
            )
        );
        // Keep the forks in sync
        uint blockNo = block.number;
        uint blockTime = block.timestamp;
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.roll(blockNo);
        hevm.warp(blockTime);
        // Never fail assertion, since there is nothing to compare
        assert(true);
    }

    function Implementation_getGuardianSetHash(uint32 a) public virtual {
        // This function does nothing with the V1, since getGuardianSetHash is new in the V2
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        address impl = address(uint160(uint256(
            hevm.load(address(wormhole),0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc)
        )));
        require(impl == address(implementationV2));
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'getGuardianSetHash(uint32)', a
            )
        );
        // Keep the forks in sync
        uint blockNo = block.number;
        uint blockTime = block.timestamp;
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.roll(blockNo);
        hevm.warp(blockTime);
        // Never fail assertion, since there is nothing to compare
        assert(true);
    }

    function Implementation_getEncodedGuardianSet(uint32 a) public virtual {
        // This function does nothing with the V1, since getEncodedGuardianSet is new in the V2
        hevm.selectFork(fork2);
        emit SwitchedFork(fork2);
        address impl = address(uint160(uint256(
            hevm.load(address(wormhole),0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc)
        )));
        require(impl == address(implementationV2));
        hevm.prank(msg.sender);
        (bool successV2, bytes memory outputV2) = address(wormhole).call(
            abi.encodeWithSignature(
                'getEncodedGuardianSet(uint32)', a
            )
        );
        // Keep the forks in sync
        uint blockNo = block.number;
        uint blockTime = block.timestamp;
        hevm.selectFork(fork1);
        emit SwitchedFork(fork1);
        hevm.roll(blockNo);
        hevm.warp(blockTime);
        // Never fail assertion, since there is nothing to compare
        assert(true);
    }


    /*** Tainted Variables ***/ 


    /*** Additional Targets ***/ 

}
