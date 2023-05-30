// contracts/Bridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "../../libraries/external/BytesLib.sol";

import "./DeliveryProviderGetters.sol";
import "./DeliveryProviderStructs.sol";

contract DeliveryProviderMessages is DeliveryProviderStructs, DeliveryProviderGetters {
    using BytesLib for bytes;
}
