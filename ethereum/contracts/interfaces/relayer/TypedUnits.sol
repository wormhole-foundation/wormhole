// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "forge-std/console.sol";

type WeiPrice is uint64;

type GasPrice is uint88;

type Gas is uint64;

type Dollar is uint256;

type Wei is uint256;

using {
    addWei as +, subWei as -, lteWei as <=, ltWei as <, gtWei as >, eqWei as ==
} for Wei global;
using {ltGas as <, subGas as -} for Gas global;

using WeiLib for Wei;
using GasLib for Gas;
using DollarLib for Dollar;
using WeiPriceLib for WeiPrice;
using GasPriceLib for GasPrice;

function ltWei(Wei a, Wei b) pure returns (bool) {
    return Wei.unwrap(a) < Wei.unwrap(b);
}

function eqWei(Wei a, Wei b) pure returns (bool) {
    return Wei.unwrap(a) == Wei.unwrap(b);
}

function gtWei(Wei a, Wei b) pure returns (bool) {
    return Wei.unwrap(a) > Wei.unwrap(b);
}

function lteWei(Wei a, Wei b) pure returns (bool) {
    return Wei.unwrap(a) <= Wei.unwrap(b);
}

function subWei(Wei a, Wei b) pure returns (Wei) {
    return Wei.wrap(Wei.unwrap(a) - Wei.unwrap(b));
}

function addWei(Wei a, Wei b) pure returns (Wei) {
    return Wei.wrap(Wei.unwrap(a) + Wei.unwrap(b));
}

function ltGas(Gas a, Gas b) pure returns (bool) {
    return Gas.unwrap(a) < Gas.unwrap(b);
}

function subGas(Gas a, Gas b) pure returns (Gas) {
    return Gas.wrap(Gas.unwrap(a) - Gas.unwrap(b));
}

library WeiLib {
    using {toDollars, toGas, convertAsset, min, scale, unwrap, toGasU256} for Wei;

    Wei constant ZERO = Wei.wrap(0);

    function zero() internal pure returns (Wei) {
        return Wei.wrap(0);
    }

    function isU128(Wei x) internal pure returns (Wei) {
        require(Wei.unwrap(x) <= type(uint128).max, "Wei must be < u128");
        return x;
    }

    function min(Wei x, Wei maxVal) internal pure returns (Wei) {
        return x > maxVal ? maxVal : x;
    }

    function toDollars(Wei w, WeiPrice price) internal pure returns (Dollar) {
        return Dollar.wrap(Wei.unwrap(w) * WeiPrice.unwrap(price));
    }

    function toGasU256(Wei w, GasPrice price) internal view returns (uint256) {
        return Wei.unwrap(w) / GasPrice.unwrap(price);
    }

    function toGas(Wei w, GasPrice price) internal view returns (Gas) {
        return GasLib.gas(Wei.unwrap(w) / GasPrice.unwrap(price));
    }

    function scale(Wei w, Gas num, Gas denom) internal pure returns (Wei) {
        return Wei.wrap(Wei.unwrap(w) * Gas.unwrap(num) / Gas.unwrap(denom));
    }

    function unwrap(Wei w) internal pure returns (uint256) {
        return Wei.unwrap(w);
    }

    function convertAsset(
        Wei w,
        WeiPrice fromPrice,
        WeiPrice toPrice,
        uint32 multiplierNum,
        uint32 multiplierDenom,
        bool roundUp
    ) internal view returns (Wei) {
        console.log("heyo");
        Dollar numerator = w.toDollars(fromPrice).mul(multiplierNum);
        console.log("numerator", numerator.unwrap());
        console.log("multiplierDenom", multiplierDenom);
        console.log("toPrice", toPrice.unwrap());
        Dollar denom = toPrice.toDollar().mul(multiplierDenom);
        console.log("denom", denom.unwrap());
        Wei res = numerator.toWeiFromDollar(denom, roundUp) ;
        console.log("res", res.unwrap());
        return res;
    }
}

