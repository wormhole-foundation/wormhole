// contracts/Wormhole.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

contract DeliveryProviderProxy is ERC1967Proxy {
    constructor(
        address implementation,
        bytes memory initData
    ) ERC1967Proxy(implementation, initData) {}
}
