// npx pretty-quick

import Web3 from "web3";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";

import { TestLib } from "./testlib";

import {
  NFTImplementation,
  NFTImplementation__factory,
  nft_bridge,
  getEmitterAddressEth,
  parseSequenceFromLogEth,
  getSignedVAAWithRetry,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_ETH,
  CHAIN_ID_NEAR,
  ChainId,
  ChainName,
  textToUint8Array,
  tryNativeToUint8Array,
  CONTRACTS,
  parseNFTPayload,
} from "@certusone/wormhole-sdk";

import {
  ParsedVAA,
  _parseVAAAlgorand,
  _parseNFTAlgorand,
} from "@certusone/wormhole-sdk/lib/cjs/algorand";

import {
  BigNumberish,
  ethers,
} from "@certusone/wormhole-sdk/node_modules/ethers";

import { Account as nearAccount } from "@certusone/wormhole-sdk/node_modules/near-api-js";
const BN = require("bn.js");
const fetch = require("node-fetch");

const ERC721 = require("@openzeppelin/contracts/build/contracts/ERC721PresetMinterPauserAutoId.json");
const nearAPI = require("near-api-js");

export const ETH_NODE_URL = "ws://localhost:8545";
export const ETH_PRIVATE_KEY =
  "0x6cbed15c793ce57650b9877cf6fa156fbef513c4e6134f022a85b1ffdd59b2a1"; // account 1
export const ETH_CORE_BRIDGE_ADDRESS =
  "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550";
export const ETH_NFT_BRIDGE_ADDRESS =
  "0x26b4afb60d6c903165150c6f0aa14f8016be4aec";

const web3 = new Web3(ETH_NODE_URL);
let provider: ethers.providers.WebSocketProvider =
  new ethers.providers.WebSocketProvider(ETH_NODE_URL);
let signer: ethers.Wallet = new ethers.Wallet(ETH_PRIVATE_KEY, provider);

export function textToHexString(name: string): string {
  return Buffer.from(name, "binary").toString("hex");
}

export const hexToUint8Array = (h: string): Uint8Array =>
  new Uint8Array(Buffer.from(h, "hex"));

export const uint8ArrayToHex = (a: Uint8Array): string =>
  Buffer.from(a).toString("hex");

async function _transferFromEth(
  erc721: string,
  token_id: BigNumberish,
  address: string,
  chain: ChainId
): Promise<ethers.ContractReceipt> {
  return nft_bridge.transferFromEth(
    ETH_NFT_BRIDGE_ADDRESS,
    signer,
    erc721,
    token_id,
    chain,
    hexToUint8Array(address)
  );
}

async function deployNFTOnEth(
  name: string,
  symbol: string,
  uri: string,
  how_many: number
): Promise<NFTImplementation> {
  const accounts = await web3.eth.getAccounts();
  const nftContract = new web3.eth.Contract(ERC721.abi);
  let nft = await nftContract
    .deploy({
      data: ERC721.bytecode,
      arguments: [name, symbol, uri],
    })
    .send({
      from: accounts[1],
      gas: 5000000,
    });

  // The eth contracts mints tokens with sequential ids, so in order to get to a
  // specific id, we need to mint multiple nfts. We need this to test that
  // foreign ids on terra get converted to the decimal stringified form of the
  // original id.
  for (var i = 0; i < how_many; i++) {
    await nft.methods.mint(accounts[1]).send({
      from: accounts[1],
      gas: 1000000,
    });
  }

  return NFTImplementation__factory.connect(nft.options.address, signer);
}

async function waitUntilEthTxObserved(
  receipt: ethers.ContractReceipt
): Promise<Uint8Array> {
  // get the sequence from the logs (needed to fetch the vaa)
  let sequence = parseSequenceFromLogEth(receipt, ETH_CORE_BRIDGE_ADDRESS);
  let emitterAddress = getEmitterAddressEth(ETH_NFT_BRIDGE_ADDRESS);
  // poll until the guardian(s) witness and sign the vaa
  const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
    ["http://localhost:7071"],
    CHAIN_ID_ETH,
    emitterAddress,
    sequence,
    {
      transport: NodeHttpTransport(),
    }
  );
  return signedVAA;
}

function getConfig(env: any) {
  switch (env) {
    case "sandbox":
    case "local":
      return {
        networkId: "sandbox",
        nodeUrl: "http://localhost:3030",
        masterAccount: "test.near",
        wormholeAccount: "wormhole.test.near",
        tokenAccount: "token.test.near",
        nftAccount: "nft.test.near",
        userAccount:
          Math.floor(Math.random() * 10000).toString() + "user.test.near",
        user2Account:
          Math.floor(Math.random() * 10000).toString() + "user.test.near",
      };
  }
  return {};
}

export const METADATA_REPLACE = new RegExp("\u0000", "g");

