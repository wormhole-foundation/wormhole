import { TokenImplementation__factory } from "@certusone/wormhole-sdk";
import { ethers } from "ethers";

export async function getEthereumToken(
  tokenAddress: string,
  provider: ethers.providers.Provider
) {
  // @ts-ignore
  const token = TokenImplementation__factory.connect(tokenAddress, provider);
  return token;
}


