import axios from "axios";
import Web3 from "web3";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  jest,
  test,
} from "@jest/globals";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { BigNumber, BigNumberish, ethers } from "ethers";
import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_TERRA,
  CONTRACTS,
  getEmitterAddressEth,
  getEmitterAddressTerra,
  parseSequenceFromLogEth,
  parseSequenceFromLogTerra,
  nft_bridge,
  parseSequenceFromLogSolana,
  getEmitterAddressSolana,
  CHAIN_ID_SOLANA,
  parseNFTPayload,
} from "../..";
import getSignedVAAWithRetry from "../../rpc/getSignedVAAWithRetry";
import { importCoreWasm, setDefaultWasm } from "../../solana/wasm";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  TERRA_GAS_PRICES_URL,
  WORMHOLE_RPC_HOSTS,
  TERRA_CW721_CODE_ID,
  TERRA_NODE_URL,
  TERRA_CHAIN_ID,
  TERRA_PRIVATE_KEY,
  SOLANA_PRIVATE_KEY,
  TEST_SOLANA_TOKEN,
  SOLANA_HOST,
} from "./consts";
import {
  NFTImplementation,
  NFTImplementation__factory,
} from "../../ethers-contracts";
import sha3 from "js-sha3";
import {
  Connection,
  Keypair,
  PublicKey,
  TransactionResponse,
} from "@solana/web3.js";
import { postVaaSolanaWithRetry } from "../../solana";
import { tryNativeToUint8Array } from "../../utils";
import { arrayify } from "ethers/lib/utils";
const ERC721 = require("@openzeppelin/contracts/build/contracts/ERC721PresetMinterPauserAutoId.json");

setDefaultWasm("node");

jest.setTimeout(60000);

type Address = string;

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
        const { parse_vaa } = await importCoreWasm();

        const fromAddress = await Token.getAssociatedTokenAddress(
          ASSOCIATED_TOKEN_PROGRAM_ID,
          TOKEN_PROGRAM_ID,
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
        let signedVAA = await waitUntilSolanaTxObserved(transaction1);

        // we get the solana token id from the VAA
        const { tokenId } = parseNFTPayload(
          Buffer.from(new Uint8Array(parse_vaa(signedVAA).payload))
        );

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
        signedVAA = await waitUntilEthTxObserved(transaction3);

        const { name, symbol } = parseNFTPayload(
          Buffer.from(new Uint8Array(parse_vaa(signedVAA).payload))
        );

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
  let sequence = parseSequenceFromLogEth(
    receipt,
    CONTRACTS.DEVNET.ethereum.core
  );
  let emitterAddress = getEmitterAddressEth(
    CONTRACTS.DEVNET.ethereum.nft_bridge
  );
  // poll until the guardian(s) witness and sign the vaa
  const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    CHAIN_ID_ETH,
    emitterAddress,
    sequence,
    {
      transport: NodeHttpTransport(),
    }
  );
  return signedVAA;
}

async function waitUntilSolanaTxObserved(
  response: TransactionResponse
): Promise<Uint8Array> {
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence = parseSequenceFromLogSolana(response);
  const emitterAddress = await getEmitterAddressSolana(
    CONTRACTS.DEVNET.solana.nft_bridge
  );
  // poll until the guardian(s) witness and sign the vaa
  const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    CHAIN_ID_SOLANA,
    emitterAddress,
    sequence,
    {
      transport: NodeHttpTransport(),
    }
  );
  return signedVAA;
}

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
