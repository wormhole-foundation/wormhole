// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

type WeiPrice is uint256;

type GasPrice is uint256;

type Gas is uint256;

type Dollar is uint256;

type Wei is uint256;

type LocalNative is uint256;

type TargetNative is uint256;

using {
    addWei as +,
    subWei as -,
    lteWei as <=,
    ltWei as <,
    gtWei as >,
    eqWei as ==,
    neqWei as !=
} for Wei global;
using {addTargetNative as +, subTargetNative as -} for TargetNative global;
using {
    leLocalNative as <,
    leqLocalNative as <=,
    neqLocalNative as !=,
    addLocalNative as +,
    subLocalNative as -
} for LocalNative global;
using {
    ltGas as <,
    lteGas as <=,
    subGas as -
} for Gas global;

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

function neqWei(Wei a, Wei b) pure returns (bool) {
    return Wei.unwrap(a) != Wei.unwrap(b);
}

function ltGas(Gas a, Gas b) pure returns (bool) {
    return Gas.unwrap(a) < Gas.unwrap(b);
}

function lteGas(Gas a, Gas b) pure returns (bool) {
    return Gas.unwrap(a) <= Gas.unwrap(b);
}

function subGas(Gas a, Gas b) pure returns (Gas) {
    return Gas.wrap(Gas.unwrap(a) - Gas.unwrap(b));
}

function addTargetNative(TargetNative a, TargetNative b) pure returns (TargetNative) {
    return TargetNative.wrap(TargetNative.unwrap(a) + TargetNative.unwrap(b));
}

function subTargetNative(TargetNative a, TargetNative b) pure returns (TargetNative) {
    return TargetNative.wrap(TargetNative.unwrap(a) - TargetNative.unwrap(b));
}

function addLocalNative(LocalNative a, LocalNative b) pure returns (LocalNative) {
    return LocalNative.wrap(LocalNative.unwrap(a) + LocalNative.unwrap(b));
}

function subLocalNative(LocalNative a, LocalNative b) pure returns (LocalNative) {
    return LocalNative.wrap(LocalNative.unwrap(a) - LocalNative.unwrap(b));
}

function neqLocalNative(LocalNative a, LocalNative b) pure returns (bool) {
    return LocalNative.unwrap(a) != LocalNative.unwrap(b);
}

function leLocalNative(LocalNative a, LocalNative b) pure returns (bool) {
    return LocalNative.unwrap(a) < LocalNative.unwrap(b);
}

function leqLocalNative(LocalNative a, LocalNative b) pure returns (bool) {
    return LocalNative.unwrap(a) <= LocalNative.unwrap(b);
}

library WeiLib {
    using {
        toDollars,
        toGas,
        convertAsset,
        min,
        max,
        scale,
        unwrap,
        asGasPrice,
        asTargetNative,
        asLocalNative
    } for Wei;

    function min(Wei x, Wei maxVal) internal pure returns (Wei) {
        return x > maxVal ? maxVal : x;
    }

    function max(Wei x, Wei maxVal) internal pure returns (Wei) {
        return x < maxVal ? maxVal : x;
    }

    function asTargetNative(Wei w) internal pure returns (TargetNative) {
        return TargetNative.wrap(Wei.unwrap(w));
    }

    function asLocalNative(Wei w) internal pure returns (LocalNative) {
        return LocalNative.wrap(Wei.unwrap(w));
    }

    function toDollars(Wei w, WeiPrice price) internal pure returns (Dollar) {
        return Dollar.wrap(Wei.unwrap(w) * WeiPrice.unwrap(price));
    }

    function toGas(Wei w, GasPrice price) internal pure returns (Gas) {
        return Gas.wrap(Wei.unwrap(w) / GasPrice.unwrap(price));
    }

    function scale(Wei w, Gas num, Gas denom) internal pure returns (Wei) {
        return Wei.wrap(Wei.unwrap(w) * Gas.unwrap(num) / Gas.unwrap(denom));
    }

    function unwrap(Wei w) internal pure returns (uint256) {
        return Wei.unwrap(w);
    }

    function asGasPrice(Wei w) internal pure returns (GasPrice) {
        return GasPrice.wrap(Wei.unwrap(w));
    }

    function convertAsset(
        Wei w,
        WeiPrice fromPrice,
        WeiPrice toPrice,
        uint32 multiplierNum,
        uint32 multiplierDenom,
        bool roundUp
    ) internal pure returns (Wei) {
        Dollar numerator = w.toDollars(fromPrice).mul(multiplierNum);
        WeiPrice denom = toPrice.mul(multiplierDenom);
        Wei res = numerator.toWei(denom, roundUp);
        return res;
    }
}

library GasLib {
    using {toWei, unwrap} for Gas;

    function min(Gas x, Gas maxVal) internal pure returns (Gas) {
        return x < maxVal ? x : maxVal;
    }

    function toWei(Gas w, GasPrice price) internal pure returns (Wei) {
        return Wei.wrap(w.unwrap() * price.unwrap());
    }

    function unwrap(Gas w) internal pure returns (uint256) {
        return Gas.unwrap(w);
    }
}

library DollarLib {
    using {toWei, mul, unwrap} for Dollar;

    function mul(Dollar a, uint256 b) internal pure returns (Dollar) {
        return Dollar.wrap(a.unwrap() * b);
    }

    function toWei(Dollar w, WeiPrice price, bool roundUp) internal pure returns (Wei) {
        return Wei.wrap((w.unwrap() + (roundUp ? price.unwrap() - 1 : 0)) / price.unwrap());
    }

    function toGas(Dollar w, GasPrice price, WeiPrice weiPrice) internal pure returns (Gas) {
        return w.toWei(weiPrice, false).toGas(price);
    }

    function unwrap(Dollar w) internal pure returns (uint256) {
        return Dollar.unwrap(w);
    }
}

library WeiPriceLib {
    using {mul, unwrap} for WeiPrice;

    function mul(WeiPrice a, uint256 b) internal pure returns (WeiPrice) {
        return WeiPrice.wrap(a.unwrap() * b);
    }

    function unwrap(WeiPrice w) internal pure returns (uint256) {
        return WeiPrice.unwrap(w);
    }
}

library GasPriceLib {
    using {unwrap, priceAsWei} for GasPrice;

    function priceAsWei(GasPrice w) internal pure returns (Wei) {
        return Wei.wrap(w.unwrap());
    }

    function unwrap(GasPrice w) internal pure returns (uint256) {
        return GasPrice.unwrap(w);
    }
}

library TargetNativeLib {
    using {unwrap, asNative} for TargetNative;

    function unwrap(TargetNative w) internal pure returns (uint256) {
        return TargetNative.unwrap(w);
    }

    function asNative(TargetNative w) internal pure returns (Wei) {
        return Wei.wrap(TargetNative.unwrap(w));
    }
}

library LocalNativeLib {
    using {unwrap, asNative} for LocalNative;

    function unwrap(LocalNative w) internal pure returns (uint256) {
        return LocalNative.unwrap(w);
    }

    function asNative(LocalNative w) internal pure returns (Wei) {
        return Wei.wrap(LocalNative.unwrap(w));
    }
}
