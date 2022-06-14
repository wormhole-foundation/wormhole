import {
  parseSequenceFromLogTerra,
  setDefaultWasm,
} from "@certusone/wormhole-sdk";
import { Bech32, toHex } from "@cosmjs/encoding";
import { describe, expect, jest, test } from "@jest/globals";
import { Int, MsgExecuteContract, TxInfo } from "@terra-money/terra.js";
import { hexZeroPad, keccak256 } from "ethers/lib/utils";
import {
  getNativeBalance,
  makeProviderAndWallet,
  transactWithoutMemo,
} from "../helpers/client";
import { computeGasPaid, parseEventsFromLog } from "../helpers/receipt";
import {
  makeAttestationVaaPayload,
  makeGovernanceVaaPayload,
  makeTransferVaaPayload,
  signAndEncodeVaa,
  TEST_SIGNER_PKS,
} from "../helpers/vaa";
import { deploy, storeCode } from "../instantiate";

setDefaultWasm("node");

jest.setTimeout(60000);

const GOVERNANCE_CHAIN = 1;
const GOVERNANCE_ADDRESS =
  "0000000000000000000000000000000000000000000000000000000000000004";
const FOREIGN_CHAIN = 1;
const FOREIGN_TOKEN_BRIDGE =
  "000000000000000000000000000000000000000000000000000000000000ffff";
const FOREIGN_TOKEN =
  "000000000000000000000000000000000000000000000000000000000000eeee";
const GUARDIAN_SET_INDEX = 0;
const GUARDIAN_ADDRESS = Buffer.from(
  "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
  "hex"
).toString("base64");
const LUNA_ADDRESS =
  "01" + keccak256(Buffer.from("uluna", "utf-8")).substring(4); // cut off 0x56 (0x prefix - 1 byte)
// const INJ_ADDRESS =
//   "01" + keccak256(Buffer.from("inj", "utf-8")).substring(4); // cut off 0x56 (0x prefix - 1 byte)
const CONSISTENCY_LEVEL = 0;

const CHAIN_ID = 18;

const WASM_WORMHOLE = "../artifacts/wormhole.wasm";
const WASM_WRAPPED_ASSET = "../artifacts/cw20_wrapped_2.wasm";
const WASM_TOKEN_BRIDGE = "../artifacts/token_bridge_terra_2.wasm";
const WASM_MOCK_BRIDGE_INTEGRATION =
  "../artifacts/mock_bridge_integration_2.wasm";

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
    > should handle ETH deposits correctly (uluna)
    > should handle ETH withdrawals and fees correctly (uluna)
    > should handle ETH deposits with payload correctly (uluna)
    > should handle ETH withdrawals with payload correctly (uluna)
    > should revert on transfer out of a total of > max(uint64) tokens
    
*/

