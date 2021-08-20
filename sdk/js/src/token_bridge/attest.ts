import { Connection, Keypair, PublicKey, Transaction } from "@solana/web3.js";
import { ethers } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import { getBridgeFeeIx, ixFromRust } from "../solana";
import { createNonce } from "../utils/createNonce";
import { ConnectedWallet as TerraConnectedWallet } from "@terra-money/wallet-provider";
import { MsgExecuteContract } from "@terra-money/terra.js";

export async function attestFromEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  tokenAddress: string
) {
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.attestToken(tokenAddress, createNonce());
  const receipt = await v.wait();
  return receipt;
}

export async function attestFromTerra(
  tokenBridgeAddress: string,
  wallet: TerraConnectedWallet,
  asset: string,
) {
  const nonce = Math.round(Math.random() * 100000);
  return await wallet.post({
    msgs: [
      new MsgExecuteContract(
        wallet.terraAddress,
        tokenBridgeAddress,
        {
          create_asset_meta: {
            asset_address: asset,
            nonce: nonce,
          },
        },
        { uluna: 10000 }
      ),
    ],
    memo: "Create Wrapped",
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
  const { attest_ix } = await import("../solana/token/token_bridge");
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
