// contracts/WrappedAsset.sol
// SPDX-License-Identifier: Apache 2
pragma solidity ^0.6.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract WrappedAsset is ERC20("Wormhole Wrapped Asset", "WASSET") {
    uint8 public assetChain;
    bytes32 public assetAddress;
    bool public initialized;
    address public bridge;

    function initialize(uint8 _assetChain, bytes32 _assetAddress) public {
        require(!initialized, "already initialized");
        // Set local fields
        assetChain = _assetChain;
        assetAddress = _assetAddress;
        bridge = msg.sender;
        initialized = true;
    }

    function mint(address account, uint256 amount) external {
        require(msg.sender == bridge, "mint can only be called by the bridge");

        super._mint(account, amount);
    }

    function burn(address account, uint256 amount) external {
        require(msg.sender == bridge, "burn can only be called by the bridge");

        super._burn(account, amount);
    }
}
