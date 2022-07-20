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
} from "@solana/web3.js";
import { ethers } from "ethers";
import {
  attestFromSolana,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CONTRACTS,
  createWrappedOnEth,
  getEmitterAddressSolana,
  getForeignAssetEth,
  getIsTransferCompletedEth,
  hexToUint8Array,
  parseSequenceFromLogSolana,
  redeemOnEth,
  TokenImplementation__factory,
  transferFromSolana,
  tryNativeToHexString,
  tryNativeToUint8Array,
} from "../..";
import getSignedVAAWithRetry from "../../rpc/getSignedVAAWithRetry";
import { setDefaultWasm } from "../../solana/wasm";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY3,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  TEST_SOLANA_TOKEN,
  WORMHOLE_RPC_HOSTS,
} from "./consts";

setDefaultWasm("node");

jest.setTimeout(60000);

describe("Solana to Ethereum", () => {
  test("Attest Solana SPL to Ethereum", (done) => {
    (async () => {
      try {
        // create a keypair for Solana
        const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
        const payerAddress = keypair.publicKey.toString();
        // attest the test token
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const transaction = await attestFromSolana(
          connection,
          CONTRACTS.DEVNET.solana.core,
          CONTRACTS.DEVNET.solana.token_bridge,
          payerAddress,
          TEST_SOLANA_TOKEN
        );
        // sign, send, and confirm transaction
        transaction.partialSign(keypair);
        const txid = await connection.sendRawTransaction(
          transaction.serialize()
        );
        await connection.confirmTransaction(txid);
        const info = await connection.getTransaction(txid);
        if (!info) {
          throw new Error(
            "An error occurred while fetching the transaction info"
          );
        }
        // get the sequence from the logs (needed to fetch the vaa)
        const sequence = parseSequenceFromLogSolana(info);
        const emitterAddress = await getEmitterAddressSolana(
          CONTRACTS.DEVNET.solana.token_bridge
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
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY3, provider);
        try {
          await createWrappedOnEth(
            CONTRACTS.DEVNET.ethereum.token_bridge,
            signer,
            signedVAA
          );
        } catch (e) {
          // this could fail because the token is already attested (in an unclean env)
        }
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done(
          "An error occurred while trying to attest from Solana to Ethereum"
        );
      }
    })();
  });
  test("Solana SPL is attested on Ethereum", async () => {
    const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
    const address = getForeignAssetEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      provider,
      "solana",
      tryNativeToUint8Array(TEST_SOLANA_TOKEN, "solana")
    );
    expect(address).toBeTruthy();
    expect(address).not.toBe(ethers.constants.AddressZero);
    provider.destroy();
  });
  test("Send Solana SPL to Ethereum", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY3, provider);
        const targetAddress = await signer.getAddress();
        // create a keypair for Solana
        const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
        const payerAddress = keypair.publicKey.toString();
        // find the associated token account
        const fromAddress = (
          await Token.getAssociatedTokenAddress(
            ASSOCIATED_TOKEN_PROGRAM_ID,
            TOKEN_PROGRAM_ID,
            new PublicKey(TEST_SOLANA_TOKEN),
            keypair.publicKey
          )
        ).toString();

        const connection = new Connection(SOLANA_HOST, "confirmed");

        // Get the initial solana token balance
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
          if (tokenInfo.mint === TEST_SOLANA_TOKEN) {
            initialSolanaBalance = amount;
          }
        }

        // Get the initial wallet balance on Eth
        const originAssetHex = tryNativeToHexString(
          TEST_SOLANA_TOKEN,
          CHAIN_ID_SOLANA
        );
        if (!originAssetHex) {
          throw new Error("originAssetHex is null");
        }
        const foreignAsset = await getForeignAssetEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          provider,
          CHAIN_ID_SOLANA,
          hexToUint8Array(originAssetHex)
        );
        if (!foreignAsset) {
          throw new Error("foreignAsset is null");
        }
        let token = TokenImplementation__factory.connect(foreignAsset, signer);
        const initialBalOnEth = await token.balanceOf(
          await signer.getAddress()
        );
        const initialBalOnEthFormatted = formatUnits(initialBalOnEth._hex, 9);

        // transfer the test token
        const amount = parseUnits("1", 9).toBigInt();
        const transaction = await transferFromSolana(
          connection,
          CONTRACTS.DEVNET.solana.core,
          CONTRACTS.DEVNET.solana.token_bridge,
          payerAddress,
          fromAddress,
          TEST_SOLANA_TOKEN,
          amount,
          tryNativeToUint8Array(targetAddress, CHAIN_ID_ETH),
          CHAIN_ID_ETH
        );
        // sign, send, and confirm transaction
        transaction.partialSign(keypair);
        const txid = await connection.sendRawTransaction(
          transaction.serialize()
        );
        await connection.confirmTransaction(txid);
        const info = await connection.getTransaction(txid);
        if (!info) {
          throw new Error(
            "An error occurred while fetching the transaction info"
          );
        }
        // get the sequence from the logs (needed to fetch the vaa)
        const sequence = parseSequenceFromLogSolana(info);
        const emitterAddress = await getEmitterAddressSolana(
          CONTRACTS.DEVNET.solana.token_bridge
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
        expect(
          await getIsTransferCompletedEth(
            CONTRACTS.DEVNET.ethereum.token_bridge,
            provider,
            signedVAA
          )
        ).toBe(false);
        await redeemOnEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          signedVAA
        );
        expect(
          await getIsTransferCompletedEth(
            CONTRACTS.DEVNET.ethereum.token_bridge,
            provider,
            signedVAA
          )
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
          if (tokenInfo.mint === TEST_SOLANA_TOKEN) {
            finalSolanaBalance = amount;
          }
        }
        expect(initialSolanaBalance - finalSolanaBalance).toBeCloseTo(1);

        // Get the final balance on Eth
        const finalBalOnEth = await token.balanceOf(await signer.getAddress());
        const finalBalOnEthFormatted = formatUnits(finalBalOnEth._hex, 9);
        expect(
          parseInt(finalBalOnEthFormatted) -
            parseInt(initialBalOnEthFormatted) ===
            1
        ).toBe(true);
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to send from Solana to Ethereum");
      }
    })();
  });
});
