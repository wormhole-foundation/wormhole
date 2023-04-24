// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

contract CoreRelayerProxy is ERC1967Proxy {
    constructor(address implementation) ERC1967Proxy(implementation, new bytes(0)) {}
}
