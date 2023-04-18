import {
  fromB64,
  normalizeSuiAddress,
  normalizeSuiObjectId,
  TransactionBlock,
} from "@mysten/sui.js";
import { SuiBuildOutput } from "./types";

export const publishCoin = async (
  coreBridgeAddress: string,
  tokenBridgeAddress: string,
  vaa: string,
  signerAddress: string
) => {
  const build = getCoinBuildOutput(coreBridgeAddress, tokenBridgeAddress, vaa);
  return publishPackage(build, signerAddress);
};

export const getCoinBuildOutput = (
  coreBridgeAddress: string,
  tokenBridgeAddress: string,
  vaa: string
): SuiBuildOutput => {
  const bytecode =
    "oRzrCwYAAAAKAQAIAggOAxYWBCwEBTAoB1iGAQjeAWAGvgLlAQqjBAUMqAQWAAMBCgELAgQAAAIAAgECAAMCDAEAAQAGAAEAAQgIAQEMAgkFBgADBwMEAQIDAgEHAggABwgBAAEIAAMJAAoCBwgBAQsCAQkAAQYIAQEFAQsCAQgAAgkABQRDT0lOCVR4Q29udGV4dBFXcmFwcGVkQXNzZXRTZXR1cARjb2luDmNyZWF0ZV93cmFwcGVkC2R1bW15X2ZpZWxkBGluaXQUcHJlcGFyZV9yZWdpc3RyYXRpb24PcHVibGljX3RyYW5zZmVyBnNlbmRlcgh0cmFuc2Zlcgp0eF9jb250ZXh0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAjVFfai/LOUUAUvqTWa3Rh8p5xhe/wYJfCRRBlwG9pGyCgLhAd8B" +
    Buffer.from(vaa, "hex").toString("base64").replace(/\=+$/, "") +
    "ACAQUBAAAAAAEJCwAHAAoBOAALAS4RAjgBAgA=";
  return {
    modules: [bytecode],
    dependencies: [
      normalizeSuiAddress("0x1"),
      normalizeSuiAddress("0x2"),
      tokenBridgeAddress,
      coreBridgeAddress,
    ],
  };
};

export const publishPackage = async (
  buildOutput: SuiBuildOutput,
  signerAddress: string
): Promise<TransactionBlock> => {
  // Publish contracts
  const tx = new TransactionBlock();
  const [upgradeCap] = tx.publish({
    modules: buildOutput.modules.map((m: string) => Array.from(fromB64(m))),
    dependencies: buildOutput.dependencies.map((d: string) =>
      normalizeSuiObjectId(d)
    ),
  });

  // Transfer upgrade capability to recipient
  tx.transferObjects([upgradeCap], tx.pure(signerAddress));
  return tx;
};
