import {
  afterAll,
  beforeEach,
  describe,
  expect,
  jest,
  test,
} from "@jest/globals";
import { BN } from "@project-serum/anchor";
import { getAssociatedTokenAddress } from "@solana/spl-token";
import { Connection, Keypair, PublicKey } from "@solana/web3.js";
import { AptosAccount, AptosClient, FaucetClient, Types } from "aptos";
import { ethers } from "ethers";
import Web3 from "web3";
import { DepositEvent } from "../../aptos/types";
import {
  CHAIN_ID_APTOS,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CONTRACTS,
  deriveCollectionHashFromTokenId,
  deriveTokenHashFromTokenId,
  ensureHexPrefix,
  generateSignAndSubmitEntryFunction,
  tryNativeToHexString,
  tryNativeToUint8Array,
} from "../../utils";
import { parseNftTransferVaa } from "../../vaa";
import { getForeignAssetAptos, getForeignAssetEth } from "../getForeignAsset";
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
  ETH_PRIVATE_KEY8,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY2,
  TEST_SOLANA_TOKEN3,
} from "./utils/consts";
import {
  deployTestNftOnAptos,
  deployTestNftOnEthereum,
} from "./utils/deployTestNft";
import {
  getSignedVaaAptos,
  getSignedVaaEthereum,
  getSignedVaaSolana,
} from "./utils/getSignedVaa";

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
const ethProvider = new ethers.providers.JsonRpcProvider(ETH_NODE_URL);
const ethSigner = new ethers.Wallet(ETH_PRIVATE_KEY8, ethProvider);

// solana setup
const solanaConnection = new Connection(SOLANA_HOST, "confirmed");
const solanaKeypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY2);
const solanaPayerAddress = solanaKeypair.publicKey.toString();

beforeEach(async () => {
  aptosClient = new AptosClient(APTOS_NODE_URL);
  aptosAccount = new AptosAccount();
  faucet = new FaucetClient(APTOS_NODE_URL, APTOS_FAUCET_URL);
  await faucet.fundAccount(aptosAccount.address(), 100_000_000);
});

