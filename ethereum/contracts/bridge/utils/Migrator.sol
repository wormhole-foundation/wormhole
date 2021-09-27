// contracts/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

contract Migrator is ERC20 {
    IERC20 public fromAsset;
    IERC20 public toAsset;
    uint public fromDecimals;
    uint public toDecimals;

    constructor (
        address _fromAsset,
        address _toAsset
    )
    // LP shares track the underlying toToken amount
    ERC20("Token Migration Pool", "Migrator-LP") {
        fromAsset = IERC20(_fromAsset);
        toAsset = IERC20(_toAsset);
        fromDecimals = ERC20(_fromAsset).decimals();
        toDecimals = ERC20(_toAsset).decimals();
    }

    // _amount denominated in toAsset
    function add(uint _amount) external {
        // deposit toAsset
        SafeERC20.safeTransferFrom(toAsset, msg.sender, address(this), _amount);
        // mint LP shares
        _mint(msg.sender, _amount);
    }

    // _amount denominated in LP shares
    function remove(uint _amount) external {
        // burn LP shares
        _burn(msg.sender, _amount);
        // send out toAsset
        SafeERC20.safeTransfer(toAsset, msg.sender, _amount);
    }

    // _amount denominated in LP shares
    function claim(uint _amount) external {
        // burn LP shares
        _burn(msg.sender, _amount);
        // send out fromAsset
        SafeERC20.safeTransfer(fromAsset, msg.sender, adjustDecimals(toDecimals, fromDecimals, _amount));
    }

    // _amount denominated in fromToken
    function migrate(uint _amount) external {
        // deposit fromAsset
        SafeERC20.safeTransferFrom(fromAsset, msg.sender, address(this), _amount);
        // send out toAsset
        SafeERC20.safeTransfer(toAsset, msg.sender, adjustDecimals(fromDecimals, toDecimals, _amount));
    }

    function adjustDecimals(uint _fromDecimals, uint _toDecimals, uint _amount) internal pure returns (uint) {
        if (_fromDecimals > _toDecimals){
            _amount /= 10 ** (_fromDecimals - _toDecimals);
        } else if (_fromDecimals < _toDecimals) {
            _amount *= 10 ** (_toDecimals - _fromDecimals);
        }
        return _amount;
    }
}
