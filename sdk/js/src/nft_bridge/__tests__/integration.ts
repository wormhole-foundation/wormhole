import {
  afterEach,
  beforeEach,
  describe,
  expect,
  jest,
  test,
} from "@jest/globals";
import { getAssociatedTokenAddress } from "@solana/spl-token";
import {
  Connection,
  Keypair,
  PublicKey,
  TransactionResponse,
} from "@solana/web3.js";
import { BigNumberish, ethers } from "ethers";
import Web3 from "web3";
import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CONTRACTS,
  nft_bridge,
} from "../..";
import { postVaaSolanaWithRetry } from "../../solana";
import { tryNativeToUint8Array } from "../../utils";
import { parseNftTransferVaa } from "../../vaa";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  TEST_SOLANA_TOKEN,
} from "./consts";
import {
  waitUntilTransactionObservedEthereum,
  waitUntilTransactionObservedSolana,
} from "./utils/waitUntilTransactionObserved";

jest.setTimeout(60000);

// ethereum setup
const web3 = new Web3(ETH_NODE_URL);
let provider: ethers.providers.WebSocketProvider;
let signer: ethers.Wallet;

// solana setup
const connection = new Connection(SOLANA_HOST, "confirmed");
const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
const payerAddress = keypair.publicKey.toString();

beforeEach(() => {
  provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
  signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider); // corresponds to accounts[1]
});

afterEach(() => {
  provider.destroy();
});

describe("Integration Tests", () => {
  // TODO: figure out why this isn't working
  // test("Send Ethereum ERC-721 to Solana and back", (done) => {
  //   (async () => {
  //     try {
  //       const erc721 = await deployNFTOnEth(
  //         "Not an APE 🐒",
  //         "APE🐒",
  //         "https://cloudflare-ipfs.com/ipfs/QmeSjSinHpPnmXmspMjwiXyN6zS4E9zccariGR3jxcaWtq/",
  //         11 // mint ids 0..10 (inclusive)
  //       );
  //       const sol_addr = await nft_bridge.getForeignAssetSol(
  //         CONTRACTS.DEVNET.solana.nft_bridge,
  //         CHAIN_ID_ETH,
  //         tryNativeToUint8Array(erc721.address, CHAIN_ID_ETH),
  //         arrayify(BigNumber.from("10"))
  //       );
  //       const fromAddress = await Token.getAssociatedTokenAddress(
  //         ASSOCIATED_TOKEN_PROGRAM_ID,
  //         TOKEN_PROGRAM_ID,
  //         new PublicKey(sol_addr),
  //         keypair.publicKey
  //       );
  //       const transaction = await _transferFromEth(
  //         erc721.address,
  //         10,
  //         fromAddress.toString(),
  //         CHAIN_ID_SOLANA
  //       );
  //       let signedVAA = await waitUntilEthTxObserved(transaction);
  //       await _redeemOnSolana(signedVAA);

  //       let ownerEth = await erc721.ownerOf(10);
  //       expect(ownerEth).not.toBe(signer.address);

  //       // TODO: check wrapped balance

  //       // Send back the NFT to ethereum
  //       const transaction2 = await _transferFromSolana(
  //         fromAddress,
  //         sol_addr,
  //         signer.address,
  //         CHAIN_ID_ETH,
  //         tryNativeToUint8Array(erc721.address, CHAIN_ID_ETH),
  //         CHAIN_ID_ETH,
  //         arrayify(BigNumber.from("10"))
  //       );
  //       signedVAA = await waitUntilSolanaTxObserved(transaction2);
  //       (await expectReceivedOnEth(signedVAA)).toBe(false);
  //       await _redeemOnEth(signedVAA);
  //       (await expectReceivedOnEth(signedVAA)).toBe(true);

  //       // ensure that the transaction roundtrips back to the original native asset
  //       ownerEth = await erc721.ownerOf(10);
  //       expect(ownerEth).toBe(signer.address);

  //       // TODO: the wrapped token should no longer exist

  //       done();
  //     } catch (e) {
  //       console.error(e);
  //       done(
  //         `An error occured while trying to transfer from Ethereum to Solana and back: ${e}`
  //       );
  //     }
  //   })();
  // });
  test("Send Solana SPL to Ethereum and back", (done) => {
    (async () => {
      try {
        const fromAddress = await getAssociatedTokenAddress(
          new PublicKey(TEST_SOLANA_TOKEN),
          keypair.publicKey
        );

        // send to eth
        const transaction1 = await _transferFromSolana(
          fromAddress,
          TEST_SOLANA_TOKEN,
          signer.address,
          CHAIN_ID_ETH
        );
        let signedVAA = await waitUntilTransactionObservedSolana(transaction1);

        // we get the solana token id from the VAA
        const { tokenId } = parseNftTransferVaa(signedVAA);

        await _redeemOnEth(signedVAA);
        const eth_addr = await nft_bridge.getForeignAssetEth(
          CONTRACTS.DEVNET.ethereum.nft_bridge,
          provider,
          CHAIN_ID_SOLANA,
          tryNativeToUint8Array(TEST_SOLANA_TOKEN, CHAIN_ID_SOLANA)
        );
        if (!eth_addr) {
          throw new Error("Ethereum address is null");
        }

        const transaction3 = await _transferFromEth(
          eth_addr,
          tokenId,
          fromAddress.toString(),
          CHAIN_ID_SOLANA
        );
        signedVAA = await waitUntilTransactionObservedEthereum(transaction3);

        const { name, symbol } = parseNftTransferVaa(signedVAA);

        // if the names match up here, it means all the spl caches work
        expect(name).toBe("Not a PUNK🎸");
        expect(symbol).toBe("PUNK🎸");

        await _redeemOnSolana(signedVAA);

        done();
      } catch (e) {
        console.error(e);
        done(
          `An error occured while trying to transfer from Solana to Ethereum: ${e}`
        );
      }
    })();
  });
});