afterAll(async () => {
  (web3.currentProvider as any).disconnect();
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
    await ethProvider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`

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
      getIsTransferCompletedAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        ethTransferVaa
      )
    ).resolves.toBe(true);

    // get token data
    const tokenId = await getForeignAssetAptos(
      aptosClient,
      APTOS_NFT_BRIDGE_ADDRESS,
      CHAIN_ID_ETH,
      tryNativeToUint8Array(ethNft.address, CHAIN_ID_ETH),
      BigInt(10)
    );
    assertIsNotNull(tokenId);
    expect(
      getIsWrappedAssetAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        tokenId.token_data_id.creator
      )
    ).resolves.toBe(true);
    expect(
      getOriginalAssetAptos(aptosClient, APTOS_NFT_BRIDGE_ADDRESS, tokenId)
    ).resolves.toStrictEqual({
      isWrapped: true,
      chainId: CHAIN_ID_ETH,
      assetAddress: Uint8Array.from(ethTransferVaaParsed.tokenAddress),
      tokenId: ensureHexPrefix(
        ethTransferVaaParsed.tokenId.toString(16).padStart(64, "0")
      ),
    });

    // transfer NFT from Aptos back to Ethereum
    const aptosTransferPayload = await transferFromAptos(
      APTOS_NFT_BRIDGE_ADDRESS,
      tokenId.token_data_id.creator,
      tokenId.token_data_id.collection,
      tokenId.token_data_id.name.padStart(64, "0"),
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
    expect(
      await getIsWrappedAssetAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        aptosAccount.address().toString()
      )
    ).toBe(false);

    // get token data from user wallet
    const event = (
      await aptosClient.getEventsByEventHandle(
        aptosAccount.address(),
        "0x3::token::TokenStore",
        "deposit_events",
        { limit: 1 }
      )
    )[0] as DepositEvent;
    const depositTokenId = event.data.id;

    // transfer NFT from Aptos to Ethereum
    const aptosTransferPayload = await transferFromAptos(
      APTOS_NFT_BRIDGE_ADDRESS,
      depositTokenId.token_data_id.creator,
      depositTokenId.token_data_id.collection,
      depositTokenId.token_data_id.name,
      Number(depositTokenId.property_version),
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
    expect(ethRedeemTx.status).toBe(1);

    // sanity check token hash and id
    const tokenHash = await deriveTokenHashFromTokenId(depositTokenId);
    expect(Buffer.from(tokenHash).toString("hex")).toBe(
      new BN(aptosTransferVaaParsed.tokenId.toString())
        .toString("hex")
        .padStart(64, "0") // conversion to BN strips leading zeros
    );

    const foreignAssetTokenId = await getForeignAssetAptos(
      aptosClient,
      APTOS_NFT_BRIDGE_ADDRESS,
      CHAIN_ID_APTOS,
      tokenHash
    );
    assertIsNotNull(foreignAssetTokenId);
    expect(foreignAssetTokenId).toStrictEqual(depositTokenId);

    // get token address on Ethereum
    const tokenAddressAptos = await deriveCollectionHashFromTokenId(
      foreignAssetTokenId
    );
    const tokenAddressEth = await getForeignAssetEth(
      ETH_NFT_BRIDGE_ADDRESS,
      ethSigner,
      CHAIN_ID_APTOS,
      tokenAddressAptos
    );
    assertIsNotNull(tokenAddressEth);

    // transfer NFT from Ethereum back to Aptos
    const ethTransferTx = await transferFromEth(
      ETH_NFT_BRIDGE_ADDRESS,
      ethSigner,
      tokenAddressEth,
      tokenHash,
      CHAIN_ID_APTOS,
      tryNativeToUint8Array(aptosAccount.address().toString(), CHAIN_ID_APTOS)
    );
    expect(ethTransferTx.status).toBe(1);
    await ethProvider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`

    // observe tx and get vaa
    const ethTransferVaa = await getSignedVaaEthereum(ethTransferTx);
    const ethTransferVaaParsed = parseNftTransferVaa(ethTransferVaa);
    expect(ethTransferVaaParsed.name).toBe(APTOS_COLLECTION_NAME);

    // redeem NFT on Aptos
    const aptosRedeemTxPayload = await redeemOnAptos(
      APTOS_NFT_BRIDGE_ADDRESS,
      ethTransferVaa
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
    expect(
      getIsTransferCompletedAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        ethTransferVaa
      )
    ).resolves.toBe(true);
    expect(
      getOriginalAssetAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        foreignAssetTokenId
      )
    ).resolves.toStrictEqual({
      isWrapped: false,
      chainId: CHAIN_ID_APTOS,
      assetAddress: tokenAddressAptos,
      tokenId: ensureHexPrefix(
        ethTransferVaaParsed.tokenId.toString(16).padStart(64, "0")
      ),
    });
  });

  test("Transfer Solana SPL to Aptos", async () => {
    // transfer SPL token to Aptos
    const fromAddress = await getAssociatedTokenAddress(
      new PublicKey(TEST_SOLANA_TOKEN3),
      solanaKeypair.publicKey
    );
    const solanaTransferTx = await transferFromSolana(
      solanaConnection,
      SOLANA_CORE_BRIDGE_ADDRESS,
      SOLANA_NFT_BRIDGE_ADDRESS,
      solanaPayerAddress,
      fromAddress.toString(),
      TEST_SOLANA_TOKEN3,
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
      new Uint8Array(solanaTransferVaaParsed.tokenAddress),
      solanaTransferVaaParsed.tokenId
    );
    assertIsNotNull(tokenData);
    expect(tokenData.token_data_id.collection).toBe(
      "Wormhole Bridged Solana-NFT" // this will change if SPL cache is deprecated in favor of separate collections
    );

    // check if token is in user's account
    const events = (await aptosClient.getEventsByEventHandle(
      aptosAccount.address(),
      "0x3::token::TokenStore",
      "deposit_events",
      { limit: 1 }
    )) as DepositEvent[];
    expect(events.length).toBe(1);
    expect(events[0].data.id.token_data_id.name).toBe(
      tryNativeToHexString(TEST_SOLANA_TOKEN3, CHAIN_ID_SOLANA)
    );
  });

  test("Transfer multiple tokens from same collection from Ethereum to Aptos", async () => {
    const ETH_COLLECTION_NAME = "Test APE 🐒";

    // create NFTs on Ethereum
    const ethNfts = await deployTestNftOnEthereum(
      web3,
      ethSigner,
      ETH_COLLECTION_NAME,
      "APE🐒",
      "https://cloudflare-ipfs.com/ipfs/QmeSjSinHpPnmXmspMjwiXyN6zS4E9zccariGR3jxcaWtq/",
      2
    );

    // transfer 2 NFTs from Ethereum to Aptos
    const ethTransferTx1 = await transferFromEth(
      ETH_NFT_BRIDGE_ADDRESS,
      ethSigner,
      ethNfts.address,
      0,
      CHAIN_ID_APTOS,
      aptosAccount.address().toUint8Array()
    );
    const ethTransferTx2 = await transferFromEth(
      ETH_NFT_BRIDGE_ADDRESS,
      ethSigner,
      ethNfts.address,
      1,
      CHAIN_ID_APTOS,
      aptosAccount.address().toUint8Array()
    );
    await ethProvider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`

    // observe txs and get vaas
    const ethTransferVaa1 = await getSignedVaaEthereum(ethTransferTx1);
    const ethTransferVaa2 = await getSignedVaaEthereum(ethTransferTx2);

    // redeem NFTs on Aptos
    const aptosRedeemPayload1 = await redeemOnAptos(
      APTOS_NFT_BRIDGE_ADDRESS,
      ethTransferVaa1
    );
    const aptosRedeemTx1 = await generateSignAndSubmitEntryFunction(
      aptosClient,
      aptosAccount,
      aptosRedeemPayload1
    );
    await aptosClient.waitForTransactionWithResult(aptosRedeemTx1.hash);

    const aptosRedeemPayload2 = await redeemOnAptos(
      APTOS_NFT_BRIDGE_ADDRESS,
      ethTransferVaa2
    );
    const aptosRedeemTx2 = await generateSignAndSubmitEntryFunction(
      aptosClient,
      aptosAccount,
      aptosRedeemPayload2
    );
    await aptosClient.waitForTransactionWithResult(aptosRedeemTx2.hash);

    // get token ids
    const tokenId1 = await getForeignAssetAptos(
      aptosClient,
      APTOS_NFT_BRIDGE_ADDRESS,
      CHAIN_ID_ETH,
      tryNativeToUint8Array(ethNfts.address, CHAIN_ID_ETH),
      BigInt(0)
    );
    const tokenId2 = await getForeignAssetAptos(
      aptosClient,
      APTOS_NFT_BRIDGE_ADDRESS,
      CHAIN_ID_ETH,
      tryNativeToUint8Array(ethNfts.address, CHAIN_ID_ETH),
      BigInt(1)
    );
    assertIsNotNull(tokenId1);
    assertIsNotNull(tokenId2);
    expect(tokenId1.property_version).toBe("0");
    expect(tokenId2.property_version).toBe("0");
    expect(tokenId1.token_data_id.collection).toBe(
      tokenId2.token_data_id.collection
    );
    expect(tokenId1.token_data_id.creator).toBe(tokenId2.token_data_id.creator);
    expect(tokenId1.token_data_id.name).not.toBe(tokenId2.token_data_id.name);

    // check if token that does not exist correctly returns null foreign asset address
    expect(
      await getForeignAssetAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        CHAIN_ID_ETH,
        tryNativeToUint8Array(ethNfts.address, CHAIN_ID_ETH),
        BigInt(2)
      )
    ).toBe(null);
  });
});

// https://github.com/microsoft/TypeScript/issues/34523
const assertIsNotNull: <T>(x: T | null) => asserts x is T = (x) => {
  expect(x).not.toBeNull();
};
