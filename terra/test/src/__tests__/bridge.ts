import { describe, expect, jest, test } from "@jest/globals";
import { Bech32, toHex } from "@cosmjs/encoding";
import {
  getNativeBalance,
  makeProviderAndWallet,
  transactWithoutMemo,
} from "../helpers/client";
import { storeCode, deploy } from "../instantiate";
import { Int, MsgExecuteContract } from "@terra-money/terra.js";
import {
  makeGovernanceVaaPayload,
  makeTransferVaaPayload,
  signAndEncodeVaa,
  TEST_SIGNER_PKS,
} from "../helpers/vaa";
import { computeGasPaid, parseEventsFromLog } from "../helpers/receipt";

jest.setTimeout(60000);

const GOVERNANCE_CHAIN = 1;
const GOVERNANCE_ADDRESS =
  "0000000000000000000000000000000000000000000000000000000000000004";
const FOREIGN_CHAIN = 1;
const FOREIGN_TOKEN_BRIDGE =
  "000000000000000000000000000000000000000000000000000000000000ffff";
const GUARDIAN_SET_INDEX = 0;
const CONSISTENCY_LEVEL = 0;

const WASM_WORMHOLE = "../artifacts/wormhole.wasm";
const WASM_WRAPPED_ASSET = "../artifacts/cw20_wrapped.wasm";
const WASM_TOKEN_BRIDGE = "../artifacts/token_bridge_terra.wasm";
const WASM_MOCK_BRIDGE_INTEGRATION =
  "../artifacts/mock_bridge_integration.wasm";

// global map of contract addresses for all tests
const contracts = new Map<string, string>();

/*
    Mirror ethereum/test/bridge.js

    > should be initialized with the correct signers and values
    > should register a foreign bridge implementation correctly
    > should accept a valid upgrade
    > bridged tokens should only be mint- and burn-able by owner (??)
    > should attest a token correctly
    > should correctly deploy a wrapped asset for a token attestation
    > should correctly update a wrapped asset for a token attestation
    > should deposit and log transfers correctly
    > should deposit and log fee token transfers correctly
    > should transfer out locked assets for a valid transfer vm
    > should deposit and log transfer with payload correctly
    > should transfer out locked assets for a valid transfer with payload vm
    > should mint bridged assets wrappers on transfer from another chain and handle fees correctly
    > should handle additional data on token bridge transfer with payload in single transaction when feeRecipient == transferRecipient
    > should not allow a redemption from msg.sender other than 'to' on token bridge transfer with payload
    > should allow a redemption from msg.sender == 'to' on token bridge transfer with payload and check that sender recieves fee
    > should burn bridged assets wrappers on transfer to another chain
    > should handle ETH deposits correctly (uusd)
    > should handle ETH withdrawals and fees correctly (uusd)
    > should handle ETH deposits with payload correctly (uusd)
    > should handle ETH withdrawals with payload correctly (uusd)
    > should revert on transfer out of a total of > max(uint64) tokens

*/

