import {
  coins,
  DirectSecp256k1HdWallet,
  OfflineDirectSigner,
  OfflineSigner,
} from "@cosmjs/proto-signing";
import {
  SigningStargateClient,
  StargateClient,
  StdFee,
} from "@cosmjs/stargate";
import { ChainRegistration } from "./modules/certusone.wormholechain.tokenbridge/types/tokenbridge/chain_registration";
import { txClient } from "./modules/certusone.wormholechain.wormhole";

//https://tutorials.cosmos.network/academy/4-my-own-chain/cosmjs.html
const ADDRESS_PREFIX = "wormhole";
export const TENDERMINT_URL = "http://localhost:26657";
export const HOLE_DENOM = "uhole";

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
