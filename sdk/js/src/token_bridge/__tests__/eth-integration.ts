import { formatUnits, parseUnits } from "@ethersproject/units";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { describe, expect, jest, test } from "@jest/globals";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
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
  approveEth,
  attestFromEth,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CONTRACTS,
  createWrappedOnSolana,
  getEmitterAddressEth,
  getForeignAssetSolana,
  getIsTransferCompletedSolana,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogEth,
  postVaaSolana,
  redeemOnSolana,
  TokenImplementation__factory,
  transferFromEth,
  tryNativeToUint8Array,
} from "../..";
import getSignedVAAWithRetry from "../../rpc/getSignedVAAWithRetry";
import { postVaaWithRetry } from "../../solana/postVaa";
import { setDefaultWasm } from "../../solana/wasm";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  TEST_ERC20,
  WORMHOLE_RPC_HOSTS,
} from "./consts";

setDefaultWasm("node");

jest.setTimeout(60000);

async function transferFromEthToSolana(): Promise<string> {
  // create a keypair for Solana
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
  // determine destination address - an associated token account
  const solanaMintKey = new PublicKey(
    (await getForeignAssetSolana(
      connection,
      CONTRACTS.DEVNET.solana.token_bridge,
      CHAIN_ID_ETH,
      hexToUint8Array(nativeToHexString(TEST_ERC20, CHAIN_ID_ETH) || "")
    )) || ""
  );
  const recipient = await Token.getAssociatedTokenAddress(
    ASSOCIATED_TOKEN_PROGRAM_ID,
    TOKEN_PROGRAM_ID,
    solanaMintKey,
    keypair.publicKey
  );
  // create the associated token account if it doesn't exist
  const associatedAddressInfo = await connection.getAccountInfo(recipient);
  if (!associatedAddressInfo) {
    const transaction = new Transaction().add(
      await Token.createAssociatedTokenAccountInstruction(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        solanaMintKey,
        recipient,
        keypair.publicKey, // owner
        keypair.publicKey // payer
      )
    );
    const { blockhash } = await connection.getRecentBlockhash();
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = keypair.publicKey;
    // sign, send, and confirm transaction
    transaction.partialSign(keypair);
    const txid = await connection.sendRawTransaction(transaction.serialize());
    await connection.confirmTransaction(txid);
  }
  // create a signer for Eth
  const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
  const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
  const amount = parseUnits("1", 18);
  // approve the bridge to spend tokens
  await approveEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    TEST_ERC20,
    signer,
    amount
  );
  // transfer tokens
  const receipt = await transferFromEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    signer,
    TEST_ERC20,
    amount,
    CHAIN_ID_SOLANA,
    hexToUint8Array(
      nativeToHexString(recipient.toString(), CHAIN_ID_SOLANA) || ""
    )
  );
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence = await parseSequenceFromLogEth(
    receipt,
    CONTRACTS.DEVNET.ethereum.core
  );
  provider.destroy();
  return sequence;
}

