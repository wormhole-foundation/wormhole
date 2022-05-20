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
import {
  LCDClient,
  MnemonicKey,
  MsgExecuteContract,
} from "@terra-money/terra.js";
import algosdk, {
  Account,
  decodeAddress,
  getApplicationAddress,
  makeApplicationCallTxnFromObject,
  OnApplicationComplete,
  waitForConfirmation,
} from "algosdk";
import axios from "axios";
import { BigNumber, ethers, utils } from "ethers";
import {
  approveEth,
  attestFromAlgorand,
  attestFromEth,
  attestFromSolana,
  attestFromTerra,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  createWrappedOnAlgorand,
  createWrappedOnEth,
  createWrappedOnSolana,
  createWrappedOnTerra,
  getEmitterAddressAlgorand,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
  getForeignAssetEth,
  getForeignAssetSolana,
  getForeignAssetTerra,
  getIsTransferCompletedAlgorand,
  getIsTransferCompletedEth,
  getIsTransferCompletedSolana,
  getIsTransferCompletedTerra,
  getOriginalAssetAlgorand,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogAlgorand,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  parseSequenceFromLogTerra,
  postVaaSolana,
  redeemOnAlgorand,
  redeemOnEth,
  redeemOnSolana,
  redeemOnTerra,
  textToUint8Array,
  TokenImplementation__factory,
  transferFromAlgorand,
  transferFromEth,
  transferFromSolana,
  transferFromTerra,
  tryNativeToHexString,
  tryNativeToUint8Array,
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
import { postVaaWithRetry } from "../../solana/postVaa";
import { setDefaultWasm } from "../../solana/wasm";
import { safeBigIntToNumber } from "../../utils/bigint";
import {
  ETH_CORE_BRIDGE_ADDRESS,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_CORE_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  SOLANA_TOKEN_BRIDGE_ADDRESS,
  TERRA_CHAIN_ID,
  TERRA_GAS_PRICES_URL,
  TERRA_NODE_URL,
  TERRA_PRIVATE_KEY,
  TERRA_TOKEN_BRIDGE_ADDRESS,
  TEST_ERC20,
  TEST_SOLANA_TOKEN,
  WORMHOLE_RPC_HOSTS,
} from "./consts";
import {
  getSignedVAABySequence,
  queryBalanceOnTerra,
  transferFromEthToSolana,
  waitForTerraExecution,
} from "./helpers";

const CORE_ID = BigInt(4);
const TOKEN_BRIDGE_ID = BigInt(6);

setDefaultWasm("node");

jest.setTimeout(60000);

// TODO: setup keypair and provider/signer before, destroy provider after
// TODO: make the repeatable (can't attest an already attested token)

describe("Integration Tests", () => {
  describe("Ethereum to Solana", () => {
    test("Attest Ethereum ERC-20 to Solana", (done) => {
      (async () => {
        try {
          // create a signer for Eth
          const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          // attest the test token
          const receipt = await attestFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            TEST_ERC20
          );
          // get the sequence from the logs (needed to fetch the vaa)
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
            SOLANA_CORE_BRIDGE_ADDRESS,
            payerAddress,
            Buffer.from(signedVAA)
          );
          // create wormhole wrapped token (mint and metadata) on solana
          const transaction = await createWrappedOnSolana(
            connection,
            SOLANA_CORE_BRIDGE_ADDRESS,
            SOLANA_TOKEN_BRIDGE_ADDRESS,
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
    // TODO: it is attested
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
            SOLANA_TOKEN_BRIDGE_ADDRESS,
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
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          const initialErc20BalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
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
            ETH_TOKEN_BRIDGE_ADDRESS,
            TEST_ERC20,
            signer,
            amount
          );
          // transfer tokens
          const receipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            TEST_ERC20,
            amount,
            CHAIN_ID_SOLANA,
            tryNativeToUint8Array(recipient.toString(), CHAIN_ID_SOLANA)
          );
          // get the sequence from the logs (needed to fetch the vaa)
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
            SOLANA_CORE_BRIDGE_ADDRESS,
            payerAddress,
            Buffer.from(signedVAA)
          );
          expect(
            await getIsTransferCompletedSolana(
              SOLANA_TOKEN_BRIDGE_ADDRESS,
              signedVAA,
              connection
            )
          ).toBe(false);
          // redeem tokens on solana
          const transaction = await redeemOnSolana(
            connection,
            SOLANA_CORE_BRIDGE_ADDRESS,
            SOLANA_TOKEN_BRIDGE_ADDRESS,
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
              SOLANA_TOKEN_BRIDGE_ADDRESS,
              signedVAA,
              connection
            )
          ).toBe(true);

          // Get the final wallet balance of ERC20 on Eth
          const finalErc20BalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
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
          done(
            "An error occurred while trying to send from Ethereum to Solana"
          );
        }
      })();
    });
  });
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
            SOLANA_CORE_BRIDGE_ADDRESS,
            SOLANA_TOKEN_BRIDGE_ADDRESS,
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
            SOLANA_TOKEN_BRIDGE_ADDRESS
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
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          try {
            await createWrappedOnEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
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
    // TODO: it is attested
    test("Send Solana SPL to Ethereum", (done) => {
      (async () => {
        try {
          // create a signer for Eth
          const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
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
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          const originAssetHex = tryNativeToHexString(
            TEST_SOLANA_TOKEN,
            CHAIN_ID_SOLANA
          );
          if (!originAssetHex) {
            throw new Error("originAssetHex is null");
          }
          const foreignAsset = await getForeignAssetEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            provider,
            CHAIN_ID_SOLANA,
            hexToUint8Array(originAssetHex)
          );
          if (!foreignAsset) {
            throw new Error("foreignAsset is null");
          }
          let token = TokenImplementation__factory.connect(
            foreignAsset,
            signer
          );
          const initialBalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const initialBalOnEthFormatted = formatUnits(initialBalOnEth._hex, 9);

          // transfer the test token
          const amount = parseUnits("1", 9).toBigInt();
          const transaction = await transferFromSolana(
            connection,
            SOLANA_CORE_BRIDGE_ADDRESS,
            SOLANA_TOKEN_BRIDGE_ADDRESS,
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
            SOLANA_TOKEN_BRIDGE_ADDRESS
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
              ETH_TOKEN_BRIDGE_ADDRESS,
              provider,
              signedVAA
            )
          ).toBe(false);
          await redeemOnEth(ETH_TOKEN_BRIDGE_ADDRESS, signer, signedVAA);
          expect(
            await getIsTransferCompletedEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
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
          const finalBalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
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
          done(
            "An error occurred while trying to send from Solana to Ethereum"
          );
        }
      })();
    });
  });
  describe("Ethereum to Terra", () => {
    test("Attest Ethereum ERC-20 to Terra", (done) => {
      (async () => {
        try {
          // create a signer for Eth
          const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          // attest the test token
          const receipt = await attestFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            TEST_ERC20
          );
          // get the sequence from the logs (needed to fetch the vaa)
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
          });
          const mk = new MnemonicKey({
            mnemonic: TERRA_PRIVATE_KEY,
          });
          const wallet = lcd.wallet(mk);
          const msg = await createWrappedOnTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            wallet.key.accAddress,
            signedVAA
          );
          const gasPrices = await axios
            .get(TERRA_GAS_PRICES_URL)
            .then((result) => result.data);
          const feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await wallet.sequence(),
                publicKey: wallet.key.publicKey,
              },
            ],
            {
              msgs: [msg],
              feeDenoms: ["uluna"],
              gasPrices,
            }
          );
          const tx = await wallet.createAndSignTx({
            msgs: [msg],
            memo: "test",
            feeDenoms: ["uluna"],
            gasPrices,
            fee: feeEstimate,
          });
          await lcd.tx.broadcast(tx);
          provider.destroy();
          done();
        } catch (e) {
          console.error(e);
          done(
            "An error occurred while trying to attest from Ethereum to Terra"
          );
        }
      })();
    });
    // TODO: it is attested
    test("Send Ethereum ERC-20 to Terra", (done) => {
      (async () => {
        try {
          // create a signer for Eth
          const provider = new ethers.providers.WebSocketProvider(
            ETH_NODE_URL
          ) as any;
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          const amount = parseUnits("1", 18);
          const ERC20 = "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A";
          const TerraWalletAddress: string =
            "terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v";
          interface Erc20Balance {
            balance: string;
          }
          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
          });

          // Get initial wallet balances
          let token = TokenImplementation__factory.connect(ERC20, signer);
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          const initialBalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          let initialBalOnEthStr = ethers.utils.formatUnits(
            initialBalOnEth,
            18
          );

          // Get initial balance of ERC20 on Terra
          const originAssetHex = tryNativeToHexString(ERC20, CHAIN_ID_ETH);
          if (!originAssetHex) {
            throw new Error("originAssetHex is null");
          }
          const foreignAsset = await getForeignAssetTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            lcd,
            CHAIN_ID_ETH,
            hexToUint8Array(originAssetHex)
          );
          if (!foreignAsset) {
            throw new Error("foreignAsset is null");
          }
          const tokenDefinition: any = await lcd.wasm.contractQuery(
            foreignAsset,
            {
              token_info: {},
            }
          );
          let cw20BalOnTerra: Erc20Balance = await lcd.wasm.contractQuery(
            foreignAsset,
            {
              balance: {
                address: TerraWalletAddress,
              },
            }
          );
          let balAmount = ethers.utils.formatUnits(
            cw20BalOnTerra.balance,
            tokenDefinition.decimals
          );
          // let initialCW20BalOnTerra: number = parseInt(balAmount);

          // approve the bridge to spend tokens
          await approveEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            TEST_ERC20,
            signer,
            amount
          );
          const mk = new MnemonicKey({
            mnemonic: TERRA_PRIVATE_KEY,
          });
          const wallet = lcd.wallet(mk);
          // transfer tokens
          const receipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            TEST_ERC20,
            amount,
            CHAIN_ID_TERRA,
            tryNativeToUint8Array(wallet.key.accAddress, CHAIN_ID_TERRA)
          );
          // get the sequence from the logs (needed to fetch the vaa)
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
          expect(
            await getIsTransferCompletedTerra(
              TERRA_TOKEN_BRIDGE_ADDRESS,
              signedVAA,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(false);
          const msg = await redeemOnTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            wallet.key.accAddress,
            signedVAA
          );
          const gasPrices = await axios
            .get(TERRA_GAS_PRICES_URL)
            .then((result) => result.data);
          const feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await wallet.sequence(),
                publicKey: wallet.key.publicKey,
              },
            ],
            {
              msgs: [msg],
              memo: "localhost",
              feeDenoms: ["uluna"],
              gasPrices,
            }
          );
          const tx = await wallet.createAndSignTx({
            msgs: [msg],
            memo: "localhost",
            feeDenoms: ["uluna"],
            gasPrices,
            fee: feeEstimate,
          });
          await lcd.tx.broadcast(tx);
          expect(
            await getIsTransferCompletedTerra(
              TERRA_TOKEN_BRIDGE_ADDRESS,
              signedVAA,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(true);

          // Get wallet balance on Eth
          const finalBalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          let finalBalOnEthStr = ethers.utils.formatUnits(finalBalOnEth, 18);
          expect(
            parseInt(initialBalOnEthStr) - parseInt(finalBalOnEthStr)
          ).toEqual(1);

          // Get wallet balance on Tera
          cw20BalOnTerra = await lcd.wasm.contractQuery(foreignAsset, {
            balance: {
              address: TerraWalletAddress,
            },
          });
          balAmount = ethers.utils.formatUnits(
            cw20BalOnTerra.balance,
            tokenDefinition.decimals
          );
          // let finalCW20BalOnTerra: number = parseInt(balAmount);
          provider.destroy();
          done();
        } catch (e) {
          console.error(e);
          done("An error occurred while trying to send from Ethereum to Terra");
        }
      })();
    });
  });
  describe("Terra deposit and transfer tokens", () => {
    test("Tokens transferred can't exceed tokens deposited", (done) => {
      (async () => {
        try {
          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
          });
          const mk = new MnemonicKey({
            mnemonic: TERRA_PRIVATE_KEY,
          });
          const wallet = lcd.wallet(mk);
          const gasPrices = await axios
            .get(TERRA_GAS_PRICES_URL)
            .then((result) => result.data);
          // deposit some tokens (separate transactions)
          for (let i = 0; i < 3; i++) {
            const deposit = new MsgExecuteContract(
              wallet.key.accAddress,
              TERRA_TOKEN_BRIDGE_ADDRESS,
              {
                deposit_tokens: {},
              },
              { uusd: "900000087654321" }
            );
            const feeEstimate = await lcd.tx.estimateFee(
              [
                {
                  sequenceNumber: await wallet.sequence(),
                  publicKey: wallet.key.publicKey,
                },
              ],
              {
                msgs: [deposit],
                memo: "localhost",
                feeDenoms: ["uluna"],
                gasPrices,
              }
            );
            const tx = await wallet.createAndSignTx({
              msgs: [deposit],
              memo: "localhost",
              feeDenoms: ["uluna"],
              gasPrices,
              fee: feeEstimate,
            });
            await lcd.tx.broadcast(tx);
          }
          const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          // attempt to transfer more than we've deposited
          const transfer = new MsgExecuteContract(
            wallet.key.accAddress,
            TERRA_TOKEN_BRIDGE_ADDRESS,
            {
              initiate_transfer: {
                asset: {
                  amount: "5900000087654321",
                  info: {
                    native_token: {
                      denom: "uusd",
                    },
                  },
                },
                recipient_chain: CHAIN_ID_ETH,
                recipient: Buffer.from(signer.publicKey).toString("base64"),
                fee: "0",
                nonce: Math.round(Math.round(Math.random() * 100000)),
              },
            },
            {}
          );
          let error = false;
          try {
            await lcd.tx.estimateFee(
              [
                {
                  sequenceNumber: await wallet.sequence(),
                  publicKey: wallet.key.publicKey,
                },
              ],
              {
                msgs: [transfer],
                memo: "localhost",
                feeDenoms: ["uluna"],
                gasPrices,
              }
            );
          } catch (e) {
            error = e.response.data.message.includes("Overflow: Cannot Sub");
          }
          expect(error).toEqual(true);
          // withdraw the tokens we deposited
          const withdraw = new MsgExecuteContract(
            wallet.key.accAddress,
            TERRA_TOKEN_BRIDGE_ADDRESS,
            {
              withdraw_tokens: {
                asset: {
                  native_token: {
                    denom: "uusd",
                  },
                },
              },
            },
            {}
          );
          const feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await wallet.sequence(),
                publicKey: wallet.key.publicKey,
              },
            ],
            {
              msgs: [withdraw],
              memo: "localhost",
              feeDenoms: ["uluna"],
              gasPrices,
            }
          );
          const tx = await wallet.createAndSignTx({
            msgs: [withdraw],
            memo: "test",
            feeDenoms: ["uluna"],
            gasPrices,
            fee: feeEstimate,
          });
          await lcd.tx.broadcast(tx);
          provider.destroy();
          done();
        } catch (e) {
          console.error(e);
          done(
            "An error occurred while testing deposits to and transfers from Terra"
          );
        }
      })();
    });
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
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
            SOLANA_CORE_BRIDGE_ADDRESS,
            payerAddress,
            Buffer.from(signedVAA),
            maxFailures
          );

          await postPromise;
          // redeem tokens on solana
          const transaction = await redeemOnSolana(
            connection,
            SOLANA_CORE_BRIDGE_ADDRESS,
            SOLANA_TOKEN_BRIDGE_ADDRESS,
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
              SOLANA_TOKEN_BRIDGE_ADDRESS,
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
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
              SOLANA_CORE_BRIDGE_ADDRESS,
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
  describe("Terra to Ethereum", () => {
    test("Attestation from Terra to ETH", (done) => {
      (async () => {
        try {
          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
          });
          const mk = new MnemonicKey({
            mnemonic: TERRA_PRIVATE_KEY,
          });
          const wallet = lcd.wallet(mk);
          const Asset: string = "uluna";
          const TerraWalletAddress: string =
            "terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v";
          const msg = await attestFromTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            TerraWalletAddress,
            Asset
          );
          const gasPrices = await lcd.config.gasPrices;
          const feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await wallet.sequence(),
                publicKey: wallet.key.publicKey,
              },
            ],
            {
              msgs: [msg],
              memo: "localhost",
              feeDenoms: ["uusd"],
              gasPrices,
            }
          );
          const executeTx = await wallet.createAndSignTx({
            msgs: [msg],
            memo: "Testing...",
            feeDenoms: ["uusd"],
            gasPrices,
            fee: feeEstimate,
          });
          const result = await lcd.tx.broadcast(executeTx);
          const info = await waitForTerraExecution(result.txhash);
          if (!info) {
            throw new Error("info not found");
          }
          const sequence = parseSequenceFromLogTerra(info);
          if (!sequence) {
            throw new Error("Sequence not found");
          }
          const emitterAddress = await getEmitterAddressTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS
          );
          const signedVaa = await getSignedVAABySequence(
            CHAIN_ID_TERRA,
            sequence,
            emitterAddress
          );
          const provider = new ethers.providers.WebSocketProvider(
            ETH_NODE_URL
          ) as any;
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          let success: boolean = true;
          try {
            const cr = await createWrappedOnEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              signer,
              signedVaa
            );
          } catch (e) {
            success = false;
          }
          if (!success) {
            const cr = await updateWrappedOnEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              signer,
              signedVaa
            );
            success = true;
          }
        } catch (e) {
          console.error("Attestation failure: ", e);
        }
        done();
      })();
    });
    test("Transfer from Terra", (done) => {
      (async () => {
        try {
          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
          });
          const mk = new MnemonicKey({
            mnemonic: TERRA_PRIVATE_KEY,
          });
          const Asset: string = "uluna";
          const FeeAsset: string = "uusd";
          const Amount: string = "1000000";

          // Get initial balance of luna on Terra
          const initialTerraBalance: number = await queryBalanceOnTerra(Asset);

          // Get initial balance of uusd on Terra
          // const initialFeeBalance: number = await queryBalanceOnTerra(FeeAsset);

          // Get initial balance of wrapped luna on Eth
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          const provider = new ethers.providers.WebSocketProvider(
            ETH_NODE_URL
          ) as any;
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          const originAssetHex = tryNativeToHexString(Asset, CHAIN_ID_TERRA);
          if (!originAssetHex) {
            throw new Error("originAssetHex is null");
          }
          const foreignAsset = await getForeignAssetEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            provider,
            CHAIN_ID_TERRA,
            hexToUint8Array(originAssetHex)
          );
          if (!foreignAsset) {
            throw new Error("foreignAsset is null");
          }
          let token = TokenImplementation__factory.connect(
            foreignAsset,
            signer
          );

          // Get initial balance of wrapped luna on ethereum
          const initialLunaBalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const initialLunaBalOnEthInt = parseInt(initialLunaBalOnEth._hex);

          // Start transfer from Terra to Ethereum
          const hexStr = tryNativeToHexString(
            ETH_TEST_WALLET_PUBLIC_KEY,
            CHAIN_ID_ETH
          );
          if (!hexStr) {
            throw new Error("Failed to convert to hexStr");
          }
          const wallet = lcd.wallet(mk);
          const msgs = await transferFromTerra(
            wallet.key.accAddress,
            TERRA_TOKEN_BRIDGE_ADDRESS,
            Asset,
            Amount,
            CHAIN_ID_ETH,
            hexToUint8Array(hexStr) // This needs to be ETH wallet
          );
          const gasPrices = await lcd.config.gasPrices;
          const feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await wallet.sequence(),
                publicKey: wallet.key.publicKey,
              },
            ],
            {
              msgs: msgs,
              memo: "localhost",
              feeDenoms: [FeeAsset],
              gasPrices,
            }
          );
          const executeTx = await wallet.createAndSignTx({
            msgs: msgs,
            memo: "Testing transfer...",
            feeDenoms: [FeeAsset],
            gasPrices,
            fee: feeEstimate,
          });
          const result = await lcd.tx.broadcast(executeTx);
          const info = await waitForTerraExecution(result.txhash);
          if (!info) {
            throw new Error("info not found");
          }

          // Get VAA in order to do redemption step
          const sequence = parseSequenceFromLogTerra(info);
          if (!sequence) {
            throw new Error("Sequence not found");
          }
          const emitterAddress = await getEmitterAddressTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS
          );
          const signedVaa = await getSignedVAABySequence(
            CHAIN_ID_TERRA,
            sequence,
            emitterAddress
          );
          const roe = await redeemOnEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            signedVaa
          );
          expect(
            await getIsTransferCompletedEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              provider,
              signedVaa
            )
          ).toBe(true);

          // Test finished.  Check wallet balances
          // Get final balance of uluna on Terra
          const finalTerraBalance = await queryBalanceOnTerra(Asset);

          // Get final balance of uusd on Terra
          // const finalFeeBalance: number = await queryBalanceOnTerra(FeeAsset);
          expect(initialTerraBalance - 1e6 === finalTerraBalance).toBe(true);
          const lunaBalOnEthAfter = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const lunaBalOnEthAfterInt = parseInt(lunaBalOnEthAfter._hex);
          expect(initialLunaBalOnEthInt + 1e6 === lunaBalOnEthAfterInt).toBe(
            true
          );
        } catch (e) {
          console.error("Terra to Ethereum failure: ", e);
          done("Terra to Ethereum Failure");
          return;
        }
        done();
      })();
    });
    test("Transfer wrapped luna back to Terra", (done) => {
      (async () => {
        try {
          // Get initial wallet balances
          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
          });
          const mk = new MnemonicKey({
            mnemonic: TERRA_PRIVATE_KEY,
          });
          const Asset: string = "uluna";
          const initialTerraBalance: number = await queryBalanceOnTerra(Asset);
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          const provider = new ethers.providers.WebSocketProvider(
            ETH_NODE_URL
          ) as any;
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          const originAssetHex = tryNativeToHexString(Asset, CHAIN_ID_TERRA);
          if (!originAssetHex) {
            throw new Error("originAssetHex is null");
          }
          const foreignAsset = await getForeignAssetEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            provider,
            CHAIN_ID_TERRA,
            hexToUint8Array(originAssetHex)
          );
          if (!foreignAsset) {
            throw new Error("foreignAsset is null");
          }
          let token = TokenImplementation__factory.connect(
            foreignAsset,
            signer
          );
          const initialLunaBalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const initialLunaBalOnEthInt = parseInt(initialLunaBalOnEth._hex);
          const Amount: string = "1000000";

          // approve the bridge to spend tokens
          await approveEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            foreignAsset,
            signer,
            Amount
          );

          // transfer wrapped luna from Ethereum to Terra
          const wallet = lcd.wallet(mk);
          const receipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            foreignAsset,
            Amount,
            CHAIN_ID_TERRA,
            tryNativeToUint8Array(wallet.key.accAddress, CHAIN_ID_TERRA)
          );

          // get the sequence from the logs (needed to fetch the vaa)
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);

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
          const msg = await redeemOnTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            wallet.key.accAddress,
            signedVAA
          );
          const gasPrices = await axios
            .get(TERRA_GAS_PRICES_URL)
            .then((result) => result.data);
          const feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await wallet.sequence(),
                publicKey: wallet.key.publicKey,
              },
            ],
            {
              msgs: [msg],
              memo: "localhost",
              feeDenoms: ["uusd"],
              gasPrices,
            }
          );
          const tx = await wallet.createAndSignTx({
            msgs: [msg],
            memo: "localhost",
            feeDenoms: ["uusd"],
            gasPrices,
            fee: feeEstimate,
          });
          await lcd.tx.broadcast(tx);
          expect(
            await getIsTransferCompletedTerra(
              TERRA_TOKEN_BRIDGE_ADDRESS,
              signedVAA,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(true);

          // Check wallet balances after
          const finalTerraBalance = await queryBalanceOnTerra(Asset);
          expect(initialTerraBalance + 1e6 === finalTerraBalance).toBe(true);
          const finalLunaBalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const finalLunaBalOnEthInt = parseInt(finalLunaBalOnEth._hex);
          expect(initialLunaBalOnEthInt - 1e6 === finalLunaBalOnEthInt).toBe(
            true
          );
          // const uusdBal = await queryBalanceOnTerra("uusd");
        } catch (e) {
          console.error("Transfer back failure: ", e);
          done("Transfer back Failure");
          return;
        }
        done();
      })();
    });
  });
  describe("Terra <=> Ethereum roundtrip", () => {
    test("Transfer CW20 token from Terra to Ethereum and back again", (done) => {
      (async () => {
        try {
          const CW20: string = "terra13nkgqrfymug724h8pprpexqj9h629sa3ncw7sh";
          const Asset: string = "uluna";
          const FeeAsset: string = "uusd";
          const Amount: string = "1000000";
          const TerraWalletAddress: string =
            "terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v";

          interface Cw20Balance {
            balance: string;
          }

          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
          });
          const mk = new MnemonicKey({
            mnemonic: TERRA_PRIVATE_KEY,
          });
          const wallet = lcd.wallet(mk);

          // This is the attestation phase of the CW20 token
          let msg = await attestFromTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            TerraWalletAddress,
            CW20
          );
          let gasPrices = await lcd.config.gasPrices;
          let feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await wallet.sequence(),
                publicKey: wallet.key.publicKey,
              },
            ],
            {
              msgs: [msg],
              memo: "localhost",
              feeDenoms: [FeeAsset],
              gasPrices,
            }
          );
          let executeTx = await wallet.createAndSignTx({
            msgs: [msg],
            memo: "Testing...",
            feeDenoms: [FeeAsset],
            gasPrices,
            fee: feeEstimate,
          });
          let result = await lcd.tx.broadcast(executeTx);
          let info = await waitForTerraExecution(result.txhash);
          if (!info) {
            throw new Error("info not found");
          }
          let sequence = parseSequenceFromLogTerra(info);
          if (!sequence) {
            throw new Error("Sequence not found");
          }
          let emitterAddress = await getEmitterAddressTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS
          );
          let signedVaa = await getSignedVAABySequence(
            CHAIN_ID_TERRA,
            sequence,
            emitterAddress
          );
          const provider = new ethers.providers.WebSocketProvider(
            ETH_NODE_URL
          ) as any;
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          let success: boolean = true;
          try {
            const cr = await createWrappedOnEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              signer,
              signedVaa
            );
          } catch (e) {
            success = false;
          }
          if (!success) {
            const cr = await updateWrappedOnEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              signer,
              signedVaa
            );
            success = true;
          }
          // Attestation is complete

          // Get initial balance of uusd on Terra
          // const initialFeeBalance: number = await queryBalanceOnTerra(FeeAsset);

          // Get wallet on eth
          const originAssetHex = tryNativeToHexString(CW20, CHAIN_ID_TERRA);
          if (!originAssetHex) {
            throw new Error("originAssetHex is null");
          }
          const foreignAsset = await getForeignAssetEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            provider,
            CHAIN_ID_TERRA,
            hexToUint8Array(originAssetHex)
          );
          if (!foreignAsset) {
            throw new Error("foreignAsset is null");
          }
          let token = TokenImplementation__factory.connect(
            foreignAsset,
            signer
          );
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          const initialCW20BalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          let initialCW20BalOnEthInt = parseInt(initialCW20BalOnEth._hex);

          // Get initial balance of CW20 on Terra
          const tokenDefinition: any = await lcd.wasm.contractQuery(CW20, {
            token_info: {},
          });
          let cw20BalOnTerra: Cw20Balance = await lcd.wasm.contractQuery(CW20, {
            balance: {
              address: TerraWalletAddress,
            },
          });
          let amount = ethers.utils.formatUnits(
            cw20BalOnTerra.balance,
            tokenDefinition.decimals
          );
          let initialCW20BalOnTerra: number = parseInt(amount);
          const hexStr = tryNativeToHexString(
            ETH_TEST_WALLET_PUBLIC_KEY,
            CHAIN_ID_ETH
          );
          if (!hexStr) {
            throw new Error("Failed to convert to hexStr");
          }
          const msgs = await transferFromTerra(
            wallet.key.accAddress,
            TERRA_TOKEN_BRIDGE_ADDRESS,
            CW20,
            Amount,
            CHAIN_ID_ETH,
            hexToUint8Array(hexStr) // This needs to be ETH wallet
          );
          gasPrices = await lcd.config.gasPrices;
          feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await wallet.sequence(),
                publicKey: wallet.key.publicKey,
              },
            ],
            {
              msgs: msgs,
              memo: "localhost",
              feeDenoms: [FeeAsset],
              gasPrices,
            }
          );
          executeTx = await wallet.createAndSignTx({
            msgs: msgs,
            memo: "Testing transfer...",
            feeDenoms: [FeeAsset],
            gasPrices,
            fee: feeEstimate,
          });
          result = await lcd.tx.broadcast(executeTx);
          info = await waitForTerraExecution(result.txhash);
          if (!info) {
            throw new Error("info not found");
          }
          sequence = parseSequenceFromLogTerra(info);
          if (!sequence) {
            throw new Error("Sequence not found");
          }
          emitterAddress = await getEmitterAddressTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS
          );
          signedVaa = await getSignedVAABySequence(
            CHAIN_ID_TERRA,
            sequence,
            emitterAddress
          );
          const roe = await redeemOnEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            signedVaa
          );
          expect(
            await getIsTransferCompletedEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              provider,
              signedVaa
            )
          ).toBe(true);

          // Check the wallet balances
          let finalCW20BalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          let finalCW20BalOnEthInt = parseInt(finalCW20BalOnEth._hex);
          expect(initialCW20BalOnEthInt + 1e6 === finalCW20BalOnEthInt).toBe(
            true
          );
          cw20BalOnTerra = await lcd.wasm.contractQuery(CW20, {
            balance: {
              address: TerraWalletAddress,
            },
          });
          amount = ethers.utils.formatUnits(
            cw20BalOnTerra.balance,
            tokenDefinition.decimals
          );
          let finalCW20BalOnTerra: number = parseInt(amount);
          expect(initialCW20BalOnTerra - finalCW20BalOnTerra === 1).toBe(true);
          // Done checking wallet balances

          // Start the reverse transfer from Ethereum back to Terra
          // Get initial wallet balances
          initialCW20BalOnTerra = finalCW20BalOnTerra;
          initialCW20BalOnEthInt = finalCW20BalOnEthInt;

          // approve the bridge to spend tokens
          await approveEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            foreignAsset,
            signer,
            Amount
          );

          // transfer token from Ethereum to Terra
          const receipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            foreignAsset,
            Amount,
            CHAIN_ID_TERRA,
            tryNativeToUint8Array(wallet.key.accAddress, CHAIN_ID_TERRA)
          );

          // get the sequence from the logs (needed to fetch the vaa)
          sequence = parseSequenceFromLogEth(receipt, ETH_CORE_BRIDGE_ADDRESS);
          emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);

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
          msg = await redeemOnTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            wallet.key.accAddress,
            signedVAA
          );
          gasPrices = await axios
            .get(TERRA_GAS_PRICES_URL)
            .then((result) => result.data);
          feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await wallet.sequence(),
                publicKey: wallet.key.publicKey,
              },
            ],
            {
              msgs: [msg],
              memo: "localhost",
              feeDenoms: ["uusd"],
              gasPrices,
            }
          );
          const tx = await wallet.createAndSignTx({
            msgs: [msg],
            memo: "localhost",
            feeDenoms: ["uusd"],
            gasPrices,
            fee: feeEstimate,
          });
          await lcd.tx.broadcast(tx);
          expect(
            await getIsTransferCompletedTerra(
              TERRA_TOKEN_BRIDGE_ADDRESS,
              signedVAA,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(true);

          // Check wallet balances after transfer back
          finalCW20BalOnEth = await token.balanceOf(ETH_TEST_WALLET_PUBLIC_KEY);
          finalCW20BalOnEthInt = parseInt(finalCW20BalOnEth._hex);
          expect(initialCW20BalOnEthInt - 1e6 === finalCW20BalOnEthInt).toBe(
            true
          );
          cw20BalOnTerra = await lcd.wasm.contractQuery(CW20, {
            balance: {
              address: TerraWalletAddress,
            },
          });
          amount = ethers.utils.formatUnits(
            cw20BalOnTerra.balance,
            tokenDefinition.decimals
          );
          finalCW20BalOnTerra = parseInt(amount);
          expect(finalCW20BalOnTerra - initialCW20BalOnTerra === 1).toBe(true);
          // Done checking wallet balances
        } catch (e) {
          console.error("CW20 Transfer failure: ", e);
          done("CW20 Transfer Failure");
          return;
        }
        done();
      })();
    });
  });
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
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          let success: boolean = true;
          try {
            const cr = await createWrappedOnEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              signer,
              vaaBytes
            );
          } catch (e) {
            success = false;
          }
          if (!success) {
            try {
              const cr = await updateWrappedOnEth(
                ETH_TOKEN_BRIDGE_ADDRESS,
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
            ETH_TOKEN_BRIDGE_ADDRESS,
            provider,
            CHAIN_ID_ALGORAND,
            hexToUint8Array(originAssetHex)
          );
          if (!foreignAsset) {
            throw new Error("foreignAsset is null");
          }
          let token = TokenImplementation__factory.connect(
            foreignAsset,
            signer
          );

          // Get initial balance on ethereum
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          const initialBalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
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
            ETH_TEST_WALLET_PUBLIC_KEY,
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
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            signedVaa.vaaBytes
          );
          expect(
            await getIsTransferCompletedEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              provider,
              signedVaa.vaaBytes
            )
          ).toBe(true);
          // Test finished.  Check wallet balances
          const balOnEthAfter = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const balOnEthAfterInt = parseInt(balOnEthAfter._hex);
          expect(balOnEthAfterInt - initialBalOnEthInt).toEqual(
            AmountToTransfer
          );

          // Get final balance on Algorand
          algoWalletBals = await getBalances(client, wallet.addr);
          const finalAlgoBal = algoWalletBals.get(
            safeBigIntToNumber(AlgoIndex)
          );
          if (!finalAlgoBal) {
            throw new Error("finalAlgoBal is undefined");
          }
          // expect(startingAlgoBal - finalAlgoBal).toBe(AmountToTransfer);

          // Attempt to transfer from Eth back to Algorand
          const Amount: string = "100";

          // approve the bridge to spend tokens
          await approveEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            foreignAsset,
            signer,
            Amount
          );
          const receipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            foreignAsset,
            Amount,
            CHAIN_ID_ALGORAND,
            decodeAddress(wallet.addr).publicKey
          );
          // get the sequence from the logs (needed to fetch the vaa)
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);

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
          // const newBal = await token.balanceOf(ETH_TEST_WALLET_PUBLIC_KEY);
          // const newBalInt = parseInt(newBal._hex);
          // expect(newBalInt).toBe(AmountToTransfer - parseInt(Amount));

          // Get second final balance on Algorand
          algoWalletBals = await getBalances(client, wallet.addr);
          const secondFinalAlgoBal = algoWalletBals.get(
            safeBigIntToNumber(AlgoIndex)
          );
          if (!secondFinalAlgoBal) {
            throw new Error("secondFinalAlgoBal is undefined");
          }
          // expect(secondFinalAlgoBal - finalAlgoBal).toBe(
          //   parseInt(Amount) * 100
          // );
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
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          let success: boolean = true;
          try {
            const cr = await createWrappedOnEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              signer,
              vaaBytes
            );
          } catch (e) {
            success = false;
          }
          if (!success) {
            try {
              const cr = await updateWrappedOnEth(
                ETH_TOKEN_BRIDGE_ADDRESS,
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
            ETH_TOKEN_BRIDGE_ADDRESS,
            provider,
            CHAIN_ID_ALGORAND,
            hexToUint8Array(originAssetHex)
          );
          if (!foreignAsset) {
            throw new Error("foreignAsset is null");
          }
          let token = TokenImplementation__factory.connect(
            foreignAsset,
            signer
          );

          // Get initial balance on ethereum
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          // const initialBalOnEth = await token.balanceOf(
          //   ETH_TEST_WALLET_PUBLIC_KEY
          // );
          // const initialBalOnEthInt = parseInt(initialBalOnEth._hex);

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
            ETH_TEST_WALLET_PUBLIC_KEY,
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
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            signedVaa.vaaBytes
          );
          expect(
            await getIsTransferCompletedEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              provider,
              signedVaa.vaaBytes
            )
          ).toBe(true);
          // Test finished.  Check wallet balances
          const balOnEthAfter = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
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
            ETH_TOKEN_BRIDGE_ADDRESS,
            foreignAsset,
            signer,
            Amount
          );
          const receipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            foreignAsset,
            Amount,
            CHAIN_ID_ALGORAND,
            decodeAddress(wallet.addr).publicKey
          );
          // get the sequence from the logs (needed to fetch the vaa)
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);

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
          const newBal = await token.balanceOf(ETH_TEST_WALLET_PUBLIC_KEY);
          const newBalInt = parseInt(newBal._hex);
          expect(newBalInt).toBe(FinalAmt - parseInt(Amount));

          // Get second final balance on Algorand
          algoWalletBals = await getBalances(client, wallet.addr);
          const secondFinalAlgoBal = algoWalletBals.get(assetIndex);
          if (!secondFinalAlgoBal) {
            throw new Error("secondFinalAlgoBal is undefined");
          }
          expect(secondFinalAlgoBal - finalAlgoBal).toBe(
            parseInt(Amount) * 100
          );
          provider.destroy();
        } catch (e) {
          console.error("Algorand chuckNorium transfer error:", e);
          done("Algorand chuckNorium transfer error");
          return;
        }
        done();
      })();
    });
    test("Transfer wrapped Luna from Terra to Algorand and back again", (done) => {
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
          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
          });
          const mk = new MnemonicKey({
            mnemonic: TERRA_PRIVATE_KEY,
          });
          const terraWallet = lcd.wallet(mk);
          const Asset: string = "uluna";
          // const Asset: string = "uusd";
          const FeeAsset: string = "uusd";
          const Amount: string = "1000000";
          const TerraWalletAddress: string =
            "terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v";
          const msg = await attestFromTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            TerraWalletAddress,
            Asset
          );
          const gasPrices = lcd.config.gasPrices;
          let feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await terraWallet.sequence(),
                publicKey: terraWallet.key.publicKey,
              },
            ],
            {
              msgs: [msg],
              memo: "localhost",
              feeDenoms: [FeeAsset],
              gasPrices,
            }
          );
          const executeAttest = await terraWallet.createAndSignTx({
            msgs: [msg],
            memo: "Testing...",
            feeDenoms: [FeeAsset],
            gasPrices,
            fee: feeEstimate,
          });
          const attestResult = await lcd.tx.broadcast(executeAttest);
          const attestInfo = await waitForTerraExecution(attestResult.txhash);
          if (!attestInfo) {
            throw new Error("info not found");
          }
          const attestSn = parseSequenceFromLogTerra(attestInfo);
          if (!attestSn) {
            throw new Error("Sequence not found");
          }
          const emitterAddress = await getEmitterAddressTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS
          );
          const attestSignedVaa = await getSignedVAABySequence(
            CHAIN_ID_TERRA,
            attestSn,
            emitterAddress
          );
          const createWrappedTxs = await createWrappedOnAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            algoWallet.addr,
            attestSignedVaa
          );
          await signSendAndConfirmAlgorand(
            client,
            createWrappedTxs,
            algoWallet
          );

          let assetIdCreated = await getForeignAssetFromVaaAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            attestSignedVaa
          );
          if (!assetIdCreated) {
            throw new Error("Failed to create asset");
          }

          // Start of transfer from Terra to Algorand
          // Get initial balance of luna on Terra
          const initialTerraBalance: number = await queryBalanceOnTerra(Asset);

          // Get initial balance of uusd on Terra
          // const initialFeeBalance: number = await queryBalanceOnTerra(FeeAsset);

          // Get initial balance of wrapped luna on Algorand
          const originAssetHex = nativeToHexString(Asset, CHAIN_ID_TERRA);
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

          // Start transfer from Terra to Algorand
          const txMsgs = await transferFromTerra(
            terraWallet.key.accAddress,
            TERRA_TOKEN_BRIDGE_ADDRESS,
            Asset,
            Amount,
            CHAIN_ID_ALGORAND,
            decodeAddress(algoWallet.addr).publicKey // This needs to be Algorand wallet
          );
          const executeTx = await terraWallet.createAndSignTx({
            msgs: txMsgs,
            memo: "Testing transfer...",
            feeDenoms: [FeeAsset],
            gasPrices,
            fee: feeEstimate,
          });
          const txResult = await lcd.tx.broadcast(executeTx);
          const txInfo = await waitForTerraExecution(txResult.txhash);
          if (!txInfo) {
            throw new Error("info not found");
          }

          // Get VAA in order to do redemption step
          const txSn = parseSequenceFromLogTerra(txInfo);
          if (!txSn) {
            throw new Error("Sequence not found");
          }
          const txSignedVaa = await getSignedVAABySequence(
            CHAIN_ID_TERRA,
            txSn,
            emitterAddress
          );
          const redeemTxs = await redeemOnAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            txSignedVaa,
            algoWallet.addr
          );
          await signSendAndConfirmAlgorand(client, redeemTxs, algoWallet);
          expect(
            await getIsTransferCompletedAlgorand(
              client,
              TOKEN_BRIDGE_ID,
              txSignedVaa
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
            parseInt(Amount)
          );

          // Get final balance of uluna on Terra
          const finalTerraBalance = await queryBalanceOnTerra(Asset);

          // Get final balance of uusd on Terra
          // const finalFeeBalance: number = await queryBalanceOnTerra(FeeAsset);
          expect(initialTerraBalance - 1e6 === finalTerraBalance).toBe(true);

          // Start of transfer back to Terra
          const TransferBackAmount: number = 100000;

          // transfer wrapped luna from Algorand to Terra
          const terraHexStr = nativeToHexString(
            terraWallet.key.accAddress,
            CHAIN_ID_TERRA
          );
          if (!terraHexStr) {
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
            terraHexStr,
            CHAIN_ID_TERRA,
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

          const redeemMsg = await redeemOnTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            terraWallet.key.accAddress,
            signedVaa.vaaBytes
          );
          feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await terraWallet.sequence(),
                publicKey: terraWallet.key.publicKey,
              },
            ],
            {
              msgs: [redeemMsg],
              memo: "localhost",
              feeDenoms: [FeeAsset],
              gasPrices,
            }
          );
          const tx = await terraWallet.createAndSignTx({
            msgs: [redeemMsg],
            memo: "localhost",
            feeDenoms: ["uusd"],
            gasPrices,
            fee: feeEstimate,
          });
          await lcd.tx.broadcast(tx);
          expect(
            await getIsTransferCompletedTerra(
              TERRA_TOKEN_BRIDGE_ADDRESS,
              signedVaa.vaaBytes,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(true);

          // Check wallet balances after
          const finalLunaOnTerraBalance = await queryBalanceOnTerra(Asset);
          expect(finalLunaOnTerraBalance - finalTerraBalance).toBe(
            TransferBackAmount
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
          expect(info.chainId).toBe(CHAIN_ID_TERRA);
          expect(info.isWrapped).toBe(true);
        } catch (e) {
          console.error("Terra <=> Algorand error:", e);
          done("Terra <=> Algorand error");
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
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          // attest the test token
          const receipt = await attestFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            TEST_ERC20
          );
          // get the sequence from the logs (needed to fetch the vaa)
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
          await signSendAndConfirmAlgorand(
            client,
            createWrappedTxs,
            algoWallet
          );

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
            ETH_TOKEN_BRIDGE_ADDRESS,
            TEST_ERC20,
            signer,
            amount
          );
          // transfer half the tokens directly
          const firstHalfReceipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            TEST_ERC20,
            halfAmount,
            CHAIN_ID_ALGORAND,
            decodeAddress(algoWallet.addr).publicKey // This needs to be Algorand wallet
          );
          // get the sequence from the logs (needed to fetch the vaa)
          const firstHalfSn = parseSequenceFromLogEth(
            firstHalfReceipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const ethEmitterAddress = getEmitterAddressEth(
            ETH_TOKEN_BRIDGE_ADDRESS
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
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            TEST_ERC20,
            halfAmount,
            CHAIN_ID_ALGORAND,
            decodeAddress(algoWallet.addr).publicKey // This needs to be Algorand wallet
          );
          // get the sequence from the logs (needed to fetch the vaa)
          const secondHalfSn = parseSequenceFromLogEth(
            secondHalfReceipt,
            ETH_CORE_BRIDGE_ADDRESS
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
});
