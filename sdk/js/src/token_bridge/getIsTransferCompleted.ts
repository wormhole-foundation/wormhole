import { ethers } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import { getSignedVAAHash } from "../bridge";
import { importCoreWasm } from "../solana/wasm";
import { Connection, PublicKey } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import axios from "axios";
import { redeemOnTerra } from ".";

export async function getIsTransferCompletedEth(
  tokenBridgeAddress: string,
  provider: ethers.Signer | ethers.providers.Provider,
  signedVAA: Uint8Array
) {
  const tokenBridge = Bridge__factory.connect(tokenBridgeAddress, provider);
  const signedVAAHash = await getSignedVAAHash(signedVAA);
  return await tokenBridge.isTransferCompleted(signedVAAHash);
}

export async function getIsTransferCompletedTerra(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  walletAddress: string,
  client: LCDClient,
  gasPriceUrl: string
) {
  const msg = await redeemOnTerra(tokenBridgeAddress, walletAddress, signedVAA);
  // TODO: remove gasPriceUrl and just use the client's gas prices
  const gasPrices = await axios.get(gasPriceUrl).then((result) => result.data);
  try {
    await client.tx.estimateFee(walletAddress, [msg], {
      memo: "already redeemed calculation",
      feeDenoms: ["uluna"],
      gasPrices,
    });
  } catch (e) {
    // redeemed if the VAA was already executed
    return e.response.data.error.includes("VaaAlreadyExecuted");
  }
  return false;
}

export async function getIsTransferCompletedSolana(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  connection: Connection
) {
  const { claim_address } = await importCoreWasm();
  const claimAddress = await claim_address(tokenBridgeAddress, signedVAA);
  const claimInfo = await connection.getAccountInfo(
    new PublicKey(claimAddress),
    "confirmed"
  );
  return !!claimInfo;
}
