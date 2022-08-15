// contracts/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../Structs.sol";

interface IWormhole is Structs {
    event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel);

    function publishMessage(
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) external payable returns (uint64 sequence);

    function parseAndVerifyVM(bytes calldata encodedVM) external view returns (Structs.VM memory vm, bool valid, string memory reason);

    function parseAndVerifyVM2(bytes calldata encodedVM) external returns (Structs.VM2 memory vm, bool valid, string memory reason);

    function parseAndVerifyVM3(bytes calldata encodedVM) external view returns (Structs.VM3 memory vm, bool valid, string memory reason);

    function parseAndVerifyVAA(bytes calldata encodedVM) external view returns (Structs.Observation memory observation, bool valid, string memory reason);

    function verifyVM(Structs.VM memory vm) external view returns (bool valid, string memory reason);

    function verifyVM2(Structs.BatchHeader memory vm) external view returns (bool valid, string memory reason);

    function verifySignatures(bytes32 hash, Structs.Signature[] memory signatures, Structs.GuardianSet memory guardianSet) external pure returns (bool valid, string memory reason) ;

    function clearBatchCache(Structs.BatchHeader memory header) external;

    function parseVM(bytes memory encodedVM) external pure returns (Structs.VM memory vm);

    function parseVM2(bytes memory encodedVM)  external pure returns (Structs.VM2 memory vm);

    function parseVM3(bytes memory encodedVM) external pure returns (Structs.VM3 memory vm);

    function getGuardianSet(uint32 index) external view returns (Structs.GuardianSet memory) ;

    function getCurrentGuardianSetIndex() external view returns (uint32) ;

    function getGuardianSetExpiry() external view returns (uint32) ;

    function governanceActionIsConsumed(bytes32 hash) external view returns (bool) ;

    function isInitialized(address impl) external view returns (bool) ;

    function chainId() external view returns (uint16) ;

    function governanceChainId() external view returns (uint16);

    function governanceContract() external view returns (bytes32);

    function messageFee() external view returns (uint256) ;

}
