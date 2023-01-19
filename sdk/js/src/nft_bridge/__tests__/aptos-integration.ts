import { beforeAll, describe, expect, jest, test } from "@jest/globals";
import { AptosAccount, AptosClient, FaucetClient, Types } from "aptos";
import { ethers } from "ethers";
import Web3 from "web3";
import {
  CHAIN_ID_APTOS,
  CHAIN_ID_ETH,
  CONTRACTS,
  deriveResourceAccountAddress,
  generateSignAndSubmitEntryFunction,
  tryNativeToUint8Array,
} from "../../utils";
import { parseNftTransferVaa } from "../../vaa";
import { getIsTransferCompletedAptos } from "../getIsTransferCompleted";
import { getIsWrappedAssetAptos } from "../getIsWrappedAsset";
import { getOriginalAssetAptos } from "../getOriginalAsset";
import { redeemOnAptos } from "../redeem";
import { transferFromAptos, transferFromEth } from "../transfer";
import {
  APTOS_FAUCET_URL,
  APTOS_NODE_URL,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
} from "./consts";
import { deployTestNftOnEthereum } from "./utils/deployTestNft";
import { getSignedVaaAptos, getSignedVaaEthereum } from "./utils/getSignedVaa";

jest.setTimeout(60000);

const APTOS_NFT_BRIDGE_ADDRESS = CONTRACTS.DEVNET.aptos.nft_bridge;
const ETH_NFT_BRIDGE_ADDRESS = CONTRACTS.DEVNET.ethereum.nft_bridge;

let aptosClient: AptosClient;
let aptosAccount: AptosAccount;
let web3: Web3;
let ethProvider: ethers.providers.WebSocketProvider;
let ethSigner: ethers.Wallet;

beforeAll(async () => {
  // aptos setup
  aptosClient = new AptosClient(APTOS_NODE_URL);
  aptosAccount = new AptosAccount();
  const faucet = new FaucetClient(APTOS_NODE_URL, APTOS_FAUCET_URL);
  await faucet.fundAccount(aptosAccount.address(), 100_000_000);

  // ethereum setup
  web3 = new Web3(ETH_NODE_URL);
  ethProvider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
  ethSigner = new ethers.Wallet(ETH_PRIVATE_KEY, ethProvider); // corresponds to accounts[1]
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

    // get creator address
    const creatorAddress = await deriveResourceAccountAddress(
      APTOS_NFT_BRIDGE_ADDRESS,
      CHAIN_ID_ETH,
      ethNft.address
    );
    expect(creatorAddress).toBeTruthy();
    expect(
      await getIsWrappedAssetAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        creatorAddress!
      )
    ).toBe(true);
    expect(
      await getOriginalAssetAptos(
        aptosClient,
        APTOS_NFT_BRIDGE_ADDRESS,
        creatorAddress!
      )
    ).toMatchObject({
      isWrapped: true,
      chainId: CHAIN_ID_ETH,
      assetAddress: Uint8Array.from(ethTransferVaaParsed.tokenAddress),
    });

    // transfer NFT from Aptos back to Ethereum
    const aptosTransferPayload = await transferFromAptos(
      APTOS_NFT_BRIDGE_ADDRESS,
      creatorAddress!,
      ethTransferVaaParsed.name, // TODO(aki): derive this properly
      ethTransferVaaParsed.tokenId.toString(16).padStart(64, "0"), // TODO(aki): derive this properly
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
    const aptosTransferVaa = await getSignedVaaAptos(
      aptosClient,
      aptosTransferTxResult
    );
    const aptosTransferVaaParsed = parseNftTransferVaa(aptosTransferVaa);
    expect(aptosTransferVaaParsed.name).toBe(ETH_COLLECTION_NAME);
    expect(aptosTransferVaaParsed.tokenAddress.toString("hex")).toBe(
      ethTransferVaaParsed.tokenAddress.toString("hex")
    );

    // TODO(aki): make this work
    // // redeem NFT on Ethereum & check NFT is the same
    // const ethRedeemTxResult = await redeemOnEth(
    //   ETH_NFT_BRIDGE_ADDRESS,
    //   ethSigner,
    //   aptosTransferVaa,
    //   { gasLimit: 3e7 }
    // );
    // console.log(
    //   "ethRedeemTxResult",
    //   JSON.stringify(ethRedeemTxResult, null, 2)
    // );
    // expect(ethRedeemTxResult.status).toBe(1);
  });
});
function afterAll(arg0: () => Promise<void>) {
  throw new Error("Function not implemented.");
}
