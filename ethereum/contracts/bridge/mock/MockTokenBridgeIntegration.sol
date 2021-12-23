// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;


import '@openzeppelin/contracts/token/ERC20/IERC20.sol';
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

import "../../libraries/external/BytesLib.sol";
import "../../interfaces/IWormhole.sol";

interface ITokenBridge {
    function completeTransferWithPayload(bytes memory encodedVm) external returns (IWormhole.VM memory);
    function wrappedAsset(uint16 tokenChainId, bytes32 tokenAddress) external view returns (address);
}

contract MockTokenBridgeIntegration {
    using BytesLib for bytes;
    using SafeERC20 for IERC20;
    address tokenBridgeAddress;
    function completeTransferAndSwap(bytes memory encodedVm) public {
        // TODO: check type = 3
        // token bridge transfers are 133 bytes, our additional payload is 32 bytes = 165
        // len - 165 + 33 = len - 132
        bytes32 tokenAddress = encodedVm.toBytes32(encodedVm.length-132);
        // len - 165 + 65 = len - 100
        uint16 tokenChainId = encodedVm.toUint16(encodedVm.length-100);
        address wrappedAddress = tokenBridge().wrappedAsset(tokenChainId, tokenAddress);
        IERC20 transferToken = IERC20(wrappedAddress);
        uint256 balanceBefore = transferToken.balanceOf(address(this));
        IWormhole.VM memory vm = tokenBridge().completeTransferWithPayload(encodedVm);
        bytes32 vmTokenAddress = vm.payload.toBytes32(33);
        require(tokenAddress == vmTokenAddress, 'Address parsed from VAA and payload do not match');
        uint16 vmTokenChainId = vm.payload.toUint16(65);
        require(tokenChainId == vmTokenChainId, 'ChainId parsed from VAA and payload do not match');
        uint256 balanceAfter = transferToken.balanceOf(address(this));
        uint256 amount = balanceAfter - balanceBefore;
        // TODO: fee?
        // additional field(s)
        bytes32 receiver = vm.payload.toBytes32(133);
        address receiverAddress = address(uint160(uint256(receiver)));
        transferToken.safeTransfer(receiverAddress, amount);
    }
    function tokenBridge() private view returns (ITokenBridge) {
        return ITokenBridge(tokenBridgeAddress);
    }
    function setup(address _tokenBridge) public {
        tokenBridgeAddress = _tokenBridge;
    }
}
