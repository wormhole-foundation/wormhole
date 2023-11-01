import { describe, expect, jest, test } from "@jest/globals";

import { Connection } from "@solana/web3.js";
import { ethers } from "ethers";

// Borrow consts
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  TEST_ERC20,
  WORMHOLE_RPC_HOSTS,
} from "./utils/consts";

import {
  api,
  encoding,
  nativeChainAddress,
  normalizeAmount,
  signSendWait
} from "@wormhole-foundation/connect-sdk";

import { EvmAddress, EvmChain, EvmPlatform, getEvmSigner } from "@wormhole-foundation/connect-sdk-evm";
import { SolanaChain, SolanaPlatform, getSolanaSigner } from "@wormhole-foundation/connect-sdk-solana";

import { getEthTokenBridge, getSolTokenBridge } from '../../src';

jest.setTimeout(60000);

const tokenId = nativeChainAddress(["Ethereum", TEST_ERC20])

describe("Ethereum to Solana and Back", () => {
  test("Attest Ethereum ERC-20 to Solana", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.WebSocketProvider(ETH_NODE_URL);
        const ethSigner = await getEvmSigner(provider, ETH_PRIVATE_KEY);

        const ethChain = EvmPlatform.getChain("Ethereum")
        const ethTb = await getEthTokenBridge(provider)

        const attestTxs = ethTb.createAttestation(tokenId.address)
        const txids = await signSendWait(ethChain, attestTxs, ethSigner)

        const [whm] = await ethChain.parseTransaction(txids[txids.length - 1].txid)

        // poll until the guardian(s) witness and sign the vaa
        const vaa = await api.getVaaWithRetry(WORMHOLE_RPC_HOSTS[0], whm, "TokenBridge:AttestMeta")
        if (!vaa)
          throw new Error("No vaa found")

        // Submit the VAA to Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const solSigner = await getSolanaSigner(connection, encoding.b58.encode(SOLANA_PRIVATE_KEY))

        const solChain = SolanaPlatform.getChain("Solana");
        const solTb = await getSolTokenBridge(connection)

        const submitTxs = solTb.submitAttestation(vaa, solSigner.address())

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
    const address = await solTb.getWrappedAsset(nativeChainAddress(["Ethereum", TEST_ERC20]))
    expect(address).toBeTruthy();
  });
  test("Send Ethereum ERC-20 to Solana", (done) => {
    (async () => {
      try {
        const DECIMALS = 18n;

        const ethChain = EvmPlatform.getChain("Ethereum") as EvmChain
        const solChain = SolanaPlatform.getChain("Solana") as SolanaChain

        // create a signer for Eth
        const provider = new ethers.WebSocketProvider(ETH_NODE_URL);
        const ethSigner = await getEvmSigner(provider, ETH_PRIVATE_KEY);

        // create a keypair for Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const solSigner = await getSolanaSigner(connection, encoding.b58.encode(SOLANA_PRIVATE_KEY))

        // Get the token bridge clients
        const ethTb = await getEthTokenBridge(provider);
        const solTb = await getSolTokenBridge(connection)

        const amount = normalizeAmount("1", DECIMALS);

        // determine destination address - an associated token account
        const solanaForeignAsset = await solTb.getWrappedAsset(tokenId);
        const recipient = await solChain.getTokenAccount(solanaForeignAsset, solSigner.address())

        const solDecimals = await solChain.getDecimals(solanaForeignAsset);
        const solAmount = normalizeAmount("1", solDecimals);

        // Get the initial wallet balances
        const initialErc20BalOnEth = await ethChain.getBalance(ethSigner.address(), tokenId.address as EvmAddress) ?? 0n;
        const initialSolanaBalance = await solChain.getBalance(solSigner.address(), solanaForeignAsset) ?? 0n;

        // Send the transfer
        const xfer = ethTb.transfer(ethSigner.address(), { chain: "Solana", address: recipient }, TEST_ERC20, amount)
        const txids = await signSendWait(ethChain, xfer, ethSigner)

        // Get the VAA from the wormhole message emitted in events
        const [whm] = await ethChain.parseTransaction(txids[txids.length - 1].txid)
        const vaa = await api.getVaaWithRetry(WORMHOLE_RPC_HOSTS[0], whm, "TokenBridge:Transfer")
        if (!vaa)
          throw new Error("No vaa found")

        expect(await solTb.isTransferCompleted(vaa)).toBe(false);

        // redeem tokens on solana
        const redeemTxs = solTb.redeem(solSigner.address(), vaa)
        await signSendWait(solChain, redeemTxs, solSigner)

        expect(await solTb.isTransferCompleted(vaa)).toBe(true);

        // Get the final wallet balance of ERC20 on Eth
        const finalErc20BalOnEth = await ethChain.getBalance(ethSigner.address(), tokenId.address as EvmAddress) ?? 0n;
        expect(initialErc20BalOnEth - finalErc20BalOnEth).toEqual(amount);

        // Get final balance on Solana
        const finalSolanaBalance = await solChain.getBalance(solSigner.address(), solanaForeignAsset) ?? 0n;
        expect(finalSolanaBalance - initialSolanaBalance).toEqual(solAmount);

        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to send from Ethereum to Solana");
      }
    })();
  });
  test("Send Wrapped ERC-20 from Solana back to Ethereum", (done) => {
    (async () => {
      try {
        const ethChain = EvmPlatform.getChain("Ethereum") as EvmChain
        const solChain = SolanaPlatform.getChain("Solana") as SolanaChain

        // create a signer for Eth
        const provider = new ethers.WebSocketProvider(ETH_NODE_URL);
        const ethSigner = await getEvmSigner(provider, ETH_PRIVATE_KEY);

        // create a keypair for Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const solSigner = await getSolanaSigner(connection, encoding.b58.encode(SOLANA_PRIVATE_KEY))

        // Get the token bridge clients
        const ethTb = await getEthTokenBridge(provider);
        const solTb = await getSolTokenBridge(connection)

        const solanaForeignAsset = await solTb.getWrappedAsset(tokenId);
        const solDecimals = await solChain.getDecimals(solanaForeignAsset);
        const amount = normalizeAmount("1", solDecimals);

        const ethAmount = normalizeAmount("1", 18n);

        // Get the initial wallet balances
        const initialErc20BalOnEth = await ethChain.getBalance(ethSigner.address(), tokenId.address as EvmAddress) ?? 0n;
        const initialSolanaBalance = await solChain.getBalance(solSigner.address(), solanaForeignAsset) ?? 0n;

        // Send the transfer
        const xfer = solTb.transfer(solSigner.address(), nativeChainAddress(["Ethereum", ethSigner.address()]), solanaForeignAsset, amount)
        const txids = await signSendWait(solChain, xfer, solSigner)

        // Get the VAA from the wormhole message emitted in events
        const [whm] = await solChain.parseTransaction(txids[txids.length - 1].txid)
        const vaa = await api.getVaaWithRetry(WORMHOLE_RPC_HOSTS[0], whm, "TokenBridge:Transfer")
        if (!vaa)
          throw new Error("No vaa found")

        expect(await ethTb.isTransferCompleted(vaa)).toBe(false);

        // redeem tokens on solana
        const redeemTxs = ethTb.redeem(ethSigner.address(), vaa)
        await signSendWait(ethChain, redeemTxs, ethSigner)

        expect(await ethTb.isTransferCompleted(vaa)).toBe(true);

        // Get final balance on Solana
        const finalSolanaBalance = await solChain.getBalance(solSigner.address(), solanaForeignAsset) ?? 0n;
        expect(initialSolanaBalance - finalSolanaBalance).toEqual(amount);

        // Get the final wallet balance of ERC20 on Eth
        const finalErc20BalOnEth = await ethChain.getBalance(ethSigner.address(), tokenId.address as EvmAddress) ?? 0n;
        expect(finalErc20BalOnEth - initialErc20BalOnEth).toEqual(ethAmount);

        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to send from Ethereum to Solana");
      }
    })();
  });
});