import { coins } from "@cosmjs/proto-signing";
import { OfflineSigner } from "@cosmjs/proto-signing";
import { StdFee } from "@cosmjs/stargate";
import {
  getAddress,
  getWallet,
  getWormchainSigningClient,
  getWormholeQueryClient,
} from "wormhole-chain-sdk";
import {
  HOLE_DENOM,
  NODE_URL,
  TENDERMINT_URL,
  TEST_WALLET_ADDRESS_1,
  TEST_WALLET_MNEMONIC_1,
} from "../consts";

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
    amount: coins(0, HOLE_DENOM),
    gas: "180000", // 180k",
  };
}
