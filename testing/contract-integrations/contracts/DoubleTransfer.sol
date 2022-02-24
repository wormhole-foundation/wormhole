pragma solidity ^0.6.7;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/math/SafeMath.sol";

interface TokenBridge {
  function transferTokens(address token, uint256 amount, uint16 recipientChain, bytes32 recipient, uint256 arbiterFee, uint32 nonce) external payable returns (uint64);
}

//https://github.com/scaffold-eth/scaffold-eth/blob/mvp-dex/packages/hardhat/contracts/YourDEX.sol
contract DoubleTransfer {

    using SafeMath for uint256;

    string public purpose = "Double Transfer";
    IERC20 token;

    constructor(address tokenAddress) public {
      token = IERC20(tokenAddress);
    }

    function transferTwice(uint256 amount, address _address, uint16 targetChain, bytes32 targetAddress, uint256 fee, uint32 nonce1, uint32 nonce2) public returns (uint256, uint256) {
      require(token.transferFrom(msg.sender, address(this), amount));
      uint256 remainder = amount - 1;
      TokenBridge(_address).transferTokens(address(token), 1, targetChain,  targetAddress, fee, nonce1);
      TokenBridge(_address).transferTokens(address(token), remainder, targetChain,  targetAddress, fee, nonce2);

      return (0,0);
    }

}