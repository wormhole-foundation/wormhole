import { ethers } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import { CHAIN_ID_ETH, ETH_TOKEN_BRIDGE_ADDRESS } from "./consts";

export async function redeemOnEth(
  provider: ethers.providers.Web3Provider | undefined,
  signer: ethers.Signer | undefined,
  signedVAA: Uint8Array
) {
  console.log(provider, signer, signedVAA);
  if (!provider || !signer) return;
  console.log("completing transfer");
  const bridge = Bridge__factory.connect(ETH_TOKEN_BRIDGE_ADDRESS, signer);
  const v = await bridge.completeTransfer(signedVAA);
  const receipt = await v.wait();
  console.log(receipt);
}

const redeemOn = {
  [CHAIN_ID_ETH]: redeemOnEth,
};

export default redeemOn;
