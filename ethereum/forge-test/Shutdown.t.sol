// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/Implementation.sol";
import "../contracts/Governance.sol";
import "../contracts/Setup.sol";
import "../contracts/Shutdown.sol";
import "../contracts/Wormhole.sol";
import "forge-test/rv-helpers/TestUtils.sol";
import "forge-test/rv-helpers/MyImplementation.sol";
import "forge-test/rv-helpers/IMyWormhole.sol";

contract TestShutdown is TestUtils {

    MyImplementation impl;

    Wormhole proxy;
    Setup setup;
    Setup proxiedSetup;
    IMyWormhole proxied;

    uint256 constant testGuardian = 93941733246223705020089879371323733820373732307041878556247502674739205313440;
    bytes32 constant governanceContract = 0x0000000000000000000000000000000000000000000000000000000000000004;

    function setUp() public {
        vm.chainId(1);
        // Deploy setup
        setup = new Setup();
        // Deploy implementation contract
        impl = new MyImplementation(1,2);
        // Deploy proxy
        proxy = new Wormhole(address(setup), bytes(""));

        address[] memory keys = new address[](1);
        keys[0] = 0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe;
        //keys[0] = vm.addr(testGuardian);

        //proxied setup
        proxiedSetup = Setup(address(proxy));

        proxiedSetup.setup({
            implementation: address(impl),
            initialGuardians: keys,
            chainId: 2,
            governanceChainId: 1,
            governanceContract: governanceContract,
            evmChainId: 1
        });

        proxied = IMyWormhole(address(proxy));
        upgradeImplementation();
    }

    function upgradeImplementation() internal {
        vm.chainId(1);

        Shutdown shutdn = new Shutdown();
        bytes memory payload = payloadSubmitContract(
            0x00000000000000000000000000000000000000000000000000000000436f7265,
            2,
            address(shutdn)
        );
        (bytes memory _vm, ) = validVm(
            0, 0, 0, 1, governanceContract, 0, 0, payload, testGuardian);

        proxied.submitContractUpgrade(_vm);
    }

    function testShutdownInit(address alice, bytes32 storageSlot)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.prank(alice);
        proxied.initialize();
    }

    function testShutdown_publishMessage_revert(address alice, bytes32 storageSlot, uint32 nonce, bytes memory payload, uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.prank(alice);
        vm.expectRevert();
        proxied.publishMessage(nonce,payload,consistencyLevel);
    }
}
