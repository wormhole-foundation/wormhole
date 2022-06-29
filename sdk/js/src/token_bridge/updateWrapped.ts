import { ethers, Overrides } from "ethers";
import {
  createWrappedOnAlgorand,
  createWrappedOnInjective,
  createWrappedOnSolana,
  createWrappedOnTerra,
} from ".";
import { Bridge__factory } from "../ethers-contracts";

export async function updateWrappedOnEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  signedVAA: Uint8Array,
  overrides: Overrides & { from?: string | Promise<string> } = {}
) {
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.updateWrapped(signedVAA, overrides);
  const receipt = await v.wait();
  return receipt;
}

export const updateWrappedOnTerra = createWrappedOnTerra;

export const updateWrappedOnInjective = createWrappedOnInjective;

export const updateWrappedOnSolana = createWrappedOnSolana;

export const updateWrappedOnAlgorand = createWrappedOnAlgorand;
