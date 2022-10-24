import { ChainGrpcWasmApi } from "@injectivelabs/sdk-ts";
import { Connection, PublicKey } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { Algodv2, bigIntToBytes } from "algosdk";
import { AptosClient } from "aptos";
import axios from "axios";
import { ethers } from "ethers";
import { fromUint8Array } from "js-base64";
import { redeemOnTerra } from ".";
import { ensureHexPrefix, TERRA_REDEEMED_CHECK_WALLET_ADDRESS } from "..";
import {
  BITS_PER_KEY,
  calcLogicSigAccount,
  MAX_BITS,
  _parseVAAAlgorand,
} from "../algorand";
import { callFunctionNear } from "../utils/near";
import { getSignedVAAHash } from "../bridge";
import { Bridge__factory } from "../ethers-contracts";
import { importCoreWasm } from "../solana/wasm";
import { safeBigIntToNumber } from "../utils/bigint";
import { Provider } from "near-api-js/lib/providers";
import { LCDClient as XplaLCDClient } from "@xpla/xpla.js";
import { State } from "../aptos/types";

export async function getIsTransferCompletedEth(
  tokenBridgeAddress: string,
  provider: ethers.Signer | ethers.providers.Provider,
  signedVAA: Uint8Array,
): Promise<boolean> {
  const tokenBridge = Bridge__factory.connect(tokenBridgeAddress, provider);
  const signedVAAHash = await getSignedVAAHash(signedVAA);
  return await tokenBridge.isTransferCompleted(signedVAAHash);
}

// Note: this function is the legacy implementation for terra classic.  New
// cosmwasm sdk functions should instead be based on
// `getIsTransferCompletedTerra2`.
export async function getIsTransferCompletedTerra(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  client: LCDClient,
  gasPriceUrl: string,
): Promise<boolean> {
  const msg = await redeemOnTerra(
    tokenBridgeAddress,
    TERRA_REDEEMED_CHECK_WALLET_ADDRESS,
    signedVAA,
  );
  // TODO: remove gasPriceUrl and just use the client's gas prices
  const gasPrices = await axios.get(gasPriceUrl).then((result) => result.data);
  const account = await client.auth.accountInfo(TERRA_REDEEMED_CHECK_WALLET_ADDRESS);
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
      },
    );
  } catch (e: any) {
    // redeemed if the VAA was already executed
    if (e.response.data.message.includes("VaaAlreadyExecuted")) {
      return true;
    } else {
      throw e;
    }
  }
  return false;
}

/**
 * This function is used to check if a VAA has been redeemed on terra2 by
 * querying the token bridge contract.
 * @param tokenBridgeAddress The token bridge address (bech32)
 * @param signedVAA The signed VAA byte array
 * @param client The LCD client. Only used for querying, not transactions will
 * be signed
 */
export async function getIsTransferCompletedTerra2(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  client: LCDClient
): Promise<boolean> {
  const result: { is_redeemed: boolean } = await client.wasm.contractQuery(
    tokenBridgeAddress,
    {
      is_vaa_redeemed: {
        vaa: fromUint8Array(signedVAA),
      },
    }
  );
  return result.is_redeemed;
}

/**
 * Return if the VAA has been redeemed or not
 * @param tokenBridgeAddress The Injective token bridge contract address
 * @param signedVAA The signed VAA byte array
 * @param client Holds the wallet and signing information
 * @returns true if the VAA has been redeemed.
 */
export async function getIsTransferCompletedInjective(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  client: ChainGrpcWasmApi
): Promise<boolean> {
  const queryResult = await client.fetchSmartContractState(
    tokenBridgeAddress,
    Buffer.from(
      JSON.stringify({
        is_vaa_redeemed: {
          vaa: fromUint8Array(signedVAA),
        },
      })
    ).toString("base64")
  );
  if (typeof queryResult.data === "string") {
    const result = JSON.parse(
      Buffer.from(queryResult.data, "base64").toString("utf-8")
    );
    return result.is_redeemed;
  }
  return false;
}

export async function getIsTransferCompletedXpla(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  client: XplaLCDClient
): Promise<boolean> {
  const result: { is_redeemed: boolean } = await client.wasm.contractQuery(tokenBridgeAddress, {
    is_vaa_redeemed: {
      vaa: fromUint8Array(signedVAA),
    },
  });
  return result.is_redeemed;
}

export async function getIsTransferCompletedSolana(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  connection: Connection,
): Promise<boolean> {
  const { claim_address } = await importCoreWasm();
  const claimAddress = await claim_address(tokenBridgeAddress, signedVAA);
  const claimInfo = await connection.getAccountInfo(new PublicKey(claimAddress), "confirmed");
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
  seq: bigint,
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
 * @returns true if VAA has been redeemed, false otherwise
 */
export async function getIsTransferCompletedAlgorand(
  client: Algodv2,
  appId: bigint,
  signedVAA: Uint8Array,
): Promise<boolean> {
  const parsedVAA = _parseVAAAlgorand(signedVAA);
  const seq: bigint = parsedVAA.sequence;
  const chainRaw: string = parsedVAA.chainRaw; // this needs to be a hex string
  const em: string = parsedVAA.emitter; // this needs to be a hex string
  const { doesExist, lsa } = await calcLogicSigAccount(
    client,
    appId,
    seq / BigInt(MAX_BITS),
    chainRaw + em,
  );
  if (!doesExist) {
    return false;
  }
  const seqAddr = lsa.address();
  const retVal: boolean = await checkBitsSet(client, appId, seqAddr, seq);
  return retVal;
}

export async function getIsTransferCompletedNear(
  provider: Provider,
  tokenBridge: string,
  signedVAA: Uint8Array,
): Promise<boolean> {
  const vaa = Buffer.from(signedVAA).toString("hex");
  return (
    await callFunctionNear(provider, tokenBridge, "is_transfer_completed", {
      vaa,
    })
  )[1];
}

export async function getIsTransferCompletedAptos(
  client: AptosClient,
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
): Promise<boolean> {
  // get handle
  tokenBridgeAddress = ensureHexPrefix(tokenBridgeAddress);
  const state = (
    await client.getAccountResource(tokenBridgeAddress, `${tokenBridgeAddress}::state::State`)
  ).data as State;
  const handle = state.consumed_vaas.elems.handle;

  // check if vaa hash is in consumed_vaas
  const signedVAAHash = await getSignedVAAHash(signedVAA);
  try {
    // when accessing Set<T>, key is type T and value is 0
    await client.getTableItem(handle, {
      key_type: "vector<u8>",
      value_type: "u8",
      key: signedVAAHash,
    });
    return true;
  } catch {
    return false;
  }
}
