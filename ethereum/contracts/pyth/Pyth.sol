// contracts/Bridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";

import "./PythGetters.sol";
import "./PythSetters.sol";
import "./PythStructs.sol";
import "./PythGovernance.sol";

contract Pyth is PythGovernance {
    using BytesLib for bytes;

    function attestPrice(bytes memory encodedVm) public returns (PythStructs.PriceAttestation memory pa) {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVm);

        require(valid, reason);
        require(verifyPythVM(vm), "invalid emitter");

        PythStructs.PriceAttestation memory price = parsePriceAttestation(vm.payload);

        PythStructs.PriceAttestation memory latestPrice = latestAttestation(price.productId, price.priceType);

        if(price.timestamp > latestPrice.timestamp) {
            setLatestAttestation(price.productId, price.priceType, price);
        }

        return price;
    }

    function verifyPythVM(IWormhole.VM memory vm) public view returns (bool valid) {
        if (vm.emitterChainId != pyth2WormholeChainId()) {
            return false;
        }
        if (vm.emitterAddress != pyth2WormholeEmitter()) {
            return false;
        }
        return true;
    }

    function parsePriceAttestation(bytes memory encodedPriceAttestation) public pure returns (PythStructs.PriceAttestation memory pa) {
        uint index = 0;

        pa.magic = encodedPriceAttestation.toUint32(index);
        index += 4;
        require(pa.magic == 0x50325748, "invalid protocol");

        pa.version = encodedPriceAttestation.toUint16(index);
        index += 2;
        require(pa.version == 1, "invalid protocol");

        pa.payloadId = encodedPriceAttestation.toUint8(index);
        index += 1;
        require(pa.payloadId == 1, "invalid PriceAttestation");

        pa.productId = encodedPriceAttestation.toBytes32(index);
        index += 32;
        pa.priceId = encodedPriceAttestation.toBytes32(index);
        index += 32;

        pa.priceType = encodedPriceAttestation.toUint8(index);
        index += 1;

        pa.price = int64(encodedPriceAttestation.toUint64(index));
        index += 8;
        pa.exponent = int32(encodedPriceAttestation.toUint32(index));
        index += 4;

        pa.twap.value = int64(encodedPriceAttestation.toUint64(index));
        index += 8;
        pa.twap.numerator = int64(encodedPriceAttestation.toUint64(index));
        index += 8;
        pa.twap.denominator = int64(encodedPriceAttestation.toUint64(index));
        index += 8;

        pa.twac.value = int64(encodedPriceAttestation.toUint64(index));
        index += 8;
        pa.twac.numerator = int64(encodedPriceAttestation.toUint64(index));
        index += 8;
        pa.twac.denominator = int64(encodedPriceAttestation.toUint64(index));
        index += 8;

        pa.confidenceInterval = encodedPriceAttestation.toUint64(index);
        index += 8;

        pa.status = encodedPriceAttestation.toUint8(index);
        index += 1;
        pa.corpAct = encodedPriceAttestation.toUint8(index);
        index += 1;

        pa.timestamp = encodedPriceAttestation.toUint64(index);
        index += 8;

        require(encodedPriceAttestation.length == index, "invalid PriceAttestation");
    }
}
