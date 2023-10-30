import { describe, expect, jest, test } from "@jest/globals";
import {
  createAssociatedTokenAccountInstruction,
  getAssociatedTokenAddress,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import {
  Connection,
  Keypair,
  PublicKey,
  TokenAccountsFilter,
  Transaction,
} from "@solana/web3.js";
import { ethers } from "ethers";

import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  TEST_ERC20,
  WORMHOLE_RPC_HOSTS,
} from "./utils/consts";

import {
  CONFIG as CONNECT_CONFIG,
  TokenBridge,
  nativeChainAddress,
  normalizeAmount,
  signSendWait,
  api,
  encoding,
  WormholeMessageId
} from "@wormhole-foundation/connect-sdk";
import { SolanaTokenBridge } from "../solana/src";
import { EvmTokenBridge } from "../evm/src";
import { EvmPlatform, getEvmSigner } from "@wormhole-foundation/connect-sdk-evm";
import { SolanaPlatform, getSolanaSigner } from "@wormhole-foundation/connect-sdk-solana";

jest.setTimeout(60000);

async function getEthTokenBridge(provider: ethers.Provider): Promise<TokenBridge<'Evm'>> {
  return EvmTokenBridge.fromProvider(provider, CONNECT_CONFIG.Devnet.chains)
}

async function getSolTokenBridge(connection: Connection): Promise<TokenBridge<'Solana'>> {
  return SolanaTokenBridge.fromProvider(connection, CONNECT_CONFIG.Devnet.chains)
}

async function transferFromEthToSolana(): Promise<WormholeMessageId> {
  // create a keypair for Solana
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);


  const solTokenBridge = await getSolTokenBridge(connection)

  const token = nativeChainAddress(["Ethereum", TEST_ERC20])

  const solanaMintKey = (await solTokenBridge.getWrappedAsset(token)).unwrap()

  const recipient = await getAssociatedTokenAddress(
    solanaMintKey,
    keypair.publicKey
  );

  // create the associated token account if it doesn't exist
  const associatedAddressInfo = await connection.getAccountInfo(recipient);

  if (!associatedAddressInfo) {
    const transaction = new Transaction().add(
      createAssociatedTokenAccountInstruction(
        keypair.publicKey, // payer
        recipient,
        keypair.publicKey, // owner
        solanaMintKey
      )
    );
    const { blockhash } = await connection.getLatestBlockhash();
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = keypair.publicKey;
    // sign, send, and confirm transaction
    transaction.partialSign(keypair);
    const txid = await connection.sendRawTransaction(transaction.serialize());
    await connection.confirmTransaction(txid);
  }

  const provider = new ethers.WebSocketProvider(ETH_NODE_URL);

  const ethTokenBridge = await getEthTokenBridge(provider)

  // create a signer for Eth
  const signer = await getEvmSigner(provider, ETH_PRIVATE_KEY);

  // Get decimals from the token itself
  const amount = normalizeAmount("1", 18n);

  // Format for connect-sdk
  const receiver = nativeChainAddress(["Solana", recipient.toString()])

  // Create transfer transaction generator
  const xfer = ethTokenBridge.transfer(signer.address(), receiver, TEST_ERC20, amount)

  // Sign, send, and wait for confirmation
  const ethChain = EvmPlatform.getChain("Ethereum")
  const txids = await signSendWait(ethChain, xfer, signer)

  // Parse the wormhole message out with sequence info
  const [whm] = await ethChain.parseTransaction(txids[txids.length - 1].txid)
  return whm
}

