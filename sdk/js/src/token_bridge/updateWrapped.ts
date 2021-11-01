import { MsgExecuteContract } from "@terra-money/terra.js";
import { ethers } from "ethers";
import { fromUint8Array } from "js-base64";
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

export async function updateWrappedOnTerra(
  tokenBridgeAddress: string,
  walletAddress: string,
  signedVAA: Uint8Array
) {
  return new MsgExecuteContract(walletAddress, tokenBridgeAddress, {
    submit_vaa: {
      data: fromUint8Array(signedVAA),
    },
  });
}
