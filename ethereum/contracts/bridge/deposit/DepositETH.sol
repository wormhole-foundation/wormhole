// contracts/deposit/Deposit.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

// This contract is used to drain a deposit address of its ETH balance
contract DepositETH {
    constructor (address token)
    {
        selfdestruct(payable(msg.sender));
    }
}