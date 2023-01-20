// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.6.0 <0.9.0;
 
import './HederaResponseCodes.sol';
import './IHederaTokenService.sol';
import './HederaTokenService.sol';
import './ExpiryHelper.sol';
 
contract TokenCreator is HederaTokenService, ExpiryHelper{
 
   // create a fungible Token with no custom fees
   function createFungible(
       string memory name,
       string memory symbol,
       uint256 initialSupply,
       uint256 decimals,
       uint32 autoRenewPeriod
   ) external payable returns (address createdTokenAddress) {
 
       IHederaTokenService.HederaToken memory token;
       token.name = name;
       token.symbol = symbol;
       token.treasury = address(this);
 
       // create the expiry schedule for the token using ExpiryHelper
       token.expiry = createAutoRenewExpiry(address(this), autoRenewPeriod);
 
       // call HTS precompiled contract, passing initial supply and decimals
       (int responseCode, address tokenAddress) =
                   HederaTokenService.createFungibleToken(token, initialSupply, decimals);
 
       if (responseCode != HederaResponseCodes.SUCCESS) {
           revert ();
       }
 
       createdTokenAddress = tokenAddress;
   }
 
}