async function testNearSDK() {
  let config = getConfig(process.env.NEAR_ENV || "sandbox");

  // Retrieve the validator key directly in the Tilt environment
  const response = await fetch("http://localhost:3031/validator_key.json");

  const keyFile = await response.json();

  let masterKey = nearAPI.utils.KeyPair.fromString(
    keyFile.secret_key || keyFile.private_key
  );
  let masterPubKey = masterKey.getPublicKey();

  let keyStore = new nearAPI.keyStores.InMemoryKeyStore();
  keyStore.setKey(config.networkId, config.masterAccount, masterKey);

  let near = await nearAPI.connect({
    deps: {
      keyStore,
    },
    networkId: config.networkId,
    nodeUrl: config.nodeUrl,
  });
  let masterAccount = new nearAPI.Account(
    near.connection,
    config.masterAccount
  );

  console.log(
    "Finish init NEAR masterAccount: " +
      JSON.stringify(await masterAccount.getAccountBalance())
  );

  let userKey = nearAPI.utils.KeyPair.fromRandom("ed25519");
  keyStore.setKey(config.networkId, config.userAccount, userKey);
  let user2Key = nearAPI.utils.KeyPair.fromRandom("ed25519");
  keyStore.setKey(config.networkId, config.user2Account, user2Key);

  console.log(
    "creating a user account: " +
      config.userAccount +
      " with key " +
      userKey.getPublicKey()
  );

  await masterAccount.createAccount(
    config.userAccount,
    userKey.getPublicKey(),
    new BN(10).pow(new BN(27))
  );
  const userAccount = new nearAPI.Account(near.connection, config.userAccount);

  console.log(
    "Creating new random non-wormhole nft and air dropping it to myself"
  );

  let randoNFT = nearAPI.providers.getTransactionLastResult(
    await userAccount.functionCall({
      contractId: "test.test.near",
      methodName: "deploy_nft",
      args: {
        account: userAccount.accountId,
      },
      attachedDeposit: new BN("10000000000000000000000000"),
      gas: 300000000000000,
    })
  );

  console.log(randoNFT);

  console.log("Registering the user");

  let myAddress = nearAPI.providers.getTransactionLastResult(
    await userAccount.functionCall({
      contractId: config.nftAccount,
      methodName: "register_account",
      args: { account: userAccount.accountId },
      gas: new BN("100000000000000"),
      attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
    })
  );
  console.log("myAddress: " + myAddress);

  let contract = textToHexString(Math.random().toString());
  let tokenid = textToHexString(Math.random().toString());

  let ts = new TestLib();
  let seq = Math.floor(new Date().getTime() / 1000);
  let v = ts.genNFTTransfer(
    ts.singleGuardianPrivKey,
    0,
    1,
    seq,

    contract,
    1, // from chain

    "George", // symbol
    "GeorgesNFT", // name
    tokenid,
    "https://cloudflare-ipfs.com/ipfs/QmeSjSinHpPnmXmspMjwiXyN6zS4E9zccariGR3jxcaWtq/10",
    myAddress,
    15
  );

  //let tvaa = ts.hexStringToUint8Array(v);
  //console.log(v);
  //console.log(_parseNFTAlgorand(tvaa));

  let res = await userAccount.viewFunction(
    config.nftAccount,
    "deposit_estimates",
    {}
  );

  console.log(res);

  let emitter = (await userAccount.viewFunction(
    config.nftAccount,
    "emitter",
    {}
  ))[1];

  console.log(emitter);

  console.log("submitting a nft");

  let ret = nearAPI.providers.getTransactionLastResult(
    await userAccount.functionCall({
      contractId: config.nftAccount,
      methodName: "submit_vaa",
      args: {
        vaa: v,
      },
      gas: 300000000000000,
      attachedDeposit: new BN(res[1]),
    })
  );

  console.log(ret);

  ret = nearAPI.providers.getTransactionLastResult(
    await userAccount.functionCall({
      contractId: config.nftAccount,
      methodName: "submit_vaa",
      args: {
        vaa: v,
      },
      gas: 300000000000000,
      attachedDeposit: new BN(res[1]),
    })
  );

  console.log(ret);

  console.log("make it go away");

  let t = nearAPI.providers.getTransactionLastResult(
    await userAccount.functionCall({
      contractId: config.nftAccount,
      methodName: "initiate_transfer",
      args: {
        asset: ret[0],
        token_id: ret[1],
        recipient_chain: 8,
        recipient: "00112233",
        nonce: 5,
      },
      gas: 300000000000000,
      attachedDeposit: 1,
    })
  );

  console.log(t);

    console.log(emitter);

    console.log("looking it up");
    const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
      ["http://localhost:7071"],
      CHAIN_ID_NEAR,
      emitter,
      t,
      {
        transport: NodeHttpTransport(),
      }
    );

    console.log(signedVAA);

  console.log("all done");
  process.exit(0);
}

testNearSDK();