describe("Ethereum to Solana and Back", () => {
  test("Attest Ethereum ERC-20 to Solana", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
        // attest the test token
        const receipt = await attestFromEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          TEST_ERC20
        );
        // get the sequence from the logs (needed to fetch the vaa)
        const sequence = parseSequenceFromLogEth(
          receipt,
          CONTRACTS.DEVNET.ethereum.core
        );
        const emitterAddress = getEmitterAddressEth(
          CONTRACTS.DEVNET.ethereum.token_bridge
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
        // create a keypair for Solana
        const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
        const payerAddress = keypair.publicKey.toString();
        // post vaa to Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        await postVaaSolana(
          connection,
          async (transaction) => {
            transaction.partialSign(keypair);
            return transaction;
          },
          CONTRACTS.DEVNET.solana.core,
          payerAddress,
          Buffer.from(signedVAA)
        );
        // create wormhole wrapped token (mint and metadata) on solana
        const transaction = await createWrappedOnSolana(
          connection,
          CONTRACTS.DEVNET.solana.core,
          CONTRACTS.DEVNET.solana.token_bridge,
          payerAddress,
          signedVAA
        );
        // sign, send, and confirm transaction
        try {
          transaction.partialSign(keypair);
          const txid = await connection.sendRawTransaction(
            transaction.serialize()
          );
          await connection.confirmTransaction(txid);
        } catch (e) {
          // this could fail because the token is already attested (in an unclean env)
        }
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
    const address = getForeignAssetSolana(
      connection,
      CONTRACTS.DEVNET.solana.token_bridge,
      "ethereum",
      tryNativeToUint8Array(TEST_ERC20, "ethereum")
    );
    expect(address).toBeTruthy();
  });
  test("Send Ethereum ERC-20 to Solana", (done) => {
    (async () => {
      try {
        const DECIMALS: number = 18;
        // create a keypair for Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
        const payerAddress = keypair.publicKey.toString();
        // determine destination address - an associated token account
        const SolanaForeignAsset = await getForeignAssetSolana(
          connection,
          CONTRACTS.DEVNET.solana.token_bridge,
          CHAIN_ID_ETH,
          tryNativeToUint8Array(TEST_ERC20, CHAIN_ID_ETH)
        );
        const solanaMintKey = new PublicKey(SolanaForeignAsset || "");
        const recipient = await Token.getAssociatedTokenAddress(
          ASSOCIATED_TOKEN_PROGRAM_ID,
          TOKEN_PROGRAM_ID,
          solanaMintKey,
          keypair.publicKey
        );
        // create the associated token account if it doesn't exist
        const associatedAddressInfo = await connection.getAccountInfo(
          recipient
        );
        if (!associatedAddressInfo) {
          const transaction = new Transaction().add(
            await Token.createAssociatedTokenAccountInstruction(
              ASSOCIATED_TOKEN_PROGRAM_ID,
              TOKEN_PROGRAM_ID,
              solanaMintKey,
              recipient,
              keypair.publicKey, // owner
              keypair.publicKey // payer
            )
          );
          const { blockhash } = await connection.getRecentBlockhash();
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
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
        const amount = parseUnits("1", DECIMALS);

        // Get the initial wallet balance of ERC20 on Eth
        let token = TokenImplementation__factory.connect(TEST_ERC20, signer);
        const initialErc20BalOnEth = await token.balanceOf(
          await signer.getAddress()
        );
        const initialErc20BalOnEthFormatted = formatUnits(
          initialErc20BalOnEth._hex,
          DECIMALS
        );

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

        // approve the bridge to spend tokens
        await approveEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          TEST_ERC20,
          signer,
          amount
        );
        // transfer tokens
        const receipt = await transferFromEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          TEST_ERC20,
          amount,
          CHAIN_ID_SOLANA,
          tryNativeToUint8Array(recipient.toString(), CHAIN_ID_SOLANA)
        );
        // get the sequence from the logs (needed to fetch the vaa)
        const sequence = parseSequenceFromLogEth(
          receipt,
          CONTRACTS.DEVNET.ethereum.core
        );
        const emitterAddress = getEmitterAddressEth(
          CONTRACTS.DEVNET.ethereum.token_bridge
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
        // post vaa to Solana
        await postVaaSolana(
          connection,
          async (transaction) => {
            transaction.partialSign(keypair);
            return transaction;
          },
          CONTRACTS.DEVNET.solana.core,
          payerAddress,
          Buffer.from(signedVAA)
        );
        expect(
          await getIsTransferCompletedSolana(
            CONTRACTS.DEVNET.solana.token_bridge,
            signedVAA,
            connection
          )
        ).toBe(false);
        // redeem tokens on solana
        const transaction = await redeemOnSolana(
          connection,
          CONTRACTS.DEVNET.solana.core,
          CONTRACTS.DEVNET.solana.token_bridge,
          payerAddress,
          signedVAA
        );
        // sign, send, and confirm transaction
        transaction.partialSign(keypair);
        const txid = await connection.sendRawTransaction(
          transaction.serialize()
        );
        await connection.confirmTransaction(txid);
        expect(
          await getIsTransferCompletedSolana(
            CONTRACTS.DEVNET.solana.token_bridge,
            signedVAA,
            connection
          )
        ).toBe(true);

        // Get the final wallet balance of ERC20 on Eth
        const finalErc20BalOnEth = await token.balanceOf(
          await signer.getAddress()
        );
        const finalErc20BalOnEthFormatted = formatUnits(
          finalErc20BalOnEth._hex,
          DECIMALS
        );
        expect(
          parseInt(initialErc20BalOnEthFormatted) -
            parseInt(finalErc20BalOnEthFormatted) ===
            1
        ).toBe(true);

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
  describe("Post VAA with retry", () => {
    test("postVAA with retry, no failures", (done) => {
      (async () => {
        try {
          // create a keypair for Solana
          const connection = new Connection(SOLANA_HOST, "confirmed");
          const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
          const payerAddress = keypair.publicKey.toString();
          const sequence = await transferFromEthToSolana();
          const emitterAddress = getEmitterAddressEth(
            CONTRACTS.DEVNET.ethereum.token_bridge
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
          let maxFailures = 0;
          // post vaa to Solana

          const postPromise = postVaaWithRetry(
            connection,
            async (transaction) => {
              await new Promise(function (resolve) {
                //We delay here so the connection has time to get wrecked
                setTimeout(function () {
                  resolve(500);
                });
              });
              transaction.partialSign(keypair);
              return transaction;
            },
            CONTRACTS.DEVNET.solana.core,
            payerAddress,
            Buffer.from(signedVAA),
            maxFailures
          );

          await postPromise;
          // redeem tokens on solana
          const transaction = await redeemOnSolana(
            connection,
            CONTRACTS.DEVNET.solana.core,
            CONTRACTS.DEVNET.solana.token_bridge,
            payerAddress,
            signedVAA
          );
          // sign, send, and confirm transaction
          transaction.partialSign(keypair);
          const txid = await connection.sendRawTransaction(
            transaction.serialize()
          );
          await connection.confirmTransaction(txid);
          expect(
            await getIsTransferCompletedSolana(
              CONTRACTS.DEVNET.solana.token_bridge,
              signedVAA,
              connection
            )
          ).toBe(true);
          done();
        } catch (e) {
          console.error(e);
          done(
            "An error occurred while happy-path testing post VAA with retry."
          );
        }
      })();
    });
    test("Reject on signature failure", (done) => {
      (async () => {
        try {
          // create a keypair for Solana
          const connection = new Connection(SOLANA_HOST, "confirmed");
          const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
          const payerAddress = keypair.publicKey.toString();
          const sequence = await transferFromEthToSolana();
          const emitterAddress = getEmitterAddressEth(
            CONTRACTS.DEVNET.ethereum.token_bridge
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
          let maxFailures = 5;
          // post vaa to Solana

          let error = false;
          try {
            const postPromise = postVaaWithRetry(
              connection,
              async (transaction) => {
                return Promise.reject();
              },
              CONTRACTS.DEVNET.solana.core,
              payerAddress,
              Buffer.from(signedVAA),
              maxFailures
            );

            await postPromise;
          } catch (e) {
            error = true;
          }
          expect(error).toBe(true);
          done();
        } catch (e) {
          console.error(e);
          done(
            "An error occurred while trying to send from Ethereum to Solana"
          );
        }
      })();
    });
  });
});
