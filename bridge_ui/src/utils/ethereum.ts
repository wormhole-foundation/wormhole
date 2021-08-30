import {
  TokenImplementation,
  TokenImplementation__factory,
} from "@certusone/wormhole-sdk";
import { ethers } from "ethers";
import { formatUnits } from "ethers/lib/utils";
import { createParsedTokenAccount } from "../hooks/useGetSourceParsedTokenAccounts";

//This is a valuable intermediate step to the parsed token account, as the token has metadata information on it.
export async function getEthereumToken(
  tokenAddress: string,
  provider: ethers.providers.Web3Provider
) {
  const token = TokenImplementation__factory.connect(tokenAddress, provider);
  return token;
}

export async function ethTokenToParsedTokenAccount(
  token: TokenImplementation,
  signerAddress: string
) {
  const decimals = await token.decimals();
  const balance = await token.balanceOf(signerAddress);
  return createParsedTokenAccount(
    signerAddress,
    token.address,
    balance.toString(),
    decimals,
    Number(formatUnits(balance, decimals)),
    formatUnits(balance, decimals)
  );
}

export function isValidEthereumAddress(address: string) {
  return ethers.utils.isAddress(address);
}
