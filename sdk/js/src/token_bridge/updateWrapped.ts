import { ethers, Overrides } from "ethers";
import {
  createWrappedOnAlgorand,
  createWrappedOnSolana,
  createWrappedOnTerra,
  createWrappedOnNear,
  submitVAAOnInjective,
  createWrappedOnXpla,
  createWrappedOnAptos,
} from ".";
import { Bridge__factory } from "../ethers-contracts";

export async function updateWrappedOnEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  signedVAA: Uint8Array,
  overrides: Overrides & { from?: string | Promise<string> } = {}
): Promise<ethers.ContractReceipt> {
  const tx = await updateWrappedOnEthTx(tokenBridgeAddress, signer, signedVAA, overrides);
  const v = await signer.sendTransaction(tx);
  const receipt = await v.wait();
  return receipt;
}

export async function updateWrappedOnEthTx(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  signedVAA: Uint8Array,
  overrides: Overrides & { from?: string | Promise<string> } = {}
): Promise<ethers.PopulatedTransaction> {
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  return bridge.populateTransaction.updateWrapped(signedVAA, overrides);
}

export const updateWrappedOnTerra = createWrappedOnTerra;

export const updateWrappedOnInjective = submitVAAOnInjective;

export const updateWrappedOnXpla = createWrappedOnXpla;

export const updateWrappedOnSolana = createWrappedOnSolana;

export const updateWrappedOnAlgorand = createWrappedOnAlgorand;

export const updateWrappedOnNear = createWrappedOnNear;

export const updateWrappedOnAptos = createWrappedOnAptos;
