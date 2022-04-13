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
import axios from "axios";
import { ethers } from "ethers";
import {
  approveEth,
  attestFromEth,
  attestFromSolana,
  attestFromTerra,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  createWrappedOnEth,
  createWrappedOnSolana,
  createWrappedOnTerra,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
  getForeignAssetEth,
  getForeignAssetSolana,
  getForeignAssetTerra,
  getIsTransferCompletedEth,
  getIsTransferCompletedSolana,
  getIsTransferCompletedTerra,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  parseSequenceFromLogTerra,
  postVaaSolana,
  redeemOnEth,
  redeemOnSolana,
  redeemOnTerra,
  TokenImplementation__factory,
  transferFromEth,
  transferFromSolana,
  transferFromTerra,
  updateWrappedOnEth,
} from "../..";
import getSignedVAAWithRetry from "../../rpc/getSignedVAAWithRetry";
import { postVaaWithRetry } from "../../solana/postVaa";
import { setDefaultWasm } from "../../solana/wasm";
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
            hexToUint8Array(nativeToHexString(TEST_ERC20, CHAIN_ID_ETH) || "")
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
            hexToUint8Array(
              nativeToHexString(recipient.toString(), CHAIN_ID_SOLANA) || ""
            )
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
          const originAssetHex = nativeToHexString(
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
            hexToUint8Array(
              nativeToHexString(targetAddress, CHAIN_ID_ETH) || ""
            ),
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
          console.log(
            "Balance on Eth before transfer = ",
            parseInt(initialBalOnEthStr)
          );

          // Get initial balance of ERC20 on Terra
          const originAssetHex = nativeToHexString(ERC20, CHAIN_ID_ETH);
          if (!originAssetHex) {
            throw new Error("originAssetHex is null");
          }
          console.log("ERC20 originAssetHex: ", originAssetHex);
          const foreignAsset = await getForeignAssetTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            lcd,
            CHAIN_ID_ETH,
            hexToUint8Array(originAssetHex)
          );
          if (!foreignAsset) {
            throw new Error("foreignAsset is null");
          }
          console.log("ERC20 foreignAssetHex: ", foreignAsset);
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
          let initialCW20BalOnTerra: number = parseInt(balAmount);
          console.log(
            "CW20 balance on Terra before transfer = ",
            initialCW20BalOnTerra
          );

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
            hexToUint8Array(
              nativeToHexString(wallet.key.accAddress, CHAIN_ID_TERRA) || ""
            )
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
          console.log(
            "Balance on Eth after transfer = ",
            parseInt(finalBalOnEthStr)
          );
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
          let finalCW20BalOnTerra: number = parseInt(balAmount);
          console.log(
            "CW20 balance on Terra after transfer = ",
            finalCW20BalOnTerra
          );
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
            console.log(
              "createWrappedOnEth() failed.  Trying updateWrappedOnEth()..."
            );
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
          console.log("Initial Terra balance of", Asset, initialTerraBalance);

          // Get initial balance of uusd on Terra
          const initialFeeBalance: number = await queryBalanceOnTerra(FeeAsset);
          console.log("Initial Terra balance of", FeeAsset, initialFeeBalance);

          // Get initial balance of wrapped luna on Eth
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          const provider = new ethers.providers.WebSocketProvider(
            ETH_NODE_URL
          ) as any;
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          const originAssetHex = nativeToHexString(Asset, CHAIN_ID_TERRA);
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
          console.log(
            "Luna balance on Eth before transfer = ",
            initialLunaBalOnEthInt
          );

          // Start transfer from Terra to Ethereum
          const hexStr = nativeToHexString(
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
          console.log("Transfer gas used: ", result.gas_used);
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
          console.log("Final Terra balance of", Asset, finalTerraBalance);

          // Get final balance of uusd on Terra
          const finalFeeBalance: number = await queryBalanceOnTerra(FeeAsset);
          console.log("Final Terra balance of", FeeAsset, finalFeeBalance);
          expect(initialTerraBalance - 1e6 === finalTerraBalance).toBe(true);
          const lunaBalOnEthAfter = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const lunaBalOnEthAfterInt = parseInt(lunaBalOnEthAfter._hex);
          console.log(
            "Luna balance on Eth after transfer = ",
            lunaBalOnEthAfterInt
          );
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
          console.log("Terra balance before transfer = ", initialTerraBalance);
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          const provider = new ethers.providers.WebSocketProvider(
            ETH_NODE_URL
          ) as any;
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          const originAssetHex = nativeToHexString(Asset, CHAIN_ID_TERRA);
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
          console.log(
            "Luna balance on Eth before transfer = ",
            initialLunaBalOnEthInt
          );
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
            hexToUint8Array(
              nativeToHexString(wallet.key.accAddress, CHAIN_ID_TERRA) || ""
            )
          );
          console.log("Transfer gas used: ", receipt.gasUsed);

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
          console.log("Terra balance after transfer = ", finalTerraBalance);
          expect(initialTerraBalance + 1e6 === finalTerraBalance).toBe(true);
          const finalLunaBalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const finalLunaBalOnEthInt = parseInt(finalLunaBalOnEth._hex);
          console.log(
            "Luna balance on Eth after transfer = ",
            finalLunaBalOnEthInt
          );
          expect(initialLunaBalOnEthInt - 1e6 === finalLunaBalOnEthInt).toBe(
            true
          );
          const uusdBal = await queryBalanceOnTerra("uusd");
          console.log("uusdBal = ", uusdBal);
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
            console.log(
              "createWrappedOnEth() failed.  Trying updateWrappedOnEth()..."
            );
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
          const initialFeeBalance: number = await queryBalanceOnTerra(FeeAsset);
          console.log("Initial Terra balance of", FeeAsset, initialFeeBalance);

          // Get wallet on eth
          const originAssetHex = nativeToHexString(CW20, CHAIN_ID_TERRA);
          console.log("CW20 originAssetHex: ", originAssetHex);
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
          console.log(
            "CW20 balance on Eth before transfer = ",
            initialCW20BalOnEthInt
          );

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
          console.log(
            "CW20 balance on Terra before transfer = ",
            initialCW20BalOnTerra
          );
          const hexStr = nativeToHexString(
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
          console.log("Finished transferring CW20 token to Ethereum");

          // Check the wallet balances
          let finalCW20BalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          let finalCW20BalOnEthInt = parseInt(finalCW20BalOnEth._hex);
          console.log(
            "CW20 balance on Eth after transfer = ",
            finalCW20BalOnEthInt
          );
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
          console.log(
            "CW20 balance on Terra after transfer = ",
            finalCW20BalOnTerra
          );
          expect(initialCW20BalOnTerra - finalCW20BalOnTerra === 1).toBe(true);
          // Done checking wallet balances

          // Start the reverse transfer from Ethereum back to Terra
          // Get initial wallet balances
          initialCW20BalOnTerra = finalCW20BalOnTerra;
          console.log(
            "CW20 balance on Terra before transfer = ",
            initialCW20BalOnTerra
          );
          initialCW20BalOnEthInt = finalCW20BalOnEthInt;
          console.log(
            "CW20 balance on Eth before transfer = ",
            initialCW20BalOnEthInt
          );

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
            hexToUint8Array(
              nativeToHexString(wallet.key.accAddress, CHAIN_ID_TERRA) || ""
            )
          );
          console.log("Transfer gas used: ", receipt.gasUsed);

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
          console.log(
            "CW20 balance on Eth after transfer = ",
            finalCW20BalOnEthInt
          );
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
          console.log(
            "CW20 balance on Terra after transfer = ",
            finalCW20BalOnTerra
          );
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
});
