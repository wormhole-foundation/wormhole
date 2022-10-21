import { coins } from "@cosmjs/proto-signing";
import { OfflineSigner } from "@cosmjs/proto-signing";
import { StdFee } from "@cosmjs/stargate";
import {
  fromValAddress,
  getAddress,
  getWallet,
  getWormchainSigningClient,
  getWormholeQueryClient,
} from "@wormhole-foundation/wormchain-sdk";
import {
  WORM_DENOM,
  NODE_URL,
  TENDERMINT_URL,
  TEST_WALLET_ADDRESS_1,
  TEST_WALLET_MNEMONIC_1,
} from "../consts.js";
//@ts-ignore
import * as elliptic from "elliptic";
import keccak256 from "keccak256";

export async function getQueryClient() {
  return getWormholeQueryClient(NODE_URL, true);
}

export async function getSigningClient() {
  const testWallet1 = await getWallet(TEST_WALLET_MNEMONIC_1);
  return await getWormchainSigningClient(TENDERMINT_URL, testWallet1);
}

export async function getBalance(address: string, denom: string) {
  const client = await getQueryClient();
  const response = await client.bank.queryBalance(address, denom);
  return response.data.balance?.amount || "0";
}

export async function sendTokens(
  wallet: OfflineSigner,
  recipient: string,
  amount: string,
  denom: string
) {
  const client = await getWormchainSigningClient(TENDERMINT_URL, wallet);
  const fromAddress = await getAddress(wallet);
  //TODO the autogenned protobuf code doesn't appear to be correct. This not working is a serious bug
  //   const msg = client.bank.msgSend({
  //     //@ts-ignore
  //     amount: [{ amount, denom }],
  //     //@ts-ignore
  //     from_address: fromAddress,
  //     to_address: recipient,
  //   });

  return client.sendTokens(
    fromAddress,
    recipient,
    [{ amount, denom }],
    getZeroFee(),
    "basicTransfer test"
  );
}

export function getZeroFee(): StdFee {
  return {
    amount: coins(0, WORM_DENOM),
    gas: "180000", // 180k",
  };
}

export function signValidatorAddress(valAddr: string, privKey: string) {
  const EC = elliptic.default.ec;
  const ec = new EC("secp256k1");
  const key = ec.keyFromPrivate(privKey);
  const valAddrHash = keccak256(
    Buffer.from(fromValAddress(valAddr).bytes)
  ).toString("hex");
  const signature = key.sign(valAddrHash, { canonical: true });
  const hexString =
    signature.r.toString("hex").padStart(64, "0") +
    signature.s.toString("hex").padStart(64, "0") +
    signature.recoveryParam.toString(16).padStart(2, "0");
  return Buffer.from(hexString, "hex");
}