describe("Ethereum to Solana and Back", () => {
  test("Attest Ethereum ERC-20 to Solana", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.WebSocketProvider(ETH_NODE_URL);
        const ethSigner = await getEvmSigner(provider, ETH_PRIVATE_KEY);

        const ethChain = EvmPlatform.getChain("Ethereum")
        const ethTb = await getEthTokenBridge(provider)

        const attestTxs = ethTb.createAttestation(TEST_ERC20)
        const txids = await signSendWait(ethChain, attestTxs, ethSigner)

        const [whm] = await ethChain.parseTransaction(txids[txids.length - 1].txid)

        // poll until the guardian(s) witness and sign the vaa
        const vaa = await api.getVaaWithRetry(WORMHOLE_RPC_HOSTS[0], whm, "TokenBridge:AttestMeta")

        // Submit the VAA to Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const solSigner = await getSolanaSigner(connection, encoding.b58.encode(SOLANA_PRIVATE_KEY))

        const solChain = SolanaPlatform.getChain("Solana");
        const solTb = await getSolTokenBridge(connection)

        const submitTxs = solTb.submitAttestation(vaa!, solSigner.address())

        await signSendWait(solChain, submitTxs, solSigner)

        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done(
          "An error occurred while trying to attest from Ethereum to Solana"
        );
      }
    })();
  });
  test("Ethereum ERC-20 is attested on Solana", async () => {
    const connection = new Connection(SOLANA_HOST, "confirmed");
    const solTb = await getSolTokenBridge(connection)
    const address = solTb.getWrappedAsset(nativeChainAddress(["Ethereum", TEST_ERC20]))
    expect(address).toBeTruthy();
  });
  test("Send Ethereum ERC-20 to Solana", (done) => {
    (async () => {
      try {
        const DECIMALS: number = 18;

        const solChain = SolanaPlatform.getChain("Solana")
        const ethChain = EvmPlatform.getChain("Ethereum")

        // create a keypair for Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const solSigner = await getSolanaSigner(connection, encoding.b58.encode(SOLANA_PRIVATE_KEY))

        const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
        const payerAddress = keypair.publicKey.toString();

        // determine destination address - an associated token account
        const tokenId = nativeChainAddress(["Ethereum", TEST_ERC20])

        const solTb = await getSolTokenBridge(connection)

        const SolanaForeignAsset = await solTb.getWrappedAsset(tokenId);
        const solanaMintKey = new PublicKey(SolanaForeignAsset || "");
        const recipient = await getAssociatedTokenAddress(
          solanaMintKey,
          keypair.publicKey,
        );

        // create the associated token account if it doesn't exist
        const associatedAddressInfo = await connection.getAccountInfo(
          recipient
        );

        if (!associatedAddressInfo) {
          const transaction = new Transaction().add(
            createAssociatedTokenAccountInstruction(
              keypair.publicKey, // payer
              recipient,
              keypair.publicKey, // owner
              solanaMintKey
            )
          );
          const { blockhash } = await connection.getLatestBlockhash();
          transaction.recentBlockhash = blockhash;
          transaction.feePayer = keypair.publicKey;
          // sign, send, and confirm transaction
          transaction.partialSign(keypair);
          const txid = await connection.sendRawTransaction(
            transaction.serialize()
          );
          await connection.confirmTransaction(txid);
        }
        // create a signer for Eth
        const provider = new ethers.WebSocketProvider(ETH_NODE_URL);
        const ethSigner = await getEvmSigner(provider, ETH_PRIVATE_KEY);
        const amount = normalizeAmount("1", BigInt(DECIMALS));

        // Get the initial wallet balance of ERC20 on Eth
        const token = EvmPlatform.getTokenImplementation(provider, TEST_ERC20)
        const initialErc20BalOnEth = await token.balanceOf(ethSigner.address());

        // Get the initial balance on Solana
        const tokenFilter: TokenAccountsFilter = {
          programId: TOKEN_PROGRAM_ID,
        };

        let results = await connection.getParsedTokenAccountsByOwner(
          keypair.publicKey,
          tokenFilter
        );

        let initialSolanaBalance: number = 0;
        for (const item of results.value) {
          const tokenInfo = item.account.data.parsed.info;
          const address = tokenInfo.mint;
          const amount = tokenInfo.tokenAmount.uiAmount;
          if (tokenInfo.mint === SolanaForeignAsset) {
            initialSolanaBalance = amount;
          }
        }

        const ethTb = await getEthTokenBridge(provider);
        const xfer = ethTb.transfer(ethSigner.address(), nativeChainAddress(["Solana", recipient.toBase58()]), TEST_ERC20, amount)

        const txids = await signSendWait(ethChain, xfer, ethSigner)

        const [whm] = await ethChain.parseTransaction(txids[txids.length - 1].txid)

        const vaa = await api.getVaaWithRetry(WORMHOLE_RPC_HOSTS[0], whm, "TokenBridge:Transfer")

        expect(await solTb.isTransferCompleted(vaa!)).toBe(false);

        // redeem tokens on solana
        const redeemTxs = solTb.redeem(payerAddress, vaa!)

        // sign, send, and confirm transaction
        await signSendWait(solChain, redeemTxs, solSigner)


        expect(await solTb.isTransferCompleted(vaa!)).toBe(true);

        // Get the final wallet balance of ERC20 on Eth
        const finalErc20BalOnEth = await token.balanceOf(ethSigner.address());

        expect(initialErc20BalOnEth - finalErc20BalOnEth === amount).toBe(true);

        // Get final balance on Solana
        results = await connection.getParsedTokenAccountsByOwner(
          keypair.publicKey,
          tokenFilter
        );
        let finalSolanaBalance: number = 0;
        for (const item of results.value) {
          const tokenInfo = item.account.data.parsed.info;
          const address = tokenInfo.mint;
          const amount = tokenInfo.tokenAmount.uiAmount;
          if (tokenInfo.mint === SolanaForeignAsset) {
            finalSolanaBalance = amount;
          }
        }
        expect(finalSolanaBalance - initialSolanaBalance === 1).toBe(true);
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to send from Ethereum to Solana");
      }
    })();
  });
  // describe("Post VAA with retry", () => {
  //   test("postVAA with retry, no failures", (done) => {
  //     (async () => {
  //       try {
  //         // create a keypair for Solana
  //         const connection = new Connection(SOLANA_HOST, "confirmed");
  //         const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
  //         const payerAddress = keypair.publicKey.toString();
  //         const whm = await transferFromEthToSolana();
  //         const vaa = await api.getVaaWithRetry(WORMHOLE_RPC_HOSTS[0], whm, "TokenBridge:Transfer")


  //         const solTb = await getSolTokenBridge(connection)

  //         let maxFailures = 0;
  //         // post vaa to Solana

  //         const postPromise = postVaaWithRetry(
  //           connection,
  //           async (transaction) => {
  //             await new Promise(function (resolve) {
  //               //We delay here so the connection has time to get wrecked
  //               setTimeout(function () {
  //                 resolve(500);
  //               });
  //             });
  //             transaction.partialSign(keypair);
  //             return transaction;
  //           },
  //           CONTRACTS.DEVNET.solana.core,
  //           payerAddress,
  //           Buffer.from(signedVAA),
  //           maxFailures
  //         );

  //         await postPromise;
  //         // redeem tokens on solana
  //         const transaction = await redeemOnSolana(
  //           connection,
  //           CONTRACTS.DEVNET.solana.core,
  //           CONTRACTS.DEVNET.solana.token_bridge,
  //           payerAddress,
  //           signedVAA
  //         );
  //         // sign, send, and confirm transaction
  //         transaction.partialSign(keypair);
  //         const txid = await connection.sendRawTransaction(
  //           transaction.serialize()
  //         );
  //         await connection.confirmTransaction(txid);

  //         expect(await solTb.isTransferCompleted(vaa!)).toBe(true);
  //         done();
  //       } catch (e) {
  //         console.error(e);
  //         done(
  //           "An error occurred while happy-path testing post VAA with retry."
  //         );
  //       }
  //     })();
  //   });
  //   test("Reject on signature failure", (done) => {
  //     (async () => {
  //       try {
  //         // create a keypair for Solana
  //         const connection = new Connection(SOLANA_HOST, "confirmed");
  //         const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
  //         const payerAddress = keypair.publicKey.toString();
  //         const sequence = await transferFromEthToSolana();
  //         const emitterAddress = getEmitterAddressEth(
  //           CONTRACTS.DEVNET.ethereum.token_bridge
  //         );
  //         // poll until the guardian(s) witness and sign the vaa
  //         const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
  //           WORMHOLE_RPC_HOSTS,
  //           CHAIN_ID_ETH,
  //           emitterAddress,
  //           sequence,
  //           {
  //             transport: NodeHttpTransport(),
  //           }
  //         );
  //         let maxFailures = 5;
  //         // post vaa to Solana

  //         let error = false;
  //         try {
  //           const postPromise = postVaaWithRetry(
  //             connection,
  //             async (transaction) => {
  //               return Promise.reject();
  //             },
  //             CONTRACTS.DEVNET.solana.core,
  //             payerAddress,
  //             Buffer.from(signedVAA),
  //             maxFailures
  //           );

  //           await postPromise;
  //         } catch (e) {
  //           error = true;
  //         }
  //         expect(error).toBe(true);
  //         done();
  //       } catch (e) {
  //         console.error(e);
  //         done(
  //           "An error occurred while trying to send from Ethereum to Solana"
  //         );
  //       }
  //     })();
  //   });
  // });
});