describe("Bridge Tests", () => {
  test("Deploy Contracts", (done) => {
    (async () => {
      try {
        const [client, wallet] = await makeProviderAndWallet();

        const governanceAddress = Buffer.from(
          GOVERNANCE_ADDRESS,
          "hex"
        ).toString("base64");

        // wormhole
        const wormhole = await deploy(
          client,
          wallet,
          WASM_WORMHOLE,
          {
            gov_chain: GOVERNANCE_CHAIN,
            gov_address: governanceAddress,
            guardian_set_expirity: 86400,
            initial_guardian_set: {
              addresses: [
                {
                  bytes: Buffer.from(
                    "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
                    "hex"
                  ).toString("base64"),
                },
              ],
              expiration_time: 0,
            },
          },
          "wormholeTest"
        );

        // token bridge
        const wrappedAssetCodeId = await storeCode(
          client,
          wallet,
          WASM_WRAPPED_ASSET
        );
        const tokenBridge = await deploy(
          client,
          wallet,
          WASM_TOKEN_BRIDGE,
          {
            gov_chain: GOVERNANCE_CHAIN,
            gov_address: governanceAddress,
            wormhole_contract: wormhole,
            wrapped_asset_code_id: wrappedAssetCodeId,
          },
          "tokenBridgeTest"
        );

        // mock bridge integration
        const mockBridgeIntegration = await deploy(
          client,
          wallet,
          WASM_MOCK_BRIDGE_INTEGRATION,
          {
            token_bridge_contract: tokenBridge,
          },
          "mockBrigeIntegration"
        );
        contracts.set("wormhole", wormhole);
        contracts.set("tokenBridge", tokenBridge);
        contracts.set("mockBridgeIntegration", mockBridgeIntegration);
        done();
      } catch (e) {
        console.error(e);
        done("Failed to Deploy Contracts");
      }
    })();
  });
  test("Register a Foreign Bridge Implementation", (done) => {
    (async () => {
      try {
        const [client, wallet] = await makeProviderAndWallet();

        const vaaPayload = makeGovernanceVaaPayload(
          GOVERNANCE_CHAIN,
          FOREIGN_CHAIN,
          FOREIGN_TOKEN_BRIDGE
        );
        console.info("vaaPayload", vaaPayload);

        const timestamp = 1;
        const nonce = 1;
        const sequence = 0;

        const signedVaa = signAndEncodeVaa(
          timestamp,
          nonce,
          GOVERNANCE_CHAIN,
          GOVERNANCE_ADDRESS,
          sequence,
          vaaPayload,
          TEST_SIGNER_PKS,
          GUARDIAN_SET_INDEX,
          CONSISTENCY_LEVEL
        );
        console.info("signedVaa", signedVaa);

        const tokenBridge = contracts.get("tokenBridge")!;
        const submitVaa = new MsgExecuteContract(
          wallet.key.accAddress,
          tokenBridge,
          {
            submit_vaa: {
              data: Buffer.from(signedVaa, "hex").toString("base64"),
            },
          }
        );

        const receipt = await transactWithoutMemo(client, wallet, [submitVaa]);
        console.info("receipt", receipt.txhash);
        done();
      } catch (e) {
        console.error(e);
        done("Failed to Register a Foreign Bridge Implementation");
      }
    })();
  });
  test("Initiate Transfer (native denom)", (done) => {
    (async () => {
      try {
        const [client, wallet] = await makeProviderAndWallet();

        const tokenBridge = contracts.get("tokenBridge")!;

        // transfer uusd
        const denom = "uusd";
        const recipientAddress =
          "0000000000000000000000004206942069420694206942069420694206942069";
        const amount = "100000000"; // one benjamin
        const relayerFee = "1000000"; // one dolla

        const walletAddress = wallet.key.accAddress;

        // need to deposit before initiating transfer
        const deposit = new MsgExecuteContract(
          wallet.key.accAddress,
          tokenBridge,
          {
            deposit_tokens: {},
          },
          { [denom]: amount }
        );

        const initiateTransfer = new MsgExecuteContract(
          walletAddress,
          tokenBridge as string,
          {
            initiate_transfer: {
              asset: {
                amount,
                info: {
                  native_token: {
                    denom,
                  },
                },
              },
              recipient_chain: 2,
              recipient: Buffer.from(recipientAddress, "hex").toString(
                "base64"
              ),
              fee: relayerFee,
              nonce: 69,
            },
          }
        );

        // check balances
        const balanceBefore = await getNativeBalance(
          client,
          tokenBridge,
          denom
        );

        // execute outbound transfer
        const receipt = await transactWithoutMemo(client, wallet, [
          deposit,
          initiateTransfer,
        ]);
        console.info("receipt", receipt.txhash);

        const balanceAfter = await getNativeBalance(client, tokenBridge, denom);
        expect(balanceBefore.add(amount).eq(balanceAfter)).toBeTruthy();

        done();
      } catch (e) {
        console.error(e);
        done("Failed to Initiate Transfer (native denom)");
      }
    })();
  });
  test("Complete Transfer (native denom)", (done) => {
    (async () => {
      try {
        const [client, wallet] = await makeProviderAndWallet();

        const tokenBridge = contracts.get("tokenBridge")!;

        const denom = "uusd";
        const amount = "100000000"; // one benjamin
        const relayerFee = "1000000"; // one dolla

        const walletAddress = wallet.key.accAddress;
        const recipient = "terra17lmam6zguazs5q5u6z5mmx76uj63gldnse2pdp"; // test2
        const encodedTo = nativeToHex(recipient);
        console.log("encodedTo", encodedTo);
        const ustAddress =
          "0100000000000000000000000000000000000000000000000000000075757364";

        const vaaPayload = makeTransferVaaPayload(
          1,
          amount,
          ustAddress,
          encodedTo,
          3,
          relayerFee,
          undefined
        );
        console.info("vaaPayload", vaaPayload);

        const timestamp = 0;
        const nonce = 0;
        const sequence = 0;

        const signedVaa = signAndEncodeVaa(
          timestamp,
          nonce,
          FOREIGN_CHAIN,
          FOREIGN_TOKEN_BRIDGE,
          sequence,
          vaaPayload,
          TEST_SIGNER_PKS,
          GUARDIAN_SET_INDEX,
          CONSISTENCY_LEVEL
        );
        console.info("signedVaa", signedVaa);

        // check balances
        const walletBalanceBefore = await getNativeBalance(
          client,
          walletAddress,
          denom
        );
        const recipientBalanceBefore = await getNativeBalance(
          client,
          recipient,
          denom
        );
        const bridgeBalanceBefore = await getNativeBalance(
          client,
          tokenBridge,
          denom
        );

        const submitVaa = new MsgExecuteContract(walletAddress, tokenBridge, {
          submit_vaa: {
            data: Buffer.from(signedVaa, "hex").toString("base64"),
          },
        });

        // execute outbound transfer with signed vaa
        const receipt = await transactWithoutMemo(client, wallet, [submitVaa]);
        console.info("receipt", receipt.txhash);

        // check wallet (relayer) balance change
        const walletBalanceAfter = await getNativeBalance(
          client,
          walletAddress,
          denom
        );
        const gasPaid = computeGasPaid(receipt);
        const walletExpectedChange = new Int("882906");

        // due to rounding, we should expect the balances to reconcile
        // within 1 unit (equivalent to 1e-6 uusd). Best-case scenario
        // we end up with slightly more balance than expected
        const reconciled = walletBalanceAfter
          .minus(walletExpectedChange)
          .minus(walletBalanceBefore);
        console.info("reconciled", reconciled.toString());
        expect(
          reconciled.greaterThanOrEqualTo("0") &&
            reconciled.lessThanOrEqualTo("1")
        ).toBeTruthy();

        const recipientBalanceAfter = await getNativeBalance(
          client,
          recipient,
          denom
        );
        // the expected change is slightly less than the amount - the relayer fee, due to tax
        const recipientExpectedChange = new Int("98901098");
        expect(
          recipientBalanceBefore
            .add(recipientExpectedChange)
            .eq(recipientBalanceAfter)
        ).toBeTruthy();

        // check bridge balance change
        // the expected change is slightly less than the amount, due to
        // a small rounding error in the tax calculation
        const bridgeExpectedChange = new Int("99999998");
        const bridgeBalanceAfter = await getNativeBalance(
          client,
          tokenBridge,
          denom
        );
        console.info("bridgeBalanceAfter", bridgeBalanceAfter.toString());
        console.info("bridgeExpectedChange", bridgeExpectedChange.toString());
        console.info("bridgeBalanceBefore", bridgeBalanceBefore.toString());
        expect(
          bridgeBalanceBefore.sub(bridgeExpectedChange).eq(bridgeBalanceAfter)
        ).toBeTruthy();

        done();
      } catch (e) {
        console.error(e);
        done("Failed to Complete Transfer (native denom)");
      }
    })();
  });
  // transfer with payload tests
  test("Initiate Transfer With Payload (native denom)", (done) => {
    (async () => {
      try {
        const [client, wallet] = await makeProviderAndWallet();

        const tokenBridge = contracts.get("tokenBridge")!;

        // transfer uusd
        const denom = "uusd";
        const recipientAddress =
          "0000000000000000000000004206942069420694206942069420694206942069";
        const amount = "100000000"; // one benjamin
        const relayerFee = "1000000"; // one dolla
        const myPayload = "ABC";

        const walletAddress = wallet.key.accAddress;

        // need to deposit before initiating transfer
        const deposit = new MsgExecuteContract(
          wallet.key.accAddress,
          tokenBridge,
          {
            deposit_tokens: {},
          },
          { [denom]: amount }
        );

        const initiateTransferWithPayload = new MsgExecuteContract(
          walletAddress,
          tokenBridge as string,
          {
            initiate_transfer_with_payload: {
              asset: {
                amount,
                info: {
                  native_token: {
                    denom,
                  },
                },
              },
              recipient_chain: 2,
              recipient: Buffer.from(recipientAddress, "hex").toString(
                "base64"
              ),
              fee: relayerFee,
              payload: Buffer.from(myPayload, "hex").toString("base64"),
              nonce: 69,
            },
          }
        );

        // check balances
        const balanceBefore = await getNativeBalance(
          client,
          tokenBridge,
          denom
        );

        // execute outbound transfer with payload
        const receipt = await transactWithoutMemo(client, wallet, [
          deposit,
          initiateTransferWithPayload,
        ]);
        console.info("receipt txHash", receipt.txhash);

        const balanceAfter = await getNativeBalance(client, tokenBridge, denom);
        expect(balanceBefore.add(amount).eq(balanceAfter)).toBeTruthy();

        done();
      } catch (e) {
        console.error(e);
        done("Failed to Initiate Transfer With Payload (native denom)");
      }
    })();
  });
  test("Complete Transfer With Payload (native denom)", (done) => {
    (async () => {
      try {
        const [client, wallet] = await makeProviderAndWallet();

        const tokenBridge = contracts.get("tokenBridge")!;
        const mockBridgeIntegration = contracts.get("mockBridgeIntegration")!;

        const denom = "uusd";
        const amount = "100000000"; // one benjamin

        const walletAddress = wallet.key.accAddress;

        const encodedTo = nativeToHex(mockBridgeIntegration);
        console.log("encodedTo", encodedTo);
        const ustAddress =
          "0100000000000000000000000000000000000000000000000000000075757364";
        const additionalPayload = "All your base are belong to us";

        const vaaPayload = makeTransferVaaPayload(
          3,
          amount,
          ustAddress,
          encodedTo,
          3,
          "0",
          additionalPayload
        );
        console.info("vaaPayload", vaaPayload);

        const timestamp = 1;
        const nonce = 1;
        const sequence = 2;

        const signedVaa = signAndEncodeVaa(
          timestamp,
          nonce,
          FOREIGN_CHAIN,
          FOREIGN_TOKEN_BRIDGE,
          sequence,
          vaaPayload,
          TEST_SIGNER_PKS,
          GUARDIAN_SET_INDEX,
          CONSISTENCY_LEVEL
        );
        console.info("signedVaa", signedVaa);

        // check balances before execute
        const walletBalanceBefore = await getNativeBalance(
          client,
          walletAddress,
          denom
        );
        const contractBalanceBefore = await getNativeBalance(
          client,
          mockBridgeIntegration,
          denom
        );
        const bridgeBalanceBefore = await getNativeBalance(
          client,
          tokenBridge,
          denom
        );

        const submitVaa = new MsgExecuteContract(
          walletAddress,
          mockBridgeIntegration,
          {
            complete_transfer_with_payload: {
              data: Buffer.from(signedVaa, "hex").toString("base64"),
            },
          }
        );

        // execute outbound transfer with signed vaa
        const receipt = await transactWithoutMemo(client, wallet, [submitVaa]);
        console.info("receipt txHash", receipt.txhash);

        // check contract balance change
        const contractBalanceAfter = await getNativeBalance(
          client,
          mockBridgeIntegration,
          denom
        );
        // tax applied, so we expect less than the original amount
        const contractBlanceAfterExpected = new Int("99900099");
        expect(
          contractBalanceAfter
            .eq(contractBlanceAfterExpected)
        ).toBeTruthy();

        // check bridge balance change
        // the expected change is slightly less than the amount, due to
        // a small rounding error in the tax calculation
        const bridgeExpectedChange = new Int("99999999");
        const bridgeBalanceAfter = await getNativeBalance(
          client,
          tokenBridge,
          denom
        );
        expect(
          bridgeBalanceBefore.sub(bridgeExpectedChange).eq(bridgeBalanceAfter)
        ).toBeTruthy();

        // verify payload
        const events = parseEventsFromLog(receipt);
        const response: any[] = events.find((event) => {
          return event.type == "wasm";
        }).attributes;

        const transferPayloadResponse = response.find((item) => {
          return item.key == "transfer_payload";
        });
        expect(
          Buffer.from(transferPayloadResponse.value, "base64").toString()
        ).toEqual(additionalPayload);

        done();
      } catch (e) {
        console.error(e);
        done("Failed to Complete Transfer With Payload (native denom)");
      }
    })();
  });
  test("Throw on Complete Transfer With Payload If Someone Else Redeems VAA", (done) => {
    (async () => {
      try {
        const [client, wallet] = await makeProviderAndWallet();

        const tokenBridge = contracts.get("tokenBridge")!;
        const mockBridgeIntegration = contracts.get("mockBridgeIntegration")!;

        const denom = "uusd";
        const amount = "100000000"; // one benjamin
        const relayerFee = "1000000"; // one dolla

        const walletAddress = wallet.key.accAddress;

        const encodedTo = nativeToHex(mockBridgeIntegration);
        console.log("encodedTo", encodedTo);
        const ustAddress =
          "0100000000000000000000000000000000000000000000000000000075757364";
        const additionalPayload = "All your base are belong to us";

        const vaaPayload = makeTransferVaaPayload(
          3,
          amount,
          ustAddress,
          encodedTo,
          3,
          relayerFee,
          additionalPayload
        );
        console.info("vaaPayload", vaaPayload);

        const timestamp = 1;
        const nonce = 1;
        const sequence = 3;

        const signedVaa = signAndEncodeVaa(
          timestamp,
          nonce,
          FOREIGN_CHAIN,
          FOREIGN_TOKEN_BRIDGE,
          sequence,
          vaaPayload,
          TEST_SIGNER_PKS,
          GUARDIAN_SET_INDEX,
          CONSISTENCY_LEVEL
        );
        console.info("signedVaa", signedVaa);

        let expectedErrorFound = false;
        try {
          const submitVaa = new MsgExecuteContract(walletAddress, tokenBridge, {
            complete_transfer_with_payload: {
              data: Buffer.from(signedVaa, "hex").toString("base64"),
              relayer: walletAddress,
            },
          });

          // execute outbound transfer with signed vaa
          const receipt = await transactWithoutMemo(client, wallet, [
            submitVaa,
          ]);
          console.info("receipt txHash", receipt.txhash);
        } catch (e) {
          const errorMsg: string = e.response.data.message;
          expectedErrorFound = errorMsg.includes(
            "transfers with payload can only be redeemed by the recipient"
          );
        }

        expect(expectedErrorFound).toBeTruthy();

        done();
      } catch (e) {
        console.error(e);
        done(
          "Failed to Throw on Complete Transfer With Payload If Someone Else Redeems VAA"
        );
      }
    })();
  });
});

function nativeToHex(address: string) {
  return toHex(Bech32.decode(address).data).padStart(64, "0");
}
