// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/bridge/token/TokenImplementation.sol";
import "../contracts/Setters.sol";
import "../contracts/bridge/utils/Migrator.sol";
import "forge-std/Test.sol";

contract TestTokenMigrator is Test {
    TokenImplementation fromToken;
    TokenImplementation toToken;
    uint8 fromDecimals = 8;
    uint8 toDecimals = 18;
    Migrator migrator;

    function setUp() public {
        fromToken = new TokenImplementation();
        fromToken.initialize(
            "TestFrom",
            "FROM",
            fromDecimals,
            0,
            address(this),
            0,
            bytes32(0x0)
        );

        toToken = new TokenImplementation();
        toToken.initialize(
            "TestTo",
            "TO",
            toDecimals,
            0,
            address(this),
            0,
            bytes32(0x0)
        );
        migrator = new Migrator(address(fromToken), address(toToken));

        assertEq(address(migrator.fromAsset()), address(fromToken));
        assertEq(address(migrator.toAsset()), address(toToken));
        assertEq(migrator.fromDecimals(), fromDecimals);
        assertEq(migrator.toDecimals(), toDecimals);
    }

    function testShouldGiveOutLPTokens1to1ForAToTokenDeposit() public {
        uint256 amount = 1000000000000000000;
        toToken.mint(address(this), amount);
        toToken.approve(address(migrator), amount);
        migrator.add(amount);
        assertEq(migrator.balanceOf(address(this)), amount);
        assertEq(toToken.balanceOf(address(migrator)), amount);
    }

    function testShouldRefundToTokenForLPTokens() public {
        testShouldGiveOutLPTokens1to1ForAToTokenDeposit();
        uint256 amount = 500000000000000000;
        migrator.remove(amount);
        assertEq(migrator.balanceOf(address(this)), amount);
        assertEq(toToken.balanceOf(address(migrator)), amount);
        assertEq(toToken.balanceOf(address(this)), amount);
    }

    function testShouldRedeemFromTokenToToTokenAdjustingForDecimals() public {
        testShouldRefundToTokenForLPTokens();
        address newAddr = address(0x1);
        fromToken.mint(newAddr, 50000000);
        vm.prank(newAddr);
        fromToken.approve(address(migrator), 50000000);
        vm.prank(newAddr);
        migrator.migrate(50000000);

        assertEq(toToken.balanceOf(newAddr), 500000000000000000);
        assertEq(toToken.balanceOf(address(migrator)), 0);
        assertEq(fromToken.balanceOf(newAddr), 0);
        assertEq(fromToken.balanceOf(address(migrator)), 50000000);
    }

    function testFromTokenShouldBeClaimableForLPTokensAdjustingForDecimals()
        public
    {
        testShouldRedeemFromTokenToToTokenAdjustingForDecimals();
        migrator.claim(500000000000000000);

        assertEq(fromToken.balanceOf(address(this)), 50000000);
        assertEq(fromToken.balanceOf(address(migrator)), 0);
        assertEq(migrator.balanceOf(address(this)), 0);
    }
}
