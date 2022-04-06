import {
  ADDRESS_PREFIX,
  HOLE_DENOM,
  NODE_URL,
  OPERATOR_PREFIX,
  TENDERMINT_URL,
} from "../consts";
import axios from "axios";
import { DeclarationName } from "typescript";
import {
  Coin,
  coins,
  DirectSecp256k1HdWallet,
  EncodeObject,
} from "@cosmjs/proto-signing";
import {
  SigningStargateClient,
  StargateClient,
  isDeliverTxSuccess,
  StdFee,
  QueryClient,
  TxExtension,
  GovExtension,
  IbcExtension,
  AuthExtension,
  BankExtension,
  MintExtension,
  StakingExtension,
  setupTxExtension,
  setupGovExtension,
  setupIbcExtension,
  setupAuthExtension,
  setupBankExtension,
  setupMintExtension,
  setupStakingExtension,
} from "@cosmjs/stargate";
import { Decimal } from "@cosmjs/math";
import { Tendermint34Client } from "@cosmjs/tendermint-rpc";

//https://www.npmjs.com/package/@cosmjs/stargate
//https://gist.github.com/webmaster128/8444d42a7eceeda2544c8a59fbd7e1d9

//TODO: make a custom gov, staking, etc extension for items which were hard forked in the cosmos SDK
//TODO: make an extension for items in the wormhole module

//One of these is inside the stargate client, but is protected for whatever reason. Not sure how else the functions get exposed.
export async function getQueryClient() {
  const tmClient = await Tendermint34Client.connect(TENDERMINT_URL);
  const client = QueryClient.withExtensions(
    tmClient,
    setupTxExtension,
    setupGovExtension,
    setupIbcExtension,
    setupAuthExtension,
    setupBankExtension,
    setupMintExtension,
    setupStakingExtension
  );

  return client;
}

export async function getStargateClient() {
  const client: StargateClient = await StargateClient.connect(TENDERMINT_URL);
  return client;
}

export async function getWallet(
  mnemonic: string
): Promise<DirectSecp256k1HdWallet> {
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
    prefix: ADDRESS_PREFIX,
  });
  return wallet;
}

export function getZeroFee(): StdFee {
  return {
    amount: coins(0, HOLE_DENOM),
    gas: "180000", // 180k",
  };
}

export async function getAddress(
  wallet: DirectSecp256k1HdWallet
): Promise<string> {
  //There are actually up to 5 accounts in a cosmos wallet. I believe this returns the first wallet.
  const [{ address }] = await wallet.getAccounts();

  return address;
}

export async function getOperatorAddress(mnemonic: string): Promise<string> {
  return await getAddress(
    await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: OPERATOR_PREFIX,
    })
  );
}

// export async function faucet(denom: string, amount: string, address: string) {
//   await axios.post(FAUCET_URL, {
//     address: address,
//     coins: [amount + denom],
//   });
//   return;
// }

export async function signSendAndConfirm(
  wallet: DirectSecp256k1HdWallet,
  msgs: EncodeObject[],
  memo?: string
) {
  const address = await getAddress(wallet);
  const client = await SigningStargateClient.connectWithSigner(
    TENDERMINT_URL,
    wallet
    //{ gasPrice: { amount: Decimal.fromUserInput("0.0", 0), denom: "uhole" } }
  );

  //TODO figure out fees
  const fee = getZeroFee();

  const result = await client.signAndBroadcast(address, msgs, fee, memo);

  return result;
}
export async function sendTokens(
  wallet: DirectSecp256k1HdWallet,
  denom: string,
  amount: string,
  recipient: string,
  fee?: number | StdFee | undefined,
  memo?: string
) {
  const signer = await getAddress(wallet);
  const client = await SigningStargateClient.connectWithSigner(
    TENDERMINT_URL,
    wallet
  );

  const coin: Coin = {
    denom,
    amount,
  };
  const result = await client.sendTokens(
    signer,
    recipient,
    [coin],
    getZeroFee(),
    memo
  );
  return result;
}

export async function getBalance(denom: string, address: string) {
  const client = await getQueryClient();
  const coin = await client.bank.balance(address, denom);

  return coin.amount;
}
