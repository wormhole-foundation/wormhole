import { Connection, Keypair, PublicKey, Transaction } from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { ethers, PayableOverrides } from "ethers";
import { isNativeDenom } from "..";
import { Bridge__factory } from "../ethers-contracts";
import { getBridgeFeeIx, ixFromRust } from "../solana";
import { importTokenWasm } from "../solana/wasm";
import { createNonce } from "../utils/createNonce";

export async function attestFromEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  tokenAddress: string,
  overrides: PayableOverrides & { from?: string | Promise<string> } = {}
) {
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.attestToken(tokenAddress, createNonce(), overrides);
  const receipt = await v.wait();
  return receipt;
}

export async function attestFromTerra(
  tokenBridgeAddress: string,
  walletAddress: string,
  asset: string
) {
  const nonce = Math.round(Math.random() * 100000);
  const isNativeAsset = isNativeDenom(asset);
  return new MsgExecuteContract(walletAddress, tokenBridgeAddress, {
    create_asset_meta: {
      asset_info: isNativeAsset
        ? {
            native_token: { denom: asset },
          }
        : {
            token: {
              contract_addr: asset,
            },
          },
      nonce: nonce,
    },
  });
}

export async function attestFromSolana(
  connection: Connection,
  bridgeAddress: string,
  tokenBridgeAddress: string,
  payerAddress: string,
  mintAddress: string
) {
  const nonce = createNonce().readUInt32LE(0);
  const transferIx = await getBridgeFeeIx(
    connection,
    bridgeAddress,
    payerAddress
  );
  const { attest_ix } = await importTokenWasm();
  const messageKey = Keypair.generate();
  const ix = ixFromRust(
    attest_ix(
      tokenBridgeAddress,
      bridgeAddress,
      payerAddress,
      messageKey.publicKey.toString(),
      mintAddress,
      nonce
    )
  );
  const transaction = new Transaction().add(transferIx, ix);
  const { blockhash } = await connection.getRecentBlockhash();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  transaction.partialSign(messageKey);
  return transaction;
}
