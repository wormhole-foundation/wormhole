var fetch = require("node-fetch");
//@ts-ignore
globalThis.fetch = fetch;

import { bech32 } from "bech32";
import {
  coins,
  DirectSecp256k1HdWallet,
  OfflineDirectSigner,
  OfflineSigner,
} from "@cosmjs/proto-signing";
import {
  QueryClient,
  setupAuthExtension,
  setupBankExtension,
  setupGovExtension,
  setupIbcExtension,
  setupMintExtension,
  setupStakingExtension,
  setupTxExtension,
  SigningStargateClient,
  StargateClient,
  StdFee,
} from "@cosmjs/stargate";
import { Tendermint34Client } from "@cosmjs/tendermint-rpc";
import {
  RpcStatus,
  HttpResponse,
} from "./modules/certusone.wormholechain.tokenbridge/rest";
import { ChainRegistration } from "./modules/certusone.wormholechain.tokenbridge/types/tokenbridge/chain_registration";
import {
  txClient,
  queryClient,
} from "./modules/certusone.wormholechain.wormhole";

//https://tutorials.cosmos.network/academy/4-my-own-chain/cosmjs.html
const ADDRESS_PREFIX = "wormhole";
const OPERATOR_PREFIX = "wormholevaloper";
export const TENDERMINT_URL = "http://localhost:26657";
export const HOLE_DENOM = "uhole";
export const LCD_URL = "http://localhost:1317";

export async function getStargateQueryClient() {
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

export function getZeroFee(): StdFee {
  return {
    amount: coins(0, HOLE_DENOM),
    gas: "180000", // 180k",
  };
}

export async function getWallet(
  mnemonic: string
): Promise<DirectSecp256k1HdWallet> {
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
    prefix: ADDRESS_PREFIX,
  });
  return wallet;
}

export async function getAddress(
  wallet: DirectSecp256k1HdWallet
): Promise<string> {
  //There are actually up to 5 accounts in a cosmos wallet. I believe this returns the first wallet.
  const [{ address }] = await wallet.getAccounts();

  return address;
}

export async function executeGovernanceVAA(
  wallet: DirectSecp256k1HdWallet,
  hexVaa: string
) {
  const offline: OfflineSigner = wallet;

  const client = await txClient(offline);
  const msg = client.msgExecuteGovernanceVAA({
    vaa: new Uint8Array(),
    signer: await getAddress(wallet),
  }); //TODO convert type

  const signingClient = await SigningStargateClient.connectWithSigner(
    TENDERMINT_URL,
    wallet
    //{ gasPrice: { amount: Decimal.fromUserInput("0.0", 0), denom: "uhole" } }
  );

  //TODO investigate signing with the stargate client, as the module txClients can't do 100% of the operations
  //   const output = signingClient.signAndBroadcast(
  //     await getAddress(wallet),
  //     [msg],
  //     getZeroFee(),
  //     "executing governance VAA"
  //   );

  //TODO the EncodingObjects from the txClient seem to be incompatible with the
  //stargate client
  // In order for all the encoding objects to be interoperable, we will have to either coerce the txClient msgs into the format of stargate,
  // or we could just just txClients for everything. I am currently leaning towards the latter, as we can generate txClients for everything out of the cosmos-sdk,
  // and we will likely need to generate txClients for our forked version of the cosmos SDK anyway.

  const output = await client.signAndBroadcast([msg]);

  return output;
}

export async function getGuardianSets() {
  const client = await queryClient({ addr: LCD_URL });
  const response = client.queryGuardianSetAll();

  return await unpackHttpReponse(response);
}

export async function getActiveGuardianSet() {
  const client = await queryClient({ addr: LCD_URL });
  const response = client.queryActiveGuardianSetIndex();

  return await unpackHttpReponse(response);
}

export async function getValidators() {
  const client = await getStargateQueryClient();
  //TODO handle pagination here
  const validators = await client.staking.validators("BOND_STATUS_BONDED");

  return validators;
}

export async function getGuardianValidatorRegistrations() {
  const client = await queryClient({ addr: LCD_URL });
  const response = client.queryGuardianValidatorAll();

  return await unpackHttpReponse(response);
}

export async function unpackHttpReponse<T>(
  response: Promise<HttpResponse<T, RpcStatus>>
) {
  const http = await response;
  //TODO check rpc status
  const content = http.data;

  return content;
}

export function fromAccAddress(address: string): BinaryAddress {
  return { words: Buffer.from(bech32.decode(address).words) };
}

export function fromValAddress(valAddress: string): BinaryAddress {
  return { words: Buffer.from(bech32.decode(valAddress).words) };
}

export function fromBase64(address: string): BinaryAddress {
  return { words: Buffer.from(bech32.toWords(Buffer.from(address, "base64"))) };
}

export function toAccAddress(address: BinaryAddress): string {
  return bech32.encode(ADDRESS_PREFIX, address.words);
}

export function toValAddress(address: BinaryAddress): string {
  return bech32.encode(OPERATOR_PREFIX, address.words);
}

export function toBase64(address: BinaryAddress): string {
  return Buffer.from(bech32.fromWords(address.words)).toString("base64");
}

type BinaryAddress = {
  words: Uint8Array;
};
