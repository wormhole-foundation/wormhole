// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/GovernanceStructs.sol";
import "forge-test/rv-helpers/TestUtils.sol";

contract TestGovernanceStructs is TestUtils {
    GovernanceStructs gs;

    function setUp() public {
        gs = new GovernanceStructs();
    }

    function testParseContractUpgrade(
        bytes32 module,
        uint16 chain,
        bytes32 newContract
    ) public {
        uint8 action = 1;

        bytes memory encodedUpgrade = abi.encodePacked(
            module,
            action,
            chain,
            newContract
        );

        assertEq(encodedUpgrade.length, 67);
        
        GovernanceStructs.ContractUpgrade memory cu =
            gs.parseContractUpgrade(encodedUpgrade);

        assertEq(cu.module, module);
        assertEq(cu.action, action);
        assertEq(cu.chain, chain);
        assertEq(cu.newContract, address(uint160(uint256(newContract))));
    }

    function testParseContractUpgrade_KEVM(
        bytes32 module,
        uint16 chain,
        bytes32 newContract
    ) public symbolic(address(gs)) {
        testParseContractUpgrade(module, chain, newContract);
    }

    function testParseContractUpgradeWrongAction(
        bytes32 module,
        uint8 action,
        uint16 chain,
        bytes32 newContract
    ) public {
        vm.assume(action != 1);

        bytes memory encodedUpgrade = abi.encodePacked(
            module,
            action,
            chain,
            newContract
        );

        assertEq(encodedUpgrade.length, 67);
        
        vm.expectRevert("invalid ContractUpgrade");

        gs.parseContractUpgrade(encodedUpgrade);
    }

    function testParseContractUpgradeWrongAction_KEVM(
        bytes32 module,
        uint8 action,
        uint16 chain,
        bytes32 newContract
    ) public symbolic(address(gs)) {
        testParseContractUpgradeWrongAction(module, action, chain, newContract);
    }

    // Needs loop invariant for unbounded bytes type
    function testParseContractUpgradeSizeTooSmall(bytes memory encodedUpgrade)
        public
    {
        vm.assume(encodedUpgrade.length < 67);

        if (32 < encodedUpgrade.length)
            encodedUpgrade[32] = bytes1(0x01); // ensure correct action

        vm.expectRevert();

        gs.parseContractUpgrade(encodedUpgrade);
    }

    // Needs loop invariant for unbounded bytes type
    function testParseContractUpgradeSizeTooLarge(
        bytes32 module,
        uint16 chain,
        bytes32 newContract,
        bytes memory extraBytes
    ) public {
        vm.assume(0 < extraBytes.length);

        uint8 action = 1;

        bytes memory encodedUpgrade = abi.encodePacked(
            module,
            action,
            chain,
            newContract,
            extraBytes
        );

        assertGt(encodedUpgrade.length, 67);

        vm.expectRevert("invalid ContractUpgrade");

        gs.parseContractUpgrade(encodedUpgrade);
    }

    function testParseSetMessageFee(
        bytes32 module,
        uint16 chain,
        uint256 messageFee
    ) public {
        uint8 action = 3;

        bytes memory encodedSetMessageFee = abi.encodePacked(
            module,
            action,
            chain,
            messageFee
        );

        assertEq(encodedSetMessageFee.length, 67);

        GovernanceStructs.SetMessageFee memory smf =
            gs.parseSetMessageFee(encodedSetMessageFee);

        assertEq(smf.module, module);
        assertEq(smf.action, action);
        assertEq(smf.chain, chain);
        assertEq(smf.messageFee, messageFee);
    }

    function testParseSetMessageFee_KEVM(
        bytes32 module,
        uint16 chain,
        uint256 messageFee
    ) public symbolic(address(gs)) {
        testParseSetMessageFee(module, chain, messageFee);
    }

    function testParseSetMessageFeeWrongAction(
        bytes32 module,
        uint8 action,
        uint16 chain,
        uint256 messageFee
    ) public {
        vm.assume(action != 3);

        bytes memory encodedSetMessageFee = abi.encodePacked(
            module,
            action,
            chain,
            messageFee
        );

        assertEq(encodedSetMessageFee.length, 67);

        vm.expectRevert("invalid SetMessageFee");

        gs.parseSetMessageFee(encodedSetMessageFee);
    }

    function testParseSetMessageFeeWrongAction_KEVM(
        bytes32 module,
        uint8 action,
        uint16 chain,
        uint256 messageFee
    ) public symbolic(address(gs)) {
        testParseSetMessageFeeWrongAction(module, action, chain, messageFee);
    }

    // Needs loop invariant for unbounded bytes type
    function testParseSetMessageFeeSizeTooSmall(bytes memory encodedSetMessageFee)
        public
    {
        vm.assume(encodedSetMessageFee.length < 67);

        if (32 < encodedSetMessageFee.length)
            encodedSetMessageFee[32] = bytes1(0x03); // ensure correct action

        vm.expectRevert();

        gs.parseSetMessageFee(encodedSetMessageFee);
    }

    // Needs loop invariant for unbounded bytes type
    function testParseSetMessageFeeSizeTooLarge(
        bytes32 module,
        uint16 chain,
        uint256 messageFee,
        bytes memory extraBytes
    ) public {
        vm.assume(0 < extraBytes.length);

        uint8 action = 3;

        bytes memory encodedSetMessageFee = abi.encodePacked(
            module,
            action,
            chain,
            messageFee,
            extraBytes
        );

        assertGt(encodedSetMessageFee.length, 67);

        vm.expectRevert("invalid SetMessageFee");

        gs.parseSetMessageFee(encodedSetMessageFee);
    }

    function testParseTransferFees(
        bytes32 module,
        uint16 chain,
        uint256 amount,
        bytes32 recipient
    ) public {
        uint8 action = 4;

        bytes memory encodedTransferFees = abi.encodePacked(
            module,
            action,
            chain,
            amount,
            recipient
        );

        assertEq(encodedTransferFees.length, 99);

        GovernanceStructs.TransferFees memory tf =
            gs.parseTransferFees(encodedTransferFees);

        assertEq(tf.module, module);
        assertEq(tf.action, action);
        assertEq(tf.chain, chain);
        assertEq(tf.amount, amount);
        assertEq(tf.recipient, recipient);
    }

    function testParseTransferFees_KEVM(
        bytes32 module,
        uint16 chain,
        uint256 amount,
        bytes32 recipient
    ) public symbolic(address(gs)) {
        testParseTransferFees(module, chain, amount, recipient);
    }

    function testParseTransferFeesWrongAction(
        bytes32 module,
        uint8 action,
        uint16 chain,
        uint256 amount,
        bytes32 recipient
    ) public {
        vm.assume(action != 4);

        bytes memory encodedTransferFees = abi.encodePacked(
            module,
            action,
            chain,
            amount,
            recipient
        );

        assertEq(encodedTransferFees.length, 99);

        vm.expectRevert("invalid TransferFees");

        gs.parseTransferFees(encodedTransferFees);
    }

    function testParseTransferFeesWrongAction_KEVM(
        bytes32 module,
        uint8 action,
        uint16 chain,
        uint256 amount,
        bytes32 recipient
    ) public symbolic(address(gs)) {
        testParseTransferFeesWrongAction(module, action, chain, amount, recipient);
    }

    // Needs loop invariant for unbounded bytes type
    function testParseTransferFeesSizeTooSmall(bytes memory encodedTransferFees)
        public
    {
        vm.assume(encodedTransferFees.length < 99);

        if (32 < encodedTransferFees.length)
            encodedTransferFees[32] = bytes1(0x04); // ensure correct action

        vm.expectRevert();

        gs.parseTransferFees(encodedTransferFees);
    }

    // Needs loop invariant for unbounded bytes type
    function testParseTransferFeesSizeTooLarge(
        bytes32 module,
        uint16 chain,
        uint256 amount,
        bytes32 recipient,
        bytes memory extraBytes
    ) public {
        vm.assume(0 < extraBytes.length);

        uint8 action = 4;

        bytes memory encodedTransferFees = abi.encodePacked(
            module,
            action,
            chain,
            amount,
            recipient,
            extraBytes
        );

        assertGt(encodedTransferFees.length, 99);

        vm.expectRevert("invalid TransferFees");

        gs.parseTransferFees(encodedTransferFees);
    }

    function testParseRecoverChainId(
        bytes32 module,
        uint256 evmChainId,
        uint16 newChainId
    ) public {
        uint8 action = 5;

        bytes memory encodedRecoverChainId = abi.encodePacked(
            module,
            action,
            evmChainId,
            newChainId
        );

        assertEq(encodedRecoverChainId.length, 67);

        GovernanceStructs.RecoverChainId memory rci =
            gs.parseRecoverChainId(encodedRecoverChainId);

        assertEq(rci.module, module);
        assertEq(rci.action, action);
        assertEq(rci.evmChainId, evmChainId);
        assertEq(rci.newChainId, newChainId);
    }

    function testParseRecoverChainId_KEVM(
        bytes32 module,
        uint256 evmChainId,
        uint16 newChainId
    ) public symbolic(address(gs)) {
        testParseRecoverChainId(module, evmChainId, newChainId);
    }

    function testParseRecoverChainIdWrongAction(
        bytes32 module,
        uint8 action,
        uint256 evmChainId,
        uint16 newChainId
    ) public {
        vm.assume(action != 5);

        bytes memory encodedRecoverChainId = abi.encodePacked(
            module,
            action,
            evmChainId,
            newChainId
        );

        assertEq(encodedRecoverChainId.length, 67);

        vm.expectRevert("invalid RecoverChainId");

        gs.parseRecoverChainId(encodedRecoverChainId);
    }

    function testParseRecoverChainIdWrongAction_KEVM(
        bytes32 module,
        uint8 action,
        uint256 evmChainId,
        uint16 newChainId
    ) public symbolic(address(gs)) {
        testParseRecoverChainIdWrongAction(module, action, evmChainId, newChainId);
    }

    // Needs loop invariant for unbounded bytes type
    function testParseRecoverChainIdSizeTooSmall(bytes memory encodedRecoverChainId)
        public
    {
        vm.assume(encodedRecoverChainId.length < 67);

        if (32 < encodedRecoverChainId.length)
            encodedRecoverChainId[32] = bytes1(0x05); // ensure correct action

        vm.expectRevert();

        gs.parseRecoverChainId(encodedRecoverChainId);
    }

    // Needs loop invariant for unbounded bytes type
    function testParseRecoverChainIdSizeTooLarge(
        bytes32 module,
        uint256 evmChainId,
        uint16 newChainId,
        bytes memory extraBytes
    ) public {
        vm.assume(0 < extraBytes.length);

        uint8 action = 5;

        bytes memory encodedRecoverChainId = abi.encodePacked(
            module,
            action,
            evmChainId,
            newChainId,
            extraBytes
        );

        assertGt(encodedRecoverChainId.length, 67);

        vm.expectRevert("invalid RecoverChainId");

        gs.parseRecoverChainId(encodedRecoverChainId);
    }
}