////////////////////////////////////////////////////////////////////////////////
// Utils

async function expectReceivedOnEth(signedVAA: Uint8Array) {
  return expect(
    await nft_bridge.getIsTransferCompletedEth(
      CONTRACTS.DEVNET.ethereum.nft_bridge,
      provider,
      signedVAA
    )
  );
}

async function _transferFromEth(
  erc721: string,
  token_id: BigNumberish,
  address: string,
  chain: ChainId
): Promise<ethers.ContractReceipt> {
  return nft_bridge.transferFromEth(
    CONTRACTS.DEVNET.ethereum.nft_bridge,
    signer,
    erc721,
    token_id,
    chain,
    tryNativeToUint8Array(address, chain)
  );
}

async function _transferFromSolana(
  fromAddress: PublicKey,
  tokenAddress: string,
  targetAddress: string,
  chain: ChainId,
  originAddress?: Uint8Array,
  originChain?: ChainId,
  originTokenId?: Uint8Array
): Promise<TransactionResponse> {
  const transaction = await nft_bridge.transferFromSolana(
    connection,
    CONTRACTS.DEVNET.solana.core,
    CONTRACTS.DEVNET.solana.nft_bridge,
    payerAddress,
    fromAddress.toString(),
    tokenAddress,
    tryNativeToUint8Array(targetAddress, chain),
    chain,
    originAddress,
    originChain,
    originTokenId
  );
  // sign, send, and confirm transaction
  transaction.partialSign(keypair);
  const txid = await connection.sendRawTransaction(transaction.serialize());
  await connection.confirmTransaction(txid);
  const info = await connection.getTransaction(txid);
  if (!info) {
    throw new Error("An error occurred while fetching the transaction info");
  }
  return info;
}

async function _redeemOnEth(
  signedVAA: Uint8Array
): Promise<ethers.ContractReceipt> {
  return nft_bridge.redeemOnEth(
    CONTRACTS.DEVNET.ethereum.nft_bridge,
    signer,
    signedVAA
  );
}

async function _redeemOnSolana(signedVAA: Uint8Array) {
  const maxRetries = 5;
  await postVaaSolanaWithRetry(
    connection,
    async (transaction) => {
      transaction.partialSign(keypair);
      return transaction;
    },
    CONTRACTS.DEVNET.solana.core,
    payerAddress,
    Buffer.from(signedVAA),
    maxRetries
  );
  const transaction = await nft_bridge.redeemOnSolana(
    connection,
    CONTRACTS.DEVNET.solana.core,
    CONTRACTS.DEVNET.solana.nft_bridge,
    payerAddress,
    signedVAA
  );
  transaction.partialSign(keypair);
  const txid = await connection.sendRawTransaction(transaction.serialize());
  await connection.confirmTransaction(txid);
  const info = await connection.getTransaction(txid);
  if (!info) {
    throw new Error("An error occurred while fetching the transaction info");
  }
  return info;
}
