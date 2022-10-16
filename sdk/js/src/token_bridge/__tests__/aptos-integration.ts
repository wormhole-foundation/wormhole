import { describe, expect, jest, test } from "@jest/globals";
import { AptosAccount, AptosClient, FaucetClient, HexString } from "aptos";
import {
  attestFromAptos,
  attestFromEth,
  CHAIN_ID_APTOS,
  CONTRACTS,
  createWrappedOnEth,
  getForeignAssetEth,
  getSignedVAAWithRetry,
  hexToUint8Array,
  signAndSubmitTransaction,
  waitForSignAndSubmitTransaction,
} from "../..";
import { setDefaultWasm } from "../../solana/wasm";
import {
  APTOS_FAUCET_URL,
  APTOS_NODE_URL,
  APTOS_PRIVATE_KEY,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY3,
  TEST_ERC20,
  WORMHOLE_RPC_HOSTS,
} from "./consts";
import { parseSequenceFromLogAptos } from "../../bridge/parseSequenceFromLog";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { ethers } from "ethers";
import {
  createWrappedCoin,
  createWrappedCoinType,
  registerChain,
} from "../../aptos";

setDefaultWasm("node");

jest.setTimeout(60000);

describe("Aptos SDK tests", () => {
  test("Transfer native token from Aptos to Ethereum", async () => {
    // setup aptos
    const client = new AptosClient(APTOS_NODE_URL);
    const faucet = new FaucetClient(APTOS_NODE_URL, APTOS_FAUCET_URL);
    const account = new AptosAccount(
      new Uint8Array(Buffer.from(APTOS_PRIVATE_KEY, "hex"))
    );
    const before = await getFunds(client, account.address());
    await faucet.fundAccount(account.address(), 100_000_000);

    // sanity check funds in the account
    const after = await getFunds(client, account.address());
    expect(after - before).toEqual(100_000_000);

    // attest native aptos token
    const tokenBridgeAddress = CONTRACTS.DEVNET.aptos.token_bridge;
    // console.log(JSON.stringify(await client.getAccountResources(account.address()), null, 2))
    // console.log(JSON.stringify(await client.getAccountResources(tokenBridgeAddress), null, 2))

    const attestPayload = attestFromAptos(
      tokenBridgeAddress,
      CHAIN_ID_APTOS,
      "0x1::aptos_coin::AptosCoin"
    );
    const tx = await waitForSignAndSubmitTransaction(
      client,
      account,
      attestPayload
    );

    // get signed vaa
    // const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
    //   WORMHOLE_RPC_HOSTS,
    //   CHAIN_ID_APTOS,
    //   tokenBridgeAddress, // already 32 bytes, so we don't need to normalize emitter address
    //   parseSequenceFromLogAptos(tx),
    //   {
    //     transport: NodeHttpTransport(),
    //   },
    // );
    // console.log({signedVAA});

    // // setup ethereum
    // const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
    // const signer = new ethers.Wallet(ETH_PRIVATE_KEY3, provider);
    // try {
    //   await createWrappedOnEth(CONTRACTS.DEVNET.ethereum.token_bridge, signer, signedVAA);
    // } catch (e) {
    //   // this could fail because the token is already attested (in an unclean env)
    // }

    // // check attestation on ethereum
    // const normalizedType = normalizeNativeAssetType("0x1::aptos_coin::AptosCoin");
    // if (!normalizedType) throw "";
    // const externalAddress = Uint8Array.from(Buffer.from(getExternalAddress(normalizedType)));
    // const address = getForeignAssetEth(
    //   CONTRACTS.DEVNET.ethereum.token_bridge,
    //   provider,
    //   CHAIN_ID_APTOS,
    //   externalAddress,
    // );
    // expect(address).toBeTruthy();
    // expect(address).not.toBe(ethers.constants.AddressZero);
    // provider.destroy();

    // transfer to ethereum
  });
  test.only("Transfer from Ethereum to Aptos", async () => {
    // setup aptos
    const client = new AptosClient(APTOS_NODE_URL);
    const account = new AptosAccount(
      new Uint8Array(Buffer.from(APTOS_PRIVATE_KEY, "hex"))
    );

    // // setup ethereum
    // const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
    // const signer = new ethers.Wallet(ETH_PRIVATE_KEY3, provider);
    // try {
    //   // await attestFromEth(CONTRACTS.DEVNET.ethereum.token_bridge, signer, TEST_ERC20);
    // } catch (e) {
    //   // this could fail because the token is already attested (in an unclean env)
    // }

    // register eth
    const tokenBridgeAddress = CONTRACTS.DEVNET.aptos.token_bridge;
    const registrationVAA = hexToUint8Array(
      "010000000001002cef01cdddcbf42adeb88eb10ddb6bfb8dd7beb97bd61548e34b299dc47c80fd6b5a6679c345aef1b868c913d46d8d33a1849bb7c11bbcb791a2cee9caac8aa4000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000001c53fe200000000000000000000000000000000000000000000546f6b656e42726964676501000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
    );
    const registrationPayload = registerChain(
      tokenBridgeAddress,
      registrationVAA
    );
    try {
      await waitForSignAndSubmitTransaction(
        client,
        account,
        registrationPayload
      );
    } catch {} // hack to run test multiple times

    // get attest vaa
    const attestVAA = hexToUint8Array(
      "010000000001008b63d1b99fe9052678bc8be230c2a0b6d911368a9c5e4c5e1f3251de91342c3d04a18236bf7f53383893a98a2b74e6c8c7afd63e68b117766e14a25ac26e0be301000000010000000100020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c1600000000028b11e300020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002085445535400000000000000000000000000000000000000000000000000000000544553545f455243323000000000000000000000000000000000000000000000"
    );

    // create cointype
    const createWrappedCoinTypePayload = createWrappedCoinType(
      tokenBridgeAddress,
      attestVAA
    );
    try {
      await waitForSignAndSubmitTransaction(
        client,
        account,
        createWrappedCoinTypePayload
      );
    } catch {} // hack to run test multiple times

    // create coin
    const createWrappedCoinPayload = createWrappedCoin(
      tokenBridgeAddress,
      attestVAA
    );
    await waitForSignAndSubmitTransaction(
      client,
      account,
      createWrappedCoinPayload
    );
  });
});

const getFunds = async (
  client: AptosClient,
  address: HexString
): Promise<number> => {
  const type = "0x1::coin::CoinStore<0x1::aptos_coin::AptosCoin>";
  const res = await client.getAccountResource(address, type);
  return (res.data as any).coin.value;
};
