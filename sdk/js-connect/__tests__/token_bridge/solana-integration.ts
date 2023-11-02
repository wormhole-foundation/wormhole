import { describe, expect, jest, test } from "@jest/globals";
import {
  NATIVE_MINT
} from "@solana/spl-token";
import {
  Connection
} from "@solana/web3.js";
import { api, encoding, nativeChainAddress, normalizeAmount, signSendWait } from "@wormhole-foundation/connect-sdk";
import { EvmChain, EvmPlatform, getEvmSigner } from "@wormhole-foundation/connect-sdk-evm";
import { SolanaAddress, SolanaChain, SolanaPlatform, getSolanaSigner } from "@wormhole-foundation/connect-sdk-solana";
import { ethers } from "ethers";
import { getEthTokenBridge, getSolTokenBridge } from "../../src";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY3,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  TEST_SOLANA_TOKEN,
  WORMHOLE_RPC_HOSTS,
} from "./utils/consts";

jest.setTimeout(60000);
const tokenId = nativeChainAddress(["Solana", TEST_SOLANA_TOKEN])

describe("Solana to Ethereum", () => {
  test("Attest Solana SPL to Ethereum", (done) => {
    (async () => {
      try {
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const solSigner = await getSolanaSigner(connection, encoding.b58.encode(SOLANA_PRIVATE_KEY))
        const solChain = SolanaPlatform.getChain("Solana") as SolanaChain;
        const solTb = await getSolTokenBridge(connection)

        // attest the test token
        const transactions = solTb.createAttestation(TEST_SOLANA_TOKEN, solSigner.address())
        const txids = await signSendWait(solChain, transactions, solSigner)

        // get the sequence from the logs (needed to fetch the vaa)
        const [whm] = await solChain.parseTransaction(txids[txids.length - 1].txid)
        const signedVAA = await api.getVaaWithRetry(WORMHOLE_RPC_HOSTS[0], whm, "TokenBridge:AttestMeta")
        if (!signedVAA)
          throw new Error("No vaa available")

        // create a signer for Eth
        const provider = new ethers.WebSocketProvider(ETH_NODE_URL);
        const ethSigner = await getEvmSigner(provider, ETH_PRIVATE_KEY3);

        const ethChain = EvmPlatform.getChain("Ethereum") as EvmChain;
        const ethTb = await getEthTokenBridge(provider);
        try {
          await signSendWait(ethChain, ethTb.submitAttestation(signedVAA), ethSigner)
        } catch (e) {
          console.error(e)
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
    const provider = new ethers.WebSocketProvider(ETH_NODE_URL);
    const ethTb = await getEthTokenBridge(provider);
    const address = await ethTb.getWrappedAsset(tokenId)
    expect(address).toBeTruthy();
    //expect(address).not.toBe(ethers.AddressZero);
    provider.destroy();
  });
  test("Send Solana SPL to Ethereum", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.WebSocketProvider(ETH_NODE_URL);
        const ethSigner = await getEvmSigner(provider, ETH_PRIVATE_KEY3)
        const ethChain = EvmPlatform.getChain("Ethereum") as EvmChain;
        const ethTb = await getEthTokenBridge(provider);

        // create a keypair for Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const solSigner = await getSolanaSigner(connection, encoding.b58.encode(SOLANA_PRIVATE_KEY))
        const solChain = SolanaPlatform.getChain("Solana") as SolanaChain
        const solTb = await getSolTokenBridge(connection);

        // Get the initial solana token balance
        const initialSolanaBalance = await solChain.getBalance(solSigner.address(), tokenId.address as SolanaAddress) ?? 0n;

        // Get the initial wallet balance on Eth
        const ethTokenAddress = await ethTb.getWrappedAsset(tokenId)
        const initialBalOnEth = await ethChain.getBalance(ethSigner.address(), ethTokenAddress) ?? 0n;

        // transfer the test token
        const amount = normalizeAmount("1", 9n)
        const transactions = solTb.transfer(solSigner.address(), nativeChainAddress(ethSigner), TEST_SOLANA_TOKEN, amount)
        const txids = await signSendWait(solChain, transactions, solSigner)

        // get the sequence from the logs (needed to fetch the vaa)
        const [whm] = await solChain.parseTransaction(txids[txids.length - 1].txid)

        // poll until the guardian(s) witness and sign the vaa
        const signedVAA = await api.getVaaWithRetry(WORMHOLE_RPC_HOSTS[0], whm, "TokenBridge:Transfer")
        if (!signedVAA)
          throw new Error("No vaa available")

        expect(await ethTb.isTransferCompleted(signedVAA)).toBe(false);
        await signSendWait(ethChain, ethTb.redeem(ethSigner.address(), signedVAA), ethSigner)
        expect(await ethTb.isTransferCompleted(signedVAA)).toBe(true);

        // Get final balance on Solana
        const finalSolanaBalance = await solChain.getBalance(solSigner.address(), tokenId.address as SolanaAddress) ?? 0n;
        expect(initialSolanaBalance - finalSolanaBalance).toEqual(amount);

        // Get the final balance on Eth
        const finalBalOnEth = await ethChain.getBalance(ethSigner.address(), ethTokenAddress) ?? 0n;
        expect(finalBalOnEth - initialBalOnEth).toEqual(amount);
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to send from Solana to Ethereum");
      }
    })();
  });
  test("Attest Native SOL to Ethereum", (done) => {
    (async () => {
      try {
        // create a keypair for Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const solSigner = await getSolanaSigner(connection, encoding.b58.encode(SOLANA_PRIVATE_KEY));
        const solChain = SolanaPlatform.getChain("Solana") as SolanaChain;
        const solTb = await getSolTokenBridge(connection);
        // attest the test token
        const transactions = solTb.createAttestation(NATIVE_MINT.toString(), solSigner.address())
        // sign, send, and confirm transaction
        const txids = await signSendWait(solChain, transactions, solSigner)

        // get the sequence from the logs (needed to fetch the vaa)
        const [whm] = await solChain.parseTransaction(txids[txids.length - 1].txid)

        // poll until the guardian(s) witness and sign the vaa
        const signedVAA = await api.getVaaWithRetry(WORMHOLE_RPC_HOSTS[0], whm, "TokenBridge:AttestMeta")
        if (!signedVAA)
          throw new Error("No vaa available")

        // create a signer for Eth
        const provider = new ethers.WebSocketProvider(ETH_NODE_URL);
        const ethSigner = await getEvmSigner(provider, ETH_PRIVATE_KEY3);
        const ethTb = await getEthTokenBridge(provider)
        const ethChain = EvmPlatform.getChain("Ethereum") as EvmChain;
        try {
          await signSendWait(ethChain, ethTb.submitAttestation(signedVAA), ethSigner)
        } catch (e) {
          console.error(e)
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
  test("Send Native SOL to Ethereum", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.WebSocketProvider(ETH_NODE_URL);
        const ethSigner = await getEvmSigner(provider, ETH_PRIVATE_KEY3)
        const ethTb = await getEthTokenBridge(provider)
        const ethChain = EvmPlatform.getChain("Ethereum") as EvmChain;
        // create a keypair for Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const solSigner = await getSolanaSigner(connection, encoding.b58.encode(SOLANA_PRIVATE_KEY));
        const solTb = await getSolTokenBridge(connection);
        const solChain = SolanaPlatform.getChain("Solana") as SolanaChain;

        // Get the initial wallet balance on Eth
        const wrappedNative = await solTb.getWrappedNative()
        const foreignAsset = await ethTb.getWrappedAsset(nativeChainAddress(["Solana", wrappedNative.toString()]));
        const initialBalOnEth = await ethChain.getBalance(ethSigner.address(), foreignAsset) ?? 0n;

        // transfer sol
        const amount = normalizeAmount("1", 9n)
        const transactions = solTb.transfer(solSigner.address(), nativeChainAddress(ethSigner), 'native', amount)
        // sign, send, and confirm transaction
        const txids = await signSendWait(solChain, transactions, solSigner)
        // get the sequence from the logs (needed to fetch the vaa)
        const [whm] = await solChain.parseTransaction(txids[txids.length - 1].txid)
        // poll until the guardian(s) witness and sign the vaa
        const signedVAA = await api.getVaaWithRetry(WORMHOLE_RPC_HOSTS[0], whm, "TokenBridge:Transfer")
        if (!signedVAA)
          throw new Error("No vaa available")

        expect(await ethTb.isTransferCompleted(signedVAA)).toBe(false);
        await signSendWait(ethChain, ethTb.redeem(ethSigner.address(), signedVAA), ethSigner)
        expect(await ethTb.isTransferCompleted(signedVAA)).toBe(true);

        // Get the final balance on Eth
        const finalBalOnEth = await ethChain.getBalance(ethSigner.address(), foreignAsset) ?? 0n;
        expect(finalBalOnEth - initialBalOnEth).toEqual(amount);
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to send from Solana to Ethereum");
      }
    })();
  });
});
