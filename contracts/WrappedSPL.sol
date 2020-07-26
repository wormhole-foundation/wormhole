// contracts/Wormhole.sol
// SPDX-License-Identifier: MIT

pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract WrappedSPLToken is ERC20 {
    constructor(address[] memory _guardians) public {
        require(_guardians.length > 0, "no guardians specified");

        for (uint i = 0; i < _guardians.length; i++) {
            guardians.add(_guardians[i]);
        }
    }
}
