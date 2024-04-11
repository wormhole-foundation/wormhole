// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";
import "forge-std/console.sol";
import "../../contracts/interfaces/relayer/TypedUnits.sol";

contract UVDTTest is Test {
    using WeiLib for Wei;
    using GasLib for Gas;
    using DollarLib for Dollar;

    function setUp() public {}

    function testWeiBasic(uint64 x) public pure {
        Wei w = Wei.wrap(x);
        WeiPrice p = WeiPrice.wrap(100);
        Dollar value = w.toDollars(p);

        require(Dollar.unwrap(value) == uint256(x) * 100, "value should be 100*x");
    }

    function testWeiToGas(uint64 x) public pure {
        Wei w = Wei.wrap(x);
        GasPrice p = GasPrice.wrap(100);
        Gas value = w.toGas(p);

        require(Gas.unwrap(value) == uint256(x) / 100, "value should be x/100");
    }

    function testGasToWei(uint64 x) public pure {
        Gas w = Gas.wrap(x);
        GasPrice p = GasPrice.wrap(100);
        Wei value = w.toWei(p);

        require(Wei.unwrap(value) == uint256(x) * 100, "value should be 100*x");
    }

    function convertAsset(
        uint64 source,
        uint64 fromPrice,
        uint64 toPrice,
        uint32 multNum,
        uint32 multDenom,
        bool roundUp
    ) public pure {
        Wei w = Wei.wrap(source);
        WeiPrice fp = WeiPrice.wrap(fromPrice);
        WeiPrice tp = WeiPrice.wrap(toPrice);
        Wei value = w.convertAsset(fp, tp, multNum, multDenom, roundUp);

        uint256 expected;
        if (roundUp) {
            expected = (
                uint256(source) * uint256(fromPrice) * multNum + (uint256(toPrice) * multDenom) - 1
            ) / (uint256(toPrice) * multDenom);
        } else {
            expected =
                (uint256(source) * uint256(fromPrice) * multNum) / (uint256(toPrice) * multDenom);
        }

        require(Wei.unwrap(value) == expected, "value should be expected");
    }

    function sourceWeiToTargetGas(uint64 sourceWei) public pure {
        Wei w = Wei.wrap(sourceWei);

        // gets smaller
        {
            WeiPrice fp = WeiPrice.wrap(10);
            WeiPrice tp = WeiPrice.wrap(100);
            GasPrice gp = GasPrice.wrap(5);
            Gas targetGas = w.convertAsset(fp, tp, 1, 1, false).toGas(gp);
            require(Gas.unwrap(targetGas) == sourceWei / 50, "targetGas should be 2");
        }

        // round up
        {
            WeiPrice fp = WeiPrice.wrap(100);
            WeiPrice tp = WeiPrice.wrap(11);
            GasPrice gp = GasPrice.wrap(5);
            Gas targetGas = w.convertAsset(fp, tp, 1, 1, true).toGas(gp);
            require(Gas.unwrap(targetGas) == sourceWei, "round down sourceWei * 1.8 => sourceWei");
        }
        // round down
        {
            WeiPrice fp = WeiPrice.wrap(100);
            WeiPrice tp = WeiPrice.wrap(11);
            GasPrice gp = GasPrice.wrap(5);
            Gas targetGas = w.convertAsset(fp, tp, 1, 1, false).toGas(gp);
            require(
                Gas.unwrap(targetGas) == sourceWei * 2, "round up sourceWei * 1.8 => sourceWei * 2"
            );
        }
    }

    function testDollarToWei(uint128 x) public pure {
        Dollar d = Dollar.wrap(x);
        WeiPrice p = WeiPrice.wrap(100);
        Wei value = d.toWei(p, false);

        require(Wei.unwrap(value) == uint256(x) / 100, "value should be x/100");
    }

    function testDollarToGas(uint128 x) public pure {
        Dollar d = Dollar.wrap(x);
        GasPrice gp = GasPrice.wrap(1 << 32);
        WeiPrice wp = WeiPrice.wrap(1 << 32);
        Gas value = d.toGas(gp, wp);

        require(Gas.unwrap(value) == uint256(x) / (1 << 64), "value should be x/(1<<32)");
    }

    function testGasMin(uint64 x, uint64 y) public pure {
        Gas a = Gas.wrap(x);
        Gas b = Gas.wrap(y);
        Gas minVal = a.min(b);

        require(Gas.unwrap(minVal) == (x < y ? x : y), "minVal should be min(x, y)");
    }
}
