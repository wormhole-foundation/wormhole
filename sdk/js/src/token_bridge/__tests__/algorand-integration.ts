import { parseUnits } from "@ethersproject/units";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { describe, expect, jest, test } from "@jest/globals";
import algosdk, {
  Account,
  decodeAddress,
  getApplicationAddress,
  makeApplicationCallTxnFromObject,
  OnApplicationComplete,
  waitForConfirmation,
} from "algosdk";
import { BigNumber, ethers, utils } from "ethers";
import {
  approveEth,
  attestFromAlgorand,
  attestFromEth,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_ETH,
  CONTRACTS,
  createWrappedOnAlgorand,
  createWrappedOnEth,
  getEmitterAddressAlgorand,
  getEmitterAddressEth,
  getForeignAssetEth,
  getIsTransferCompletedAlgorand,
  getIsTransferCompletedEth,
  getOriginalAssetAlgorand,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogAlgorand,
  parseSequenceFromLogEth,
  redeemOnAlgorand,
  redeemOnEth,
  textToUint8Array,
  TokenImplementation__factory,
  transferFromAlgorand,
  transferFromEth,
  uint8ArrayToHex,
  updateWrappedOnEth,
  WormholeWrappedInfo,
} from "../..";
import { _parseVAAAlgorand } from "../../algorand";
import {
  createAsset,
  getAlgoClient,
  getBalance,
  getBalances,
  getForeignAssetFromVaaAlgorand,
  getTempAccounts,
  signSendAndConfirmAlgorand,
} from "../../algorand/__tests__/testHelpers";
import getSignedVAAWithRetry from "../../rpc/getSignedVAAWithRetry";
import { setDefaultWasm } from "../../solana/wasm";
import { safeBigIntToNumber } from "../../utils/bigint";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY7,
  TEST_ERC20,
  WORMHOLE_RPC_HOSTS,
} from "./consts";

const CORE_ID = BigInt(4);
const TOKEN_BRIDGE_ID = BigInt(6);

setDefaultWasm("node");

jest.setTimeout(60000);

