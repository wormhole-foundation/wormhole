import { getNetworkInfo, Network } from "@injectivelabs/networks";
import {
  ChainRestAuthApi,
  DEFAULT_STD_FEE,
  privateKeyToPublicKeyBase64,
} from "@injectivelabs/sdk-ts";
import { createTransaction, TxGrpcClient } from "@injectivelabs/tx-ts";
import { PrivateKey } from "@injectivelabs/sdk-ts/dist/local";
import { Connection, PublicKey } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { Algodv2, bigIntToBytes } from "algosdk";
import axios from "axios";
import { ethers } from "ethers";
import { redeemOnInjective, redeemOnTerra } from ".";
import { TERRA_REDEEMED_CHECK_WALLET_ADDRESS } from "..";
import {
  BITS_PER_KEY,
  calcLogicSigAccount,
  MAX_BITS,
  _parseVAAAlgorand,
} from "../algorand";
import { getSignedVAAHash } from "../bridge";
import { Bridge__factory } from "../ethers-contracts";
import { importCoreWasm } from "../solana/wasm";
import { safeBigIntToNumber } from "../utils/bigint";

export async function getIsTransferCompletedEth(
  tokenBridgeAddress: string,
  provider: ethers.Signer | ethers.providers.Provider,
  signedVAA: Uint8Array
): Promise<boolean> {
  const tokenBridge = Bridge__factory.connect(tokenBridgeAddress, provider);
  const signedVAAHash = await getSignedVAAHash(signedVAA);
  return await tokenBridge.isTransferCompleted(signedVAAHash);
}

export async function getIsTransferCompletedTerra(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  client: LCDClient,
  gasPriceUrl: string
): Promise<boolean> {
  const msg = await redeemOnTerra(
    tokenBridgeAddress,
    TERRA_REDEEMED_CHECK_WALLET_ADDRESS,
    signedVAA
  );
  // TODO: remove gasPriceUrl and just use the client's gas prices
  const gasPrices = await axios.get(gasPriceUrl).then((result) => result.data);
  const account = await client.auth.accountInfo(
    TERRA_REDEEMED_CHECK_WALLET_ADDRESS
  );
  try {
    await client.tx.estimateFee(
      [
        {
          sequenceNumber: account.getSequenceNumber(),
          publicKey: account.getPublicKey(),
        },
      ],
      {
        msgs: [msg],
        memo: "already redeemed calculation",
        feeDenoms: ["uluna"],
        gasPrices,
      }
    );
  } catch (e: any) {
    // redeemed if the VAA was already executed
    return e.response.data.message.includes("VaaAlreadyExecuted");
  }
  return false;
}

// TODO: Is there an Injective address for this?
export async function getIsTransferCompletedInjective(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  walletPKHash: string
): Promise<boolean> {
  try {
    const walletPK = PrivateKey.fromPrivateKey(walletPKHash);
    const walletInjAddr = walletPK.toBech32();
    const walletPublicKey = privateKeyToPublicKeyBase64(
      Buffer.from(walletPKHash, "hex")
    );
    const roi = await redeemOnInjective(
      tokenBridgeAddress,
      walletInjAddr,
      signedVAA
    );
    // TODO:  Remove hardcoded network.
    const network = getNetworkInfo(Network.TestnetK8s);
    const accountDetails = await new ChainRestAuthApi(
      network.sentryHttpApi
    ).fetchAccount(walletInjAddr);
    const txFee = DEFAULT_STD_FEE;
    txFee.amount[0] = { amount: "250000000000000", denom: "inj" };
    txFee.gas = "500000";
    const { signBytes, txRaw } = createTransaction({
      message: roi.toDirectSign(),
      memo: "",
      fee: txFee,
      pubKey: Buffer.from(walletPublicKey).toString("base64"),
      sequence: parseInt(accountDetails.account.base_account.sequence, 10),
      accountNumber: parseInt(
        accountDetails.account.base_account.account_number,
        10
      ),
      chainId: network.chainId,
    });
    console.log("txRaw", txRaw);

    console.log("sign transaction...");
    /** Sign transaction */
    const sig = await walletPK.sign(signBytes);

    /** Append Signatures */
    txRaw.setSignaturesList([sig]);

    const txService = new TxGrpcClient({
      txRaw,
      endpoint: network.sentryGrpcApi,
    });

    console.log("simulate transaction...");
    /** Simulate transaction */
    const simulationResponse = await txService.simulate();
  } catch (e: unknown) {
    if (e instanceof Error) {
      const msgTxt: string = e.message;
      if (msgTxt.includes("VaaAlreadyExecuted")) {
        return true;
      }
      return false;
    }
  }
  return false;
}

