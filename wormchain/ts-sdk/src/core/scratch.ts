var fetch = require("node-fetch");
//@ts-ignore
globalThis.fetch = fetch;

import { bech32 } from "bech32";
import {
  coins,
  DirectSecp256k1HdWallet,
  EncodeObject,
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
} from "../modules/wormhole_foundation.wormchain.wormhole/rest";
import {
  txClient,
  queryClient,
} from "../modules/wormhole_foundation.wormchain.wormhole";
import { keccak256 } from "ethers/lib/utils";
import { MsgRegisterAccountAsGuardian } from "../modules/wormhole_foundation.wormchain.wormhole/types/wormhole/tx";
import { GuardianKey } from "../modules/wormhole_foundation.wormchain.wormhole/types/wormhole/guardian_key";
let elliptic = require("elliptic"); //No TS defs?

//https://tutorials.cosmos.network/academy/4-my-own-chain/cosmjs.html
const ADDRESS_PREFIX = "wormhole";
const OPERATOR_PREFIX = "wormholevaloper";
export const TENDERMINT_URL = "http://localhost:26658";
export const WORM_DENOM = "uworm";
export const LCD_URL = "http://localhost:1318";

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
    amount: coins(0, WORM_DENOM),
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

  const client = await txClient(offline, { addr: TENDERMINT_URL });
  const msg = client.msgExecuteGovernanceVAA({
    vaa: new Uint8Array(),
    signer: await getAddress(wallet),
  }); //TODO convert type

  const signingClient = await SigningStargateClient.connectWithSigner(
    TENDERMINT_URL,
    wallet
    //{ gasPrice: { amount: Decimal.fromUserInput("0.0", 0), denom: "uworm" } }
  );

  //TODO investigate signing with the stargate client, as the module txClients can't do 100% of the operations
  const output = signingClient.signAndBroadcast(
    await getAddress(wallet),
    [msg],
    getZeroFee(),
    "executing governance VAA"
  );

  //TODO the EncodingObjects from the txClient seem to be incompatible with the
  //stargate client
  // In order for all the encoding objects to be interoperable, we will have to either coerce the txClient msgs into the format of stargate,
  // or we could just just txClients for everything. I am currently leaning towards the latter, as we can generate txClients for everything out of the cosmos-sdk,
  // and we will likely need to generate txClients for our forked version of the cosmos SDK anyway.

  //const output = await client.signAndBroadcast([msg]);

  return output;
}

export async function getGuardianSets() {
  const client = await queryClient({ addr: LCD_URL });
  const response = client.queryGuardianSetAll();

  return await unpackHttpReponse(response);
}

export async function getConsensusGuardianSet() {
  const client = await queryClient({ addr: LCD_URL });
  const response = client.queryConsensusGuardianSetIndex();

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

export async function registerGuardianValidator(
  wallet: DirectSecp256k1HdWallet,
  guardianPubkeyBase64: string,
  guardianPrivkeyHex: string,
  valAddress: string
) {
  const ec = new elliptic.ec("secp256k1");
  const key = ec.keyFromPrivate(guardianPrivkeyHex);

  const binaryData = fromValAddress(valAddress);
  const bytes = binaryData.bytes;

  const hash = keccak256(bytes);

  const signature = key.sign(hash, { canonical: true });

  const args: MsgRegisterAccountAsGuardian = {
    signer: await getAddress(wallet),
    // guardianPubkey: GuardianKey.fromJSON(guardianPubkeyBase64), //TODO fix this type, it's bad
    signature: signature,
  };

  const offline: OfflineSigner = wallet;
  const client = await txClient(offline, { addr: TENDERMINT_URL });
  const msg = client.msgRegisterAccountAsGuardian(args);

  const output = await client.signAndBroadcast([msg]);

  return output;
}

export function fromAccAddress(address: string): BinaryAddress {
  return { bytes: Buffer.from(bech32.fromWords(bech32.decode(address).words)) };
}

export function fromValAddress(valAddress: string): BinaryAddress {
  return {
    bytes: Buffer.from(bech32.fromWords(bech32.decode(valAddress).words)),
  };
}

export function fromBase64(address: string): BinaryAddress {
  return { bytes: Buffer.from(address, "base64") };
}

export function toAccAddress(address: BinaryAddress): string {
  return bech32.encode(ADDRESS_PREFIX, bech32.toWords(address.bytes));
}

export function toValAddress(address: BinaryAddress): string {
  return bech32.encode(OPERATOR_PREFIX, bech32.toWords(address.bytes));
}

export function toBase64(address: BinaryAddress): string {
  return Buffer.from(address.bytes).toString("base64");
}

type BinaryAddress = {
  bytes: Uint8Array;
};