test("LUNA Hashing", () => {
  const knownLuna =
    "01fa6c6fbc36d8c245b0a852a43eb5d644e8b4c477b27bfab9537c10945939da";
  expect(LUNA_ADDRESS).toBe(knownLuna);
});

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
                  bytes: GUARDIAN_ADDRESS,
                },
              ],
              expiration_time: 0,
            },
          },
          "wormhole"
        );
        console.log("wormhole deployed at", wormhole);
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
          "tokenBridge"
        );
        console.log("tokenBridge deployed at", tokenBridge);
        // mock bridge integration
        const mockBridgeIntegration = await deploy(
          client,
          wallet,
          WASM_MOCK_BRIDGE_INTEGRATION,
          {
            token_bridge_contract: tokenBridge,
          },
          "mockBridgeIntegration"
        );
        console.log("mockBridgeIntegration deployed at", mockBridgeIntegration);
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
  test("Query GuardianSetInfo", (done) => {
    (async () => {
      try {
        const [client] = await makeProviderAndWallet();

        const wormhole = contracts.get("wormhole")!;
        const result: any = await client.wasm.contractQuery(wormhole, {
          guardian_set_info: {},
        });
        expect(result.guardian_set_index).toBe(0);
        expect(result.addresses.length).toBe(1);
        expect(result.addresses[0].bytes).toBe(GUARDIAN_ADDRESS);
        done();
      } catch (e) {
        console.error(e);
        done("Failed to Query GuardianSetInfo");
      }
    })();
  });
  test("Query State", (done) => {
    (async () => {
      try {
        const [client] = await makeProviderAndWallet();

        const wormhole = contracts.get("wormhole")!;
        const result: any = await client.wasm.contractQuery(wormhole, {
          get_state: {},
        });
        expect(result.fee.amount).toBe("0");
        expect(result.fee.denom).toBe("uluna");
        done();
      } catch (e) {
        console.error(e);
        done("Failed to Query State");
      }
    })();
  });
  test("Post a Message", (done) => {
    (async () => {
      try {
        const [client, wallet] = await makeProviderAndWallet();

        const wormhole = contracts.get("wormhole")!;
        const postMessage = new MsgExecuteContract(
          wallet.key.accAddress,
          wormhole,
          {
            post_message: {
              message: Buffer.from("0001020304050607", "hex").toString(
                "base64"
              ),
              nonce: 69,
            },
          }
        );
        const receipt = await transactWithoutMemo(client, wallet, [
          postMessage,
        ]);
        const seq0 = parseSequenceFromLogTerra(receipt as any);
        expect(seq0).toBe("0");
        const receipt2 = await transactWithoutMemo(client, wallet, [
          postMessage,
        ]);
        const seq1 = parseSequenceFromLogTerra(receipt2 as any);
        expect(seq1).toBe("1");
        done();
      } catch (e) {
        console.error(e);
        done("Failed to Post a Message");
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

        // transfer uluna
        const denom = "uluna";
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
        await transactWithoutMemo(client, wallet, [deposit]);
        const receipt = await transactWithoutMemo(client, wallet, [
          initiateTransfer,
        ]);

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

        const denom = "uluna";
        const amount = "100000000"; // one benjamin
        const relayerFee = "1000000"; // one dolla

        const walletAddress = wallet.key.accAddress;
        const recipient = "terra17lmam6zguazs5q5u6z5mmx76uj63gldnse2pdp"; // test2
        const encodedTo = nativeToHex(recipient);

        const vaaPayload = makeTransferVaaPayload(
          1,
          amount,
          LUNA_ADDRESS,
          encodedTo,
          CHAIN_ID,
          relayerFee,
          undefined
        );

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

        // check wallet (relayer) balance change
        const walletBalanceAfter = await getNativeBalance(
          client,
          walletAddress,
          denom
        );
        const gasPaid = computeGasPaid(receipt);
        const walletExpectedChange = new Int(relayerFee).sub(gasPaid);

        // due to rounding, we should expect the balances to reconcile
        // within 1 unit (equivalent to 1e-6 uluna). Best-case scenario
        // we end up with slightly more balance than expected
        const reconciled = walletBalanceAfter
          .minus(walletExpectedChange)
          .minus(walletBalanceBefore);
        expect(
          reconciled.greaterThanOrEqualTo("0") &&
            reconciled.lessThanOrEqualTo("1")
        ).toBeTruthy();

        const recipientBalanceAfter = await getNativeBalance(
          client,
          recipient,
          denom
        );
        const recipientExpectedChange = new Int(amount).sub(relayerFee);
        expect(
          recipientBalanceBefore
            .add(recipientExpectedChange)
            .eq(recipientBalanceAfter)
        ).toBeTruthy();

        // check bridge balance change
        const bridgeExpectedChange = new Int(amount);
        const bridgeBalanceAfter = await getNativeBalance(
          client,
          tokenBridge,
          denom
        );
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

        // transfer uluna
        const denom = "uluna";
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

        const denom = "uluna";
        const amount = "100000000"; // one benjamin
        const relayerFee = "1000000"; // one dolla

        const walletAddress = wallet.key.accAddress;

        const encodedTo = nativeToHex(mockBridgeIntegration);
        const additionalPayload = "All your base are belong to us";

        const vaaPayload = makeTransferVaaPayload(
          3,
          amount,
          LUNA_ADDRESS,
          encodedTo,
          CHAIN_ID,
          relayerFee,
          additionalPayload
        );

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

        // check wallet (relayer) balance change
        const walletBalanceAfter = await getNativeBalance(
          client,
          walletAddress,
          denom
        );
        const gasPaid = computeGasPaid(receipt);
        const walletExpectedChange = new Int(relayerFee).sub(gasPaid);

        // due to rounding, we should expect the balances to reconcile
        // within 1 unit (equivalent to 1e-6 uluna). Best-case scenario
        // we end up with slightly more balance than expected
        const reconciled = walletBalanceAfter
          .minus(walletExpectedChange)
          .minus(walletBalanceBefore);
        expect(
          reconciled.greaterThanOrEqualTo("0") &&
            reconciled.lessThanOrEqualTo("1")
        ).toBeTruthy();

        // check contract balance change
        const contractBalanceAfter = await getNativeBalance(
          client,
          mockBridgeIntegration,
          denom
        );
        const contractExpectedChange = new Int(amount).sub(relayerFee);
        expect(
          contractBalanceBefore
            .add(contractExpectedChange)
            .eq(contractBalanceAfter)
        ).toBeTruthy();

        // cehck bridge balance change
        const bridgeExpectedChange = new Int(amount);
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

        const denom = "uluna";
        const amount = "100000000"; // one benjamin
        const relayerFee = "1000000"; // one dolla

        const walletAddress = wallet.key.accAddress;

        const encodedTo = nativeToHex(mockBridgeIntegration);
        const additionalPayload = "All your base are belong to us";

        const vaaPayload = makeTransferVaaPayload(
          3,
          amount,
          LUNA_ADDRESS,
          encodedTo,
          CHAIN_ID,
          relayerFee,
          additionalPayload
        );

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
  test("Attest a Native Asset", (done) => {
    (async () => {
      try {
        const [client, wallet] = await makeProviderAndWallet();

        const tokenBridge = contracts.get("tokenBridge")!;
        const attest = new MsgExecuteContract(
          wallet.key.accAddress,
          tokenBridge,
          {
            create_asset_meta: {
              asset_info: {
                native_token: { denom: "uluna" },
              },
              nonce: 69,
            },
          }
        );

        const receipt = await transactWithoutMemo(client, wallet, [attest]);
        console.log(receipt);
        done();
      } catch (e) {
        console.error(e);
        done("Failed to Attest a Native Asset");
      }
    })();
  });
  // test("Register a Foreign Asset", (done) => {
  //   (async () => {
  //     try {
  //       const [client, wallet] = await makeProviderAndWallet();

  //       const vaaPayload = makeAttestationVaaPayload(
  //         FOREIGN_CHAIN,
  //         FOREIGN_TOKEN,
  //         18,
  //         hexZeroPad("0x" + Buffer.from("TEST", "utf-8").toString("hex"), 32),
  //         hexZeroPad(
  //           "0x" + Buffer.from("Testy Token", "utf-8").toString("hex"),
  //           32
  //         )
  //       );

  //       const timestamp = 1;
  //       const nonce = 1;
  //       const sequence = 0;

  //       const signedVaa = signAndEncodeVaa(
  //         timestamp,
  //         nonce,
  //         GOVERNANCE_CHAIN,
  //         GOVERNANCE_ADDRESS,
  //         sequence,
  //         vaaPayload,
  //         TEST_SIGNER_PKS,
  //         GUARDIAN_SET_INDEX,
  //         CONSISTENCY_LEVEL
  //       );

  //       const tokenBridge = contracts.get("tokenBridge")!;
  //       const submitVaa = new MsgExecuteContract(
  //         wallet.key.accAddress,
  //         tokenBridge,
  //         {
  //           submit_vaa: {
  //             data: Buffer.from(signedVaa, "hex").toString("base64"),
  //           },
  //         }
  //       );

  //       const receipt = await transactWithoutMemo(client, wallet, [submitVaa]);
  //       console.log(receipt);
  //       done();
  //     } catch (e) {
  //       console.error(e);
  //       done("Failed to Register a Foreign Asset");
  //     }
  //   })();
  // });
});

function nativeToHex(address: string) {
  return toHex(Bech32.decode(address).data).padStart(64, "0");
}