export async function getIsTransferCompletedSolana(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  connection: Connection
): Promise<boolean> {
  const { claim_address } = await importCoreWasm();
  const claimAddress = await claim_address(tokenBridgeAddress, signedVAA);
  const claimInfo = await connection.getAccountInfo(
    new PublicKey(claimAddress),
    "confirmed"
  );
  return !!claimInfo;
}

// Algorand

/**
 * This function is used to check if a VAA has been redeemed by looking at a specific bit.
 * @param client AlgodV2 client
 * @param appId Application Id
 * @param addr Wallet address. Someone has to pay for this.
 * @param seq The sequence number of the redemption
 * @returns true, if the bit was set and VAA was redeemed, false otherwise.
 */
async function checkBitsSet(
  client: Algodv2,
  appId: bigint,
  addr: string,
  seq: bigint
): Promise<boolean> {
  let retval: boolean = false;
  let appState: any[] = [];
  const acctInfo = await client.accountInformation(addr).do();
  const als = acctInfo["apps-local-state"];
  als.forEach((app: any) => {
    if (BigInt(app["id"]) === appId) {
      appState = app["key-value"];
    }
  });
  if (appState.length === 0) {
    return retval;
  }

  const BIG_MAX_BITS: bigint = BigInt(MAX_BITS);
  const BIG_EIGHT: bigint = BigInt(8);
  // Start on a MAX_BITS boundary
  const start: bigint = (seq / BIG_MAX_BITS) * BIG_MAX_BITS;
  // beg should be in the range [0..MAX_BITS]
  const beg: number = safeBigIntToNumber(seq - start);
  // s should be in the range [0..15]
  const s: number = Math.floor(beg / BITS_PER_KEY);
  const b: number = Math.floor((beg - s * BITS_PER_KEY) / 8);

  const key = Buffer.from(bigIntToBytes(s, 1)).toString("base64");
  appState.forEach((kv) => {
    if (kv["key"] === key) {
      const v = Buffer.from(kv["value"]["bytes"], "base64");
      const bt = 1 << safeBigIntToNumber(seq % BIG_EIGHT);
      retval = (v[b] & bt) != 0;
      return;
    }
  });
  return retval;
}

/**
 * <p>Returns true if this transfer was completed on Algorand</p>
 * @param client AlgodV2 client
 * @param appId Most likely the Token bridge ID
 * @param signedVAA VAA to check
 * @param wallet The account paying the bill for this (it isn't free)
 * @returns true if VAA has been redeemed, false otherwise
 */
export async function getIsTransferCompletedAlgorand(
  client: Algodv2,
  appId: bigint,
  signedVAA: Uint8Array
): Promise<boolean> {
  const parsedVAA = _parseVAAAlgorand(signedVAA);
  const seq: bigint = parsedVAA.sequence;
  const chainRaw: string = parsedVAA.chainRaw; // this needs to be a hex string
  const em: string = parsedVAA.emitter; // this needs to be a hex string
  const { doesExist, lsa } = await calcLogicSigAccount(
    client,
    appId,
    seq / BigInt(MAX_BITS),
    chainRaw + em
  );
  if (!doesExist) {
    return false;
  }
  const seqAddr = lsa.address();
  const retVal: boolean = await checkBitsSet(client, appId, seqAddr, seq);
  return retVal;
}
