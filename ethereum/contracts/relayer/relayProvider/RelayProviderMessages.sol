// contracts/Bridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "../../libraries/external/BytesLib.sol";

import "./RelayProviderGetters.sol";
import "./RelayProviderStructs.sol";

contract RelayProviderMessages is RelayProviderStructs, RelayProviderGetters {
    using BytesLib for bytes;
}
