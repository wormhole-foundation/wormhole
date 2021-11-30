pragma solidity ^0.6.7;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/math/SafeMath.sol";

interface TokenBridge {
  function transferTokens(address token, uint256 amount, uint16 recipientChain, bytes32 recipient, uint256 arbiterFee, uint32 nonce) external payable returns (uint64);
}

//https://github.com/scaffold-eth/scaffold-eth/blob/mvp-dex/packages/hardhat/contracts/YourDEX.sol
contract SimpleDex {

    using SafeMath for uint256;

    string public purpose = "Swapping Rabbits";
    IERC20 rabbits;

    constructor(address tokenAddress) public {
      rabbits = IERC20(tokenAddress);
    }

    uint256 public totalLiquidity;

    mapping (address => uint256) public liquidity;

    function init(uint256 tokens) public payable returns (uint256) {
      require(totalLiquidity==0,"DEX:init - already has liquidity");
      totalLiquidity = address(this).balance;
      liquidity[msg.sender] = totalLiquidity;
      require(rabbits.transferFrom(msg.sender, address(this), tokens));
      return totalLiquidity;
    }

    function price(uint256 input_amount, uint256 input_reserve, uint256 output_reserve) public pure returns (uint256) {
      uint256 input_amount_with_fee = input_amount.mul(997);
      uint256 numerator = input_amount_with_fee.mul(output_reserve);
      uint256 denominator = input_reserve.mul(1000).add(input_amount_with_fee);
      return numerator / denominator;
    }

    function ethToToken() public payable returns (uint256) {
      uint256 token_reserve = rabbits.balanceOf(address(this));
      uint256 tokens_bought = price(msg.value, address(this).balance.sub(msg.value), token_reserve);
      require(rabbits.transfer(msg.sender, tokens_bought));
      return tokens_bought;
    }

    function tokenToEth(uint256 tokens) public returns (uint256) {
      uint256 token_reserve = rabbits.balanceOf(address(this));
      uint256 eth_bought = price(tokens, token_reserve, address(this).balance);
      msg.sender.transfer(eth_bought);
      require(rabbits.transferFrom(msg.sender, address(this), tokens));
      return eth_bought;
    }

    function deposit() public payable returns (uint256) {
      uint256 eth_reserve = address(this).balance.sub(msg.value);
      uint256 token_reserve = rabbits.balanceOf(address(this));
      uint256 token_amount = (msg.value.mul(token_reserve) / eth_reserve).add(1);
      uint256 liquidity_minted = msg.value.mul(totalLiquidity) / eth_reserve;
      liquidity[msg.sender] = liquidity[msg.sender].add(liquidity_minted);
      totalLiquidity = totalLiquidity.add(liquidity_minted);
      require(rabbits.transferFrom(msg.sender, address(this), token_amount));
      return liquidity_minted;
    }

    function withdraw(uint256 amount) public returns (uint256, uint256) {
      uint256 token_reserve = rabbits.balanceOf(address(this));
      uint256 eth_amount = amount.mul(address(this).balance) / totalLiquidity;
      uint256 token_amount = amount.mul(token_reserve) / totalLiquidity;
      liquidity[msg.sender] = liquidity[msg.sender].sub(eth_amount);
      totalLiquidity = totalLiquidity.sub(eth_amount);
      msg.sender.transfer(eth_amount);
      require(rabbits.transfer(msg.sender, token_amount));
      return (eth_amount, token_amount);
    }

    function swapNGo(address _address, uint16 targetChain, bytes32 targetAddress, uint256 fee, uint32 nonce) public payable returns (uint64) {
      uint256 token_reserve = rabbits.balanceOf(address(this));
      uint256 tokens_bought = price(msg.value, address(this).balance.sub(msg.value), token_reserve);
      rabbits.approve(_address, tokens_bought);
      return TokenBridge(_address).transferTokens(address(rabbits), tokens_bought, targetChain,  targetAddress, fee, nonce);
    }

}