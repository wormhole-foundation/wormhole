// contracts/Structs.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";

contract PythStructs {
    using BytesLib for bytes;

    struct Ema {
        int64 value;
        int64 numerator;
        int64 denominator;
    }

    struct PriceAttestation {
        uint32 magic; // constant "P2WH"
        uint16 version;

        // PayloadID uint8 = 1
        uint8 payloadId;

        bytes32 productId;
        bytes32 priceId;

        uint8 priceType;

        int64 price;
        int32 exponent;

        Ema twap;
        Ema twac;

        uint64 confidenceInterval;

        uint8 status;
        uint8 corpAct;

        uint64 timestamp;
    }

    struct UpgradeContract {
        bytes32 module;
        uint8 action;
        uint16 chain;

        address newContract;
    }
}