import {
  afterAll,
  beforeEach,
  describe,
  expect,
  jest,
  test,
} from "@jest/globals";
import { getAssociatedTokenAddress } from "@solana/spl-token";
import { Connection, Keypair, PublicKey } from "@solana/web3.js";
import { AptosAccount, AptosClient, FaucetClient, Types } from "aptos";
import { ethers } from "ethers";
import Web3 from "web3";
import { DepositEvent, TokenId } from "../../aptos/types";
import {
  CHAIN_ID_APTOS,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CONTRACTS,
  deriveTokenHashFromTokenId,
  generateSignAndSubmitEntryFunction,
  tryNativeToHexString,
  tryNativeToUint8Array,
} from "../../utils";
import { parseNftTransferVaa } from "../../vaa";
import { getForeignAssetAptos } from "../getForeignAsset";
import { getIsTransferCompletedAptos } from "../getIsTransferCompleted";
import { getIsWrappedAssetAptos } from "../getIsWrappedAsset";
import { getOriginalAssetAptos } from "../getOriginalAsset";
import { redeemOnAptos, redeemOnEth } from "../redeem";
import {
  transferFromAptos,
  transferFromEth,
  transferFromSolana,
} from "../transfer";
import {
  APTOS_FAUCET_URL,
  APTOS_NODE_URL,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  TEST_SOLANA_TOKEN,
} from "./consts";
import {
  deployTestNftOnAptos,
  deployTestNftOnEthereum,
} from "./utils/deployTestNft";
import {
  getSignedVaaAptos,
  getSignedVaaEthereum,
  getSignedVaaSolana,
} from "./utils/getSignedVaa";

jest.setTimeout(60000);

const APTOS_NFT_BRIDGE_ADDRESS = CONTRACTS.DEVNET.aptos.nft_bridge;
const ETH_NFT_BRIDGE_ADDRESS = CONTRACTS.DEVNET.ethereum.nft_bridge;
const SOLANA_NFT_BRIDGE_ADDRESS = CONTRACTS.DEVNET.solana.nft_bridge;
const SOLANA_CORE_BRIDGE_ADDRESS = CONTRACTS.DEVNET.solana.core;

// aptos setup
let aptosClient: AptosClient;
let aptosAccount: AptosAccount;
let faucet: FaucetClient;

// ethereum setup
const web3 = new Web3(ETH_NODE_URL);
const ethProvider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
const ethSigner = new ethers.Wallet(ETH_PRIVATE_KEY, ethProvider); // corresponds to accounts[1]

// solana setup
const solanaConnection = new Connection(SOLANA_HOST, "confirmed");
const solanaKeypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
const solanaPayerAddress = solanaKeypair.publicKey.toString();

beforeEach(async () => {
  aptosClient = new AptosClient(APTOS_NODE_URL);
  aptosAccount = new AptosAccount();
  faucet = new FaucetClient(APTOS_NODE_URL, APTOS_FAUCET_URL);
  await faucet.fundAccount(aptosAccount.address(), 100_000_000);
});

afterAll(async () => {
  (web3.currentProvider as any).disconnect();
  await ethProvider.destroy();
});

