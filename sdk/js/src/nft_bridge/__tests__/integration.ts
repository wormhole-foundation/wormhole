import { beforeEach, describe, expect, jest, test } from "@jest/globals";
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
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CONTRACTS,
  ChainId,
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
} from "./utils/consts";
import { getSignedVaaEthereum, getSignedVaaSolana } from "./utils/getSignedVaa";

// ethereum setup
const web3 = new Web3(ETH_NODE_URL);
let provider: ethers.providers.JsonRpcProvider;
let signer: ethers.Wallet;

// solana setup
const connection = new Connection(SOLANA_HOST, "confirmed");
const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
const payerAddress = keypair.publicKey.toString();

beforeEach(() => {
  provider = new ethers.providers.JsonRpcProvider(ETH_NODE_URL);
  signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider); // corresponds to accounts[1]
});

describe("Integration Tests", () => {
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
        let signedVAA = await getSignedVaaSolana(transaction1);

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
        await provider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
        signedVAA = await getSignedVaaEthereum(transaction3);

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
