import {
  NFTImplementation,
  NFTImplementation__factory,
  TokenImplementation,
  TokenImplementation__factory,
} from "@certusone/wormhole-sdk";
import { ethers } from "ethers";
import { arrayify, formatUnits } from "ethers/lib/utils";
import {
  createNFTParsedTokenAccount,
  createParsedTokenAccount,
} from "../hooks/useGetSourceParsedTokenAccounts";

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

//This is a valuable intermediate step to the parsed token account, as the token has metadata information on it.
export async function getEthereumNFT(
  tokenAddress: string,
  provider: ethers.providers.Web3Provider
) {
  const token = NFTImplementation__factory.connect(tokenAddress, provider);
  return token;
}

export async function isNFT(token: NFTImplementation) {
  const erc721 = "0x80ac58cd";
  const erc721metadata = "0x5b5e139f";
  return (
    (await token.supportsInterface(arrayify(erc721))) &&
    (await token.supportsInterface(arrayify(erc721metadata)))
  );
}

export async function ethNFTToNFTParsedTokenAccount(
  token: NFTImplementation,
  tokenId: string,
  signerAddress: string
) {
  const decimals = 0;
  const balance = (await token.ownerOf(tokenId)) === signerAddress ? 1 : 0;
  // const uri = await token.tokenURI(tokenId);
  return createNFTParsedTokenAccount(
    signerAddress,
    token.address,
    balance.toString(),
    decimals,
    Number(formatUnits(balance, decimals)),
    formatUnits(balance, decimals),
    tokenId
  );
}

export function isValidEthereumAddress(address: string) {
  return ethers.utils.isAddress(address);
}
