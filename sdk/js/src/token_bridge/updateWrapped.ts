import { ethers } from "ethers";
import { createWrappedOnSolana, createWrappedOnTerra } from ".";
import { Bridge__factory } from "../ethers-contracts";

export async function updateWrappedOnEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  signedVAA: Uint8Array
) {
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.updateWrapped(signedVAA);
  const receipt = await v.wait();
  return receipt;
}

export const updateWrappedOnTerra = createWrappedOnTerra;

export const updateWrappedOnSolana = createWrappedOnSolana;