library GasLib {
    using {toDollars, toWei, unwrap} for Gas;

    Gas constant ZERO = Gas.wrap(0);

    function gas(uint256 x) internal view returns (Gas) {
        console.log("in 'gas' helper", x);
        require(x <= type(uint64).max, "Gas must be < u64");
        return Gas.wrap(uint64(x));
    }

    function isU32(Gas x) internal pure returns (Gas) {
        require(Gas.unwrap(x) <= type(uint32).max, "Gas must be < u32");
        return x;
    }

    function toU32(Gas x) internal pure returns (uint32) {
        return uint32(Gas.unwrap(x.isU32()));
    }

    function min(Gas x, Gas maxVal) internal pure returns (Gas) {
        return x < maxVal ? x : maxVal;
    }

    function toDollars(
        Gas w,
        GasPrice gasPrice,
        WeiPrice weiPrice
    ) internal pure returns (Dollar) {
        return Wei.wrap(uint256(Gas.unwrap(w)) * GasPrice.unwrap(gasPrice)).toDollars(weiPrice);
    }

    function toWei(Gas w, GasPrice price) internal pure returns (Wei) {
        return Wei.wrap(uint256(Gas.unwrap(w)) * GasPrice.unwrap(price));
    }

    function unwrap(Gas w) internal pure returns (uint256) {
        return Gas.unwrap(w);
    }
}

library DollarLib {
    using {toWeiFromDollar, toWei, toWeiRoundUp, mul, unwrap} for Dollar;

    function dollar(uint128 x) internal pure returns (Dollar) {
        require(x <= type(uint128).max, "Dollar must be < u128");
        return Dollar.wrap(uint256(x));
    }

    function mul(Dollar a, uint256 b) internal pure returns (Dollar) {
        return Dollar.wrap(Dollar.unwrap(a) * b);
    }

    function toWeiFromDollar(Dollar w, Dollar price, bool roundUp) internal pure returns (Wei) {
        uint256 price_ = Dollar.unwrap(price);
        return Wei.wrap((w.unwrap() + (roundUp ? price_ - 1 : 0)) / price_);
    }

    function toWeiRoundUp(Dollar w, WeiPrice price) internal pure returns (Wei) {
        uint256 price_ = WeiPrice.unwrap(price);
        return Wei.wrap((Dollar.unwrap(w) + price_ - 1) / price_);
    }

    function toWei(Dollar w, WeiPrice price) internal pure returns (Wei) {
        return Wei.wrap(Dollar.unwrap(w) / WeiPrice.unwrap(price));
    }

    function toGas(Dollar w, GasPrice price, WeiPrice weiPrice) internal view returns (Gas) {
        return w.toWei(weiPrice).toGas(price);
    }

    function unwrap(Dollar w) internal pure returns (uint256) {
        return Dollar.unwrap(w);
    }
}

library WeiPriceLib {
    using {mul, unwrap, toDollar} for WeiPrice;

    WeiPrice constant ZERO = WeiPrice.wrap(0);

    function toDollar(WeiPrice w) internal pure returns (Dollar) {
        return Dollar.wrap(WeiPrice.unwrap(w));
    }

    function mul(WeiPrice a, uint32 b) internal pure returns (WeiPrice) {
        return WeiPrice.wrap(WeiPrice.unwrap(a) * b);
    }

    function unwrap(WeiPrice w) internal pure returns (uint64) {
        return WeiPrice.unwrap(w);
    }
}

library GasPriceLib {
    using {unwrap, toWei} for GasPrice;

    GasPrice constant ZERO = GasPrice.wrap(0);

    function unwrap(GasPrice w) internal pure returns (uint88) {
        return GasPrice.unwrap(w);
    }

    function toWei(GasPrice w) internal pure returns (Wei) {
        return Wei.wrap(GasPrice.unwrap(w));
    }
}