describe("Aptos NFT SDK tests", () => {
  test("Transfer ERC-721 from Ethereum to Aptos and back", async () => {
    const ETH_COLLECTION_NAME = "Not an APE 🐒";

    // create NFT on Ethereum
    const ethNft = await deployTestNftOnEthereum(
      web3,
      ethSigner,
      ETH_COLLECTION_NAME,
      "APE🐒",
      "https://cloudflare-ipfs.com/ipfs/QmeSjSinHpPnmXmspMjwiXyN6zS4E9zccariGR3jxcaWtq/",
      11
    );

    // transfer NFT from Ethereum to Aptos
    const ethTransferTx = await transferFromEth(
      ETH_NFT_BRIDGE_ADDRESS,
      ethSigner,
      ethNft.address,
      10,
      CHAIN_ID_APTOS,
      aptosAccount.address().toUint8Array()
    );

    // observe tx and get vaa
    const ethTransferVaa = await getSignedVaaEthereum(ethTransferTx);
    const ethTransferVaaParsed = parseNftTransferVaa(ethTransferVaa);
    expect(ethTransferVaaParsed.name).toBe(ETH_COLLECTION_NAME);

    // redeem NFT on Aptos
    const aptosRedeemPayload = await redeemOnAptos(
      APTOS_NFT_BRIDGE_ADDRESS,
      ethTransferVaa
    );
    const aptosRedeemTx = await generateSignAndSubmitEntryFunction(
      aptosClient,
      aptosAccount,
      aptosRedeemPayload
    );
    const aptosRedeemTxResult = (await aptosClient.waitForTransactionWithResult(
      aptosRedeemTx.hash
    )) as Types.UserTransaction;
    expect(aptosRedeemTxResult.success).toBe(true);
    expect(
      await getIsTransferCompletedAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        ethTransferVaa
      )
    ).toBe(true);

    // get token data
    const tokenData = await getForeignAssetAptos(
      aptosClient,
      APTOS_NFT_BRIDGE_ADDRESS,
      CHAIN_ID_ETH,
      tryNativeToUint8Array(ethNft.address, CHAIN_ID_ETH)
    );
    assertIsNotNull(tokenData);
    expect(
      await getIsWrappedAssetAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        tokenData.creatorAddress
      )
    ).toBe(true);
    expect(
      await getOriginalAssetAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        tokenData.creatorAddress
      )
    ).toMatchObject({
      isWrapped: true,
      chainId: CHAIN_ID_ETH,
      assetAddress: Uint8Array.from(ethTransferVaaParsed.tokenAddress),
    });

    // transfer NFT from Aptos back to Ethereum
    const aptosTransferPayload = await transferFromAptos(
      APTOS_NFT_BRIDGE_ADDRESS,
      tokenData.creatorAddress,
      tokenData.collectionName,
      tokenData.tokenName.padStart(64, "0"),
      0,
      CHAIN_ID_ETH,
      tryNativeToUint8Array(ethSigner.address, CHAIN_ID_ETH)
    );
    const aptosTransferTx = await generateSignAndSubmitEntryFunction(
      aptosClient,
      aptosAccount,
      aptosTransferPayload
    );
    const aptosTransferTxResult =
      (await aptosClient.waitForTransactionWithResult(
        aptosTransferTx.hash
      )) as Types.UserTransaction;
    expect(aptosTransferTxResult.success).toBe(true);

    // observe tx and get vaa
    let aptosTransferVaa = await getSignedVaaAptos(
      aptosClient,
      aptosTransferTxResult
    );
    const aptosTransferVaaParsed = parseNftTransferVaa(aptosTransferVaa);
    expect(aptosTransferVaaParsed.name).toBe(ETH_COLLECTION_NAME);
    expect(aptosTransferVaaParsed.tokenAddress.toString("hex")).toBe(
      ethTransferVaaParsed.tokenAddress.toString("hex")
    );

    // redeem NFT on Ethereum
    const ethRedeemTxResult = await redeemOnEth(
      ETH_NFT_BRIDGE_ADDRESS,
      ethSigner,
      aptosTransferVaa
    );
    expect(ethRedeemTxResult.status).toBe(1);
  });

  test("Transfer Aptos native token to Ethereum and back", async () => {
    const APTOS_COLLECTION_NAME = "Not an APE 🦧";

    // mint NFT on Aptos
    const aptosNftMintTxResult = await deployTestNftOnAptos(
      aptosClient,
      aptosAccount,
      APTOS_COLLECTION_NAME,
      "APE🦧"
    );

    // get token data from user wallet
    const event = (
      await aptosClient.getEventsByEventHandle(
        aptosAccount.address(),
        "0x3::token::TokenStore",
        "deposit_events",
        { limit: 1 } // most users will more than one deposit event
      )
    )[0] as DepositEvent;
    const tokenId: TokenId = {
      creatorAddress: event.data.id.token_data_id.creator,
      collectionName: event.data.id.token_data_id.collection,
      tokenName: event.data.id.token_data_id.name,
      propertyVersion: Number(event.data.id.property_version),
    };
    console.log(tokenId);

    // transfer NFT from Aptos to Solana
    const aptosTransferPayload = await transferFromAptos(
      APTOS_NFT_BRIDGE_ADDRESS,
      tokenId.creatorAddress,
      tokenId.collectionName,
      tokenId.tokenName,
      tokenId.propertyVersion,
      CHAIN_ID_ETH,
      tryNativeToUint8Array(ethSigner.address, CHAIN_ID_ETH)
    );
    const aptosTransferTx = await generateSignAndSubmitEntryFunction(
      aptosClient,
      aptosAccount,
      aptosTransferPayload
    );
    const aptosTransferTxResult =
      (await aptosClient.waitForTransactionWithResult(
        aptosTransferTx.hash
      )) as Types.UserTransaction;
    expect(aptosTransferTxResult.success).toBe(true);

    // observe tx and get vaa
    const aptosTransferVaa = await getSignedVaaAptos(
      aptosClient,
      aptosTransferTxResult
    );
    const aptosTransferVaaParsed = parseNftTransferVaa(aptosTransferVaa);
    expect(aptosTransferVaaParsed.name).toBe(APTOS_COLLECTION_NAME);

    // redeem NFT on Ethereum
    const ethRedeemTx = await redeemOnEth(
      ETH_NFT_BRIDGE_ADDRESS,
      ethSigner,
      aptosTransferVaa
    );
    console.log(JSON.stringify(ethRedeemTx, null, 2));
    expect(ethRedeemTx.status).toBe(1);

    // get token address on Ethereum
    const tokenHash = await deriveTokenHashFromTokenId(tokenId);
    expect(
      await getForeignAssetAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        CHAIN_ID_APTOS,
        tokenHash
      )
    ).toMatchObject(tokenId);
    // const tokenAddress = await getForeignAssetEth(
    //   ETH_NFT_BRIDGE_ADDRESS,
    //   ethSigner,
    //   CHAIN_ID_APTOS,
    //   tokenHash
    // );
    // console.log(tokenAddress);
    // assertIsNotNull(tokenAddress);

    // // transfer NFT from Ethereum back to Aptos
    // const ethTransferTx = await transferFromEth(
    //   ETH_NFT_BRIDGE_ADDRESS,
    //   ethSigner,
    //   tokenAddress,
    //   0,
    //   CHAIN_ID_APTOS,
    //   tryNativeToUint8Array(aptosAccount.address().toString(), CHAIN_ID_APTOS)
    // );
    // expect(ethTransferTx.status).toBe(1);

    // // observe tx and get vaa

    // // redeem NFT on Aptos

    // check NFT is the same
  });

  test("Transfer Solana SPL to Aptos", async () => {
    // transfer SPL token to Aptos
    const fromAddress = await getAssociatedTokenAddress(
      new PublicKey(TEST_SOLANA_TOKEN),
      solanaKeypair.publicKey
    );
    const solanaTransferTx = await transferFromSolana(
      solanaConnection,
      SOLANA_CORE_BRIDGE_ADDRESS,
      SOLANA_NFT_BRIDGE_ADDRESS,
      solanaPayerAddress,
      fromAddress.toString(),
      TEST_SOLANA_TOKEN,
      tryNativeToUint8Array(aptosAccount.address().toString(), CHAIN_ID_APTOS),
      CHAIN_ID_APTOS
    );
    solanaTransferTx.partialSign(solanaKeypair);
    const txid = await solanaConnection.sendRawTransaction(
      solanaTransferTx.serialize()
    );
    await solanaConnection.confirmTransaction(txid);
    const solanaTransferTxResult = await solanaConnection.getTransaction(txid);
    assertIsNotNull(solanaTransferTxResult);

    // observe tx and get vaa
    const solanaTransferVaa = await getSignedVaaSolana(solanaTransferTxResult);
    const solanaTransferVaaParsed = parseNftTransferVaa(solanaTransferVaa);

    // redeem SPL on Aptos
    const aptosRedeemTxPayload = await redeemOnAptos(
      APTOS_NFT_BRIDGE_ADDRESS,
      solanaTransferVaa
    );
    const aptosRedeemTx = await generateSignAndSubmitEntryFunction(
      aptosClient,
      aptosAccount,
      aptosRedeemTxPayload
    );
    const aptosRedeemTxResult = (await aptosClient.waitForTransactionWithResult(
      aptosRedeemTx.hash
    )) as Types.UserTransaction;
    expect(aptosRedeemTxResult.success).toBe(true);

    // check if token is in SPL cache
    const tokenData = await getForeignAssetAptos(
      aptosClient,
      APTOS_NFT_BRIDGE_ADDRESS,
      CHAIN_ID_SOLANA,
      new Uint8Array(solanaTransferVaaParsed.tokenAddress)
    );
    assertIsNotNull(tokenData);
    expect(tokenData.collectionName).toBe("Wormhole Bridged Solana-NFT"); // this will change if SPL cache is deprecated in favor of separate collections

    // check if token is in user's account
    const events = (await aptosClient.getEventsByEventHandle(
      aptosAccount.address(),
      "0x3::token::TokenStore",
      "deposit_events",
      { limit: 1 }
    )) as DepositEvent[];
    expect(events.length).toBe(1);
    expect(events[0].data.id.token_data_id.name).toBe(
      tryNativeToHexString(TEST_SOLANA_TOKEN, CHAIN_ID_SOLANA)
    );
  });
});

// https://github.com/microsoft/TypeScript/issues/34523
const assertIsNotNull: <T>(x: T | null) => asserts x is T = (x) => {
  expect(x).not.toBeNull();
};