describe("Algorand tests", () => {
  test("Algorand transfer native ALGO to Eth and back again", (done) => {
    (async () => {
      try {
        const client: algosdk.Algodv2 = getAlgoClient();
        const tempAccts: Account[] = await getTempAccounts();
        const numAccts: number = tempAccts.length;
        expect(numAccts).toBeGreaterThan(0);
        const wallet: Account = tempAccts[0];

        // let accountInfo = await client.accountInformation(wallet.addr).do();
        // Asset Index of native ALGO is 0
        const AlgoIndex = BigInt(0);
        // const b = await getBalances(client, wallet.addr);
        const txs = await attestFromAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          wallet.addr,
          AlgoIndex
        );

        const result = await signSendAndConfirmAlgorand(client, txs, wallet);

        const sn = parseSequenceFromLogAlgorand(result);

        // Now, try to send a NOP
        const suggParams: algosdk.SuggestedParams = await client
          .getTransactionParams()
          .do();
        const nopTxn = makeApplicationCallTxnFromObject({
          from: wallet.addr,
          appIndex: safeBigIntToNumber(TOKEN_BRIDGE_ID),
          onComplete: OnApplicationComplete.NoOpOC,
          appArgs: [textToUint8Array("nop")],
          suggestedParams: suggParams,
        });
        const resp = await client
          .sendRawTransaction(nopTxn.signTxn(wallet.sk))
          .do();
        await waitForConfirmation(client, resp.txId, 1);
        // End of NOP

        const emitterAddr = getEmitterAddressAlgorand(TOKEN_BRIDGE_ID);
        const { vaaBytes } = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_ALGORAND,
          emitterAddr,
          sn,
          { transport: NodeHttpTransport() }
        );
        const pvaa = _parseVAAAlgorand(vaaBytes);
        const provider = new ethers.providers.WebSocketProvider(
          ETH_NODE_URL
        ) as any;
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY7, provider);
        let success: boolean = true;
        try {
          const cr = await createWrappedOnEth(
            CONTRACTS.DEVNET.ethereum.token_bridge,
            signer,
            vaaBytes
          );
        } catch (e) {
          success = false;
        }
        if (!success) {
          try {
            const cr = await updateWrappedOnEth(
              CONTRACTS.DEVNET.ethereum.token_bridge,
              signer,
              vaaBytes
            );
            success = true;
          } catch (e) {
            console.error("failed to updateWrappedOnEth", e);
          }
        }
        // Check wallet
        const a = parseInt(AlgoIndex.toString());
        const originAssetHex = (
          "0000000000000000000000000000000000000000000000000000000000000000" +
          a.toString(16)
        ).slice(-64);
        const foreignAsset = await getForeignAssetEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          provider,
          CHAIN_ID_ALGORAND,
          hexToUint8Array(originAssetHex)
        );
        if (!foreignAsset) {
          throw new Error("foreignAsset is null");
        }
        let token = TokenImplementation__factory.connect(foreignAsset, signer);

        // Get initial balance on ethereum
        const initialBalOnEth = await token.balanceOf(
          await signer.getAddress()
        );
        const initialBalOnEthInt = parseInt(initialBalOnEth._hex);

        // Get initial balance on Algorand
        let algoWalletBals: Map<number, number> = await getBalances(
          client,
          wallet.addr
        );
        const startingAlgoBal = algoWalletBals.get(
          safeBigIntToNumber(AlgoIndex)
        );
        if (!startingAlgoBal) {
          throw new Error("startingAlgoBal is undefined");
        }

        // Start transfer from Algorand to Ethereum
        const hexStr = nativeToHexString(
          await signer.getAddress(),
          CHAIN_ID_ETH
        );
        if (!hexStr) {
          throw new Error("Failed to convert to hexStr");
        }
        const AmountToTransfer: number = 12300;
        const Fee: number = 0;
        const transferTxs = await transferFromAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          wallet.addr,
          AlgoIndex,
          BigInt(AmountToTransfer),
          hexStr,
          CHAIN_ID_ETH,
          BigInt(Fee)
        );
        const transferResult = await signSendAndConfirmAlgorand(
          client,
          transferTxs,
          wallet
        );
        const txSid = parseSequenceFromLogAlgorand(transferResult);
        const signedVaa = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_ALGORAND,
          emitterAddr,
          txSid,
          { transport: NodeHttpTransport() }
        );
        const pv = _parseVAAAlgorand(signedVaa.vaaBytes);
        const roe = await redeemOnEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          signedVaa.vaaBytes
        );
        expect(
          await getIsTransferCompletedEth(
            CONTRACTS.DEVNET.ethereum.token_bridge,
            provider,
            signedVaa.vaaBytes
          )
        ).toBe(true);
        // Test finished.  Check wallet balances
        const balOnEthAfter = await token.balanceOf(await signer.getAddress());
        const balOnEthAfterInt = parseInt(balOnEthAfter._hex);
        expect(balOnEthAfterInt - initialBalOnEthInt).toEqual(AmountToTransfer);

        // Get final balance on Algorand
        algoWalletBals = await getBalances(client, wallet.addr);
        const finalAlgoBal = algoWalletBals.get(safeBigIntToNumber(AlgoIndex));
        if (!finalAlgoBal) {
          throw new Error("finalAlgoBal is undefined");
        }
        // expect(startingAlgoBal - finalAlgoBal).toBe(AmountToTransfer);

        // Attempt to transfer from Eth back to Algorand
        const Amount: string = "100";

        // approve the bridge to spend tokens
        await approveEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          foreignAsset,
          signer,
          Amount
        );
        const receipt = await transferFromEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          foreignAsset,
          Amount,
          CHAIN_ID_ALGORAND,
          decodeAddress(wallet.addr).publicKey
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
        algoWalletBals = await getBalances(client, wallet.addr);
        const redeemTxs = await redeemOnAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          signedVAA,
          wallet.addr
        );
        await signSendAndConfirmAlgorand(client, redeemTxs, wallet);
        const completed = await getIsTransferCompletedAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          signedVAA
        );
        expect(completed).toBe(true);

        // Get second final balance on Algorand
        algoWalletBals = await getBalances(client, wallet.addr);
        const secondFinalAlgoBal = algoWalletBals.get(
          safeBigIntToNumber(AlgoIndex)
        );
        if (!secondFinalAlgoBal) {
          throw new Error("secondFinalAlgoBal is undefined");
        }

        provider.destroy();
      } catch (e) {
        console.error("Algorand ALGO transfer error:", e);
        done("Algorand ALGO transfer error");
        return;
      }
      done();
    })();
  });
  test("Algorand create chuckNorium, transfer to Eth and back again", (done) => {
    (async () => {
      try {
        const client: algosdk.Algodv2 = getAlgoClient();
        const tempAccts: Account[] = await getTempAccounts();
        const numAccts: number = tempAccts.length;
        expect(numAccts).toBeGreaterThan(0);
        const wallet: Account = tempAccts[0];

        // let accountInfo = await client.accountInformation(wallet.addr).do();

        const assetIndex: number = await createAsset(wallet);
        const attestTxs = await attestFromAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          wallet.addr,
          BigInt(assetIndex)
        );
        const attestResult = await signSendAndConfirmAlgorand(
          client,
          attestTxs,
          wallet
        );
        const attestSn = parseSequenceFromLogAlgorand(attestResult);

        // Now, try to send a NOP
        const suggParams: algosdk.SuggestedParams = await client
          .getTransactionParams()
          .do();
        const nopTxn = makeApplicationCallTxnFromObject({
          from: wallet.addr,
          appIndex: safeBigIntToNumber(TOKEN_BRIDGE_ID),
          onComplete: OnApplicationComplete.NoOpOC,
          appArgs: [textToUint8Array("nop")],
          suggestedParams: suggParams,
        });
        const resp = await client
          .sendRawTransaction(nopTxn.signTxn(wallet.sk))
          .do();
        await waitForConfirmation(client, resp.txId, 1);
        // End of NOP

        const emitterAddr = getEmitterAddressAlgorand(TOKEN_BRIDGE_ID);
        const { vaaBytes } = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_ALGORAND,
          emitterAddr,
          attestSn,
          { transport: NodeHttpTransport() }
        );
        const provider = new ethers.providers.WebSocketProvider(
          ETH_NODE_URL
        ) as any;
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY7, provider);
        let success: boolean = true;
        try {
          const cr = await createWrappedOnEth(
            CONTRACTS.DEVNET.ethereum.token_bridge,
            signer,
            vaaBytes
          );
        } catch (e) {
          success = false;
        }
        if (!success) {
          try {
            const cr = await updateWrappedOnEth(
              CONTRACTS.DEVNET.ethereum.token_bridge,
              signer,
              vaaBytes
            );
            success = true;
          } catch (e) {
            console.error("failed to updateWrappedOnEth", e);
            done("failed to update attestation on Eth");
            return;
          }
        }
        // Check wallet
        const a = parseInt(assetIndex.toString());
        const originAssetHex = (
          "0000000000000000000000000000000000000000000000000000000000000000" +
          a.toString(16)
        ).slice(-64);
        const foreignAsset = await getForeignAssetEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          provider,
          CHAIN_ID_ALGORAND,
          hexToUint8Array(originAssetHex)
        );
        if (!foreignAsset) {
          throw new Error("foreignAsset is null");
        }
        let token = TokenImplementation__factory.connect(foreignAsset, signer);

        // Get initial balance on ethereum

        // Get initial balance on Algorand
        let algoWalletBals: Map<number, number> = await getBalances(
          client,
          wallet.addr
        );
        const startingAlgoBal = algoWalletBals.get(assetIndex);
        if (!startingAlgoBal) {
          throw new Error("startingAlgoBal is undefined");
        }

        // Start transfer from Algorand to Ethereum
        const hexStr = nativeToHexString(
          await signer.getAddress(),
          CHAIN_ID_ETH
        );
        if (!hexStr) {
          throw new Error("Failed to convert to hexStr");
        }
        const AmountToTransfer: number = 12300;
        const Fee: number = 0;
        const transferTxs = await transferFromAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          wallet.addr,
          BigInt(assetIndex),
          BigInt(AmountToTransfer),
          hexStr,
          CHAIN_ID_ETH,
          BigInt(Fee)
        );
        const transferResult = await signSendAndConfirmAlgorand(
          client,
          transferTxs,
          wallet
        );
        const txSid = parseSequenceFromLogAlgorand(transferResult);
        const signedVaa = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_ALGORAND,
          emitterAddr,
          txSid,
          { transport: NodeHttpTransport() }
        );
        await redeemOnEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          signedVaa.vaaBytes
        );
        expect(
          await getIsTransferCompletedEth(
            CONTRACTS.DEVNET.ethereum.token_bridge,
            provider,
            signedVaa.vaaBytes
          )
        ).toBe(true);
        // Test finished.  Check wallet balances
        const balOnEthAfter = await token.balanceOf(await signer.getAddress());
        const balOnEthAfterInt = parseInt(balOnEthAfter._hex);
        const FinalAmt: number = AmountToTransfer / 100;
        expect(balOnEthAfterInt).toEqual(FinalAmt);

        // Get final balance on Algorand
        algoWalletBals = await getBalances(client, wallet.addr);
        const finalAlgoBal = algoWalletBals.get(assetIndex);
        if (!finalAlgoBal) {
          throw new Error("finalAlgoBal is undefined");
        }
        expect(startingAlgoBal - finalAlgoBal).toBe(AmountToTransfer);

        // Attempt to transfer from Eth back to Algorand
        const Amount: string = "100";

        // approve the bridge to spend tokens
        await approveEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          foreignAsset,
          signer,
          Amount
        );
        const receipt = await transferFromEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          foreignAsset,
          Amount,
          CHAIN_ID_ALGORAND,
          decodeAddress(wallet.addr).publicKey
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
        const redeemTxs = await redeemOnAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          signedVAA,
          wallet.addr
        );
        await signSendAndConfirmAlgorand(client, redeemTxs, wallet);
        const completed = await getIsTransferCompletedAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          signedVAA
        );
        expect(completed).toBe(true);
        const newBal = await token.balanceOf(await signer.getAddress());
        const newBalInt = parseInt(newBal._hex);
        expect(newBalInt).toBe(FinalAmt - parseInt(Amount));

        // Get second final balance on Algorand
        algoWalletBals = await getBalances(client, wallet.addr);
        const secondFinalAlgoBal = algoWalletBals.get(assetIndex);
        if (!secondFinalAlgoBal) {
          throw new Error("secondFinalAlgoBal is undefined");
        }
        expect(secondFinalAlgoBal - finalAlgoBal).toBe(parseInt(Amount) * 100);
        provider.destroy();
      } catch (e) {
        console.error("Algorand chuckNorium transfer error:", e);
        done("Algorand chuckNorium transfer error");
        return;
      }
      done();
    })();
  });
  test("Transfer ERC-20 from Eth to Algorand and back again", (done) => {
    (async () => {
      try {
        const tbAddr: string = getApplicationAddress(TOKEN_BRIDGE_ID);
        const decTbAddr: Uint8Array = decodeAddress(tbAddr).publicKey;
        const aa: string = uint8ArrayToHex(decTbAddr);
        const client: algosdk.Algodv2 = getAlgoClient();
        const tempAccts: Account[] = await getTempAccounts();
        const numAccts: number = tempAccts.length;
        expect(numAccts).toBeGreaterThan(0);
        const algoWallet: Account = tempAccts[0];
        const Amount = "10";
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY7, provider);
        // attest the test token
        const attestReceipt = await attestFromEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          TEST_ERC20
        );
        // get the sequence from the logs (needed to fetch the vaa)
        const attestSequence = parseSequenceFromLogEth(
          attestReceipt,
          CONTRACTS.DEVNET.ethereum.core
        );
        const emitterAddress = getEmitterAddressEth(
          CONTRACTS.DEVNET.ethereum.token_bridge
        );
        // poll until the guardian(s) witness and sign the vaa
        const { vaaBytes: attestSignedVaa } = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_ETH,
          emitterAddress,
          attestSequence,
          {
            transport: NodeHttpTransport(),
          }
        );
        const createWrappedTxs = await createWrappedOnAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          algoWallet.addr,
          attestSignedVaa
        );
        await signSendAndConfirmAlgorand(client, createWrappedTxs, algoWallet);

        let assetIdCreated = await getForeignAssetFromVaaAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          attestSignedVaa
        );
        if (!assetIdCreated) {
          throw new Error("Failed to create asset");
        }

        // Start of transfer from Eth to Algorand
        let token = TokenImplementation__factory.connect(TEST_ERC20, signer);
        // Get initial balance on ethereum
        const initialBalOnEth = await token.balanceOf(
          await signer.getAddress()
        );
        const initialBalOnEthInt = parseInt(initialBalOnEth._hex);

        // Get initial balance of TEST_ERC20 on Algorand
        const originAssetHex = nativeToHexString(TEST_ERC20, CHAIN_ID_ETH);
        if (!originAssetHex) {
          throw new Error("originAssetHex is null");
        }
        // TODO:  Get wallet balance on Algorand

        // Get Balances
        const tbBals: Map<number, number> = await getBalances(
          client,
          algoWallet.addr
          // "TPFKQBOR7RJ475XW6XMOZMSMBCZH6WNGFQNT7CM7NL2UMBCMBIU5PVBGPM"
        );
        let assetIdCreatedBegBal: number = 0;
        const tempBal = tbBals.get(safeBigIntToNumber(assetIdCreated));
        if (tempBal) {
          assetIdCreatedBegBal = tempBal;
        }

        // Start transfer from Eth to Algorand
        const parsedAmount = parseUnits(Amount, 18);
        const expectedAmount = parseUnits(Amount, 8);
        await approveEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          TEST_ERC20,
          signer,
          parsedAmount
        );
        const receipt = await transferFromEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          TEST_ERC20,
          parsedAmount,
          CHAIN_ID_ALGORAND,
          decodeAddress(algoWallet.addr).publicKey // This needs to be Algorand wallet
        );
        const transferSequence = parseSequenceFromLogEth(
          receipt,
          CONTRACTS.DEVNET.ethereum.core
        );
        // poll until the guardian(s) witness and sign the vaa
        const { vaaBytes: transferSignedVaa } = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_ETH,
          emitterAddress,
          transferSequence,
          {
            transport: NodeHttpTransport(),
          }
        );
        const redeemTxs = await redeemOnAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          transferSignedVaa,
          algoWallet.addr
        );
        await signSendAndConfirmAlgorand(client, redeemTxs, algoWallet);
        expect(
          await getIsTransferCompletedAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            transferSignedVaa
          )
        ).toBe(true);

        // Test finished.  Check wallet balances
        // Get Balances
        const bals: Map<number, number> = await getBalances(
          client,
          algoWallet.addr
        );
        let assetIdCreatedEndBal: number = 0;
        const tmpBal = bals.get(safeBigIntToNumber(assetIdCreated));
        if (tmpBal) {
          assetIdCreatedEndBal = tmpBal;
        }
        expect(assetIdCreatedEndBal - assetIdCreatedBegBal).toBe(
          expectedAmount.toNumber()
        );

        // Get intermediate balance of test token on Eth
        const midBalOnEth = await token.balanceOf(await signer.getAddress());
        const midBalOnEthInt = parseInt(midBalOnEth._hex);

        expect(
          BigInt(initialBalOnEthInt) - parsedAmount.toBigInt() ===
            BigInt(midBalOnEthInt)
        ).toBe(true);

        // Start of transfer back to Eth
        const TransferBackAmount: number = parseUnits("1", 8).toNumber();

        // transfer wrapped luna from Algorand to Eth
        const hexStr = nativeToHexString(
          await signer.getAddress(),
          CHAIN_ID_ETH
        );
        if (!hexStr) {
          throw new Error("Failed to convert to hexStr");
        }
        const Fee: number = 0;
        const transferTxs = await transferFromAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          algoWallet.addr,
          assetIdCreated,
          BigInt(TransferBackAmount),
          hexStr,
          CHAIN_ID_ETH,
          BigInt(Fee)
        );
        const transferResult = await signSendAndConfirmAlgorand(
          client,
          transferTxs,
          algoWallet
        );
        const txSid = parseSequenceFromLogAlgorand(transferResult);
        const signedVaa = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_ALGORAND,
          aa,
          txSid,
          { transport: NodeHttpTransport() }
        );

        const roe = await redeemOnEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          signedVaa.vaaBytes
        );
        expect(
          await getIsTransferCompletedEth(
            CONTRACTS.DEVNET.ethereum.token_bridge,
            provider,
            signedVaa.vaaBytes
          )
        ).toBe(true);

        // Check wallet balances after
        const finalBalOnEth = await token.balanceOf(await signer.getAddress());
        const finalBalOnEthInt = parseInt(finalBalOnEth._hex);
        expect(BigInt(finalBalOnEthInt - midBalOnEthInt)).toBe(
          parseUnits("1", 18).toBigInt()
        );
        const retBals: Map<number, number> = await getBalances(
          client,
          algoWallet.addr
        );
        let assetIdCreatedFinBal: number = 0;
        const tBal = retBals.get(safeBigIntToNumber(assetIdCreated));
        if (tBal) {
          assetIdCreatedFinBal = tBal;
        }
        expect(assetIdCreatedEndBal - assetIdCreatedFinBal).toBe(
          TransferBackAmount
        );
        const info: WormholeWrappedInfo = await getOriginalAssetAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          assetIdCreated
        );
        expect(info.chainId).toBe(CHAIN_ID_ETH);
        expect(info.isWrapped).toBe(true);
        provider.destroy();
      } catch (e) {
        console.error("Eth <=> Algorand error:", e);
        done("Eth <=> Algorand error");
        return;
      }
      done();
    })();
  });
  test("Testing relay type redeem", (done) => {
    (async () => {
      try {
        const client: algosdk.Algodv2 = getAlgoClient();
        const tempAccts: Account[] = await getTempAccounts();
        const numAccts: number = tempAccts.length;
        expect(numAccts).toBeGreaterThan(0);
        const algoWallet: Account = tempAccts[0];
        const algoWalletBalance = await getBalance(
          client,
          algoWallet.addr,
          BigInt(0)
        );
        expect(algoWalletBalance).toBeGreaterThan(0);
        const relayerWallet: Account = tempAccts[1];
        const relayerWalletBalance = await getBalance(
          client,
          relayerWallet.addr,
          BigInt(0)
        );
        expect(relayerWalletBalance).toBeGreaterThan(0);
        // ETH setup to transfer LUNA to Algorand

        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY7, provider);
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
        const createWrappedTxs = await createWrappedOnAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          algoWallet.addr,
          signedVAA
        );
        await signSendAndConfirmAlgorand(client, createWrappedTxs, algoWallet);

        let assetIdCreated = await getForeignAssetFromVaaAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          signedVAA
        );
        if (!assetIdCreated) {
          throw new Error("Failed to create asset");
        }

        // Start of transfer from ETH to Algorand
        // approve the bridge to spend tokens
        const amount = parseUnits("2", 18);
        const halfAmount = parseUnits("1", 18);
        await approveEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          TEST_ERC20,
          signer,
          amount
        );
        // transfer half the tokens directly
        const firstHalfReceipt = await transferFromEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          TEST_ERC20,
          halfAmount,
          CHAIN_ID_ALGORAND,
          decodeAddress(algoWallet.addr).publicKey // This needs to be Algorand wallet
        );
        // get the sequence from the logs (needed to fetch the vaa)
        const firstHalfSn = parseSequenceFromLogEth(
          firstHalfReceipt,
          CONTRACTS.DEVNET.ethereum.core
        );
        const ethEmitterAddress = getEmitterAddressEth(
          CONTRACTS.DEVNET.ethereum.token_bridge
        );
        // poll until the guardian(s) witness and sign the vaa
        const { vaaBytes: firstHalfVaa } = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_ETH,
          ethEmitterAddress,
          firstHalfSn,
          {
            transport: NodeHttpTransport(),
          }
        );

        // Redeem half the amount on Algorand
        const firstHalfRedeemTxs = await redeemOnAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          firstHalfVaa,
          algoWallet.addr
        );
        await signSendAndConfirmAlgorand(
          client,
          firstHalfRedeemTxs,
          algoWallet
        );
        expect(
          await getIsTransferCompletedAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            firstHalfVaa
          )
        ).toBe(true);
        // transfer second half of tokens via relayer
        const secondHalfReceipt = await transferFromEth(
          CONTRACTS.DEVNET.ethereum.token_bridge,
          signer,
          TEST_ERC20,
          halfAmount,
          CHAIN_ID_ALGORAND,
          decodeAddress(algoWallet.addr).publicKey // This needs to be Algorand wallet
        );
        // get the sequence from the logs (needed to fetch the vaa)
        const secondHalfSn = parseSequenceFromLogEth(
          secondHalfReceipt,
          CONTRACTS.DEVNET.ethereum.core
        );
        // poll until the guardian(s) witness and sign the vaa
        const { vaaBytes: secondHalfVaa } = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_ETH,
          ethEmitterAddress,
          secondHalfSn,
          {
            transport: NodeHttpTransport(),
          }
        );

        // Redeem second half the amount on Algorand
        const redeemTxs = await redeemOnAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          secondHalfVaa,
          relayerWallet.addr
        );
        await signSendAndConfirmAlgorand(client, redeemTxs, relayerWallet);
        expect(
          await getIsTransferCompletedAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            secondHalfVaa
          )
        ).toBe(true);
        provider.destroy();
      } catch (e) {
        console.error("new test error:", e);
        done("new test error");
        return;
      }
      done();
    })();
  });

  test("testing algorand payload3", (done) => {
    (async () => {
      try {
        const tbAddr: string = getApplicationAddress(TOKEN_BRIDGE_ID);
        const decTbAddr: Uint8Array = decodeAddress(tbAddr).publicKey;
        const aa: string = uint8ArrayToHex(decTbAddr);

        const client: algosdk.Algodv2 = getAlgoClient();
        const tempAccts: Account[] = await getTempAccounts();
        const numAccts: number = tempAccts.length;
        expect(numAccts).toBeGreaterThan(0);
        const algoWallet: Account = tempAccts[0];

        const Fee: number = 0;
        var testapp: number = 8;
        var dest = utils
          .hexZeroPad(BigNumber.from(testapp).toHexString(), 32)
          .substring(2);

        const transferTxs = await transferFromAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          algoWallet.addr,
          BigInt(0),
          BigInt(100),
          dest,
          CHAIN_ID_ALGORAND,
          BigInt(Fee),
          hexToUint8Array("ff")
        );

        const transferResult = await signSendAndConfirmAlgorand(
          client,
          transferTxs,
          algoWallet
        );
        const txSid = parseSequenceFromLogAlgorand(transferResult);
        const signedVaa = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_ALGORAND,
          aa,
          txSid,
          { transport: NodeHttpTransport() }
        );

        const txns = await redeemOnAlgorand(
          client,
          TOKEN_BRIDGE_ID,
          CORE_ID,
          signedVaa.vaaBytes,
          algoWallet.addr
        );

        const wbefore = await getBalance(
          client,
          getApplicationAddress(testapp),
          BigInt(0)
        );

        await signSendAndConfirmAlgorand(client, txns, algoWallet);
        expect(
          await getIsTransferCompletedAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            signedVaa.vaaBytes
          )
        ).toBe(true);
        const wafter = await getBalance(
          client,
          getApplicationAddress(testapp),
          BigInt(0)
        );

        expect(BigInt(wafter - wbefore) === BigInt(100));
      } catch (e) {
        console.error("new test error:", e);
        done("new test error");
        return;
      }
      done();
    })();
  });
});
