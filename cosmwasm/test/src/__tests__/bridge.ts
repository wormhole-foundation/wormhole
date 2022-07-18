import {
  getEmitterAddressTerra,
  parseSequenceFromLogTerra,
  setDefaultWasm,
} from "@certusone/wormhole-sdk";
import { Bech32, toHex } from "@cosmjs/encoding";
import { describe, expect, jest, test } from "@jest/globals";
import { Int, MsgExecuteContract } from "@terra-money/terra.js";
import { keccak256 } from "ethers/lib/utils";
import {
  getNativeBalance,
  makeProviderAndWallet,
  transactWithoutMemo,
} from "../helpers/client";
import { computeGasPaid, parseEventsFromLog } from "../helpers/receipt";
import {
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
const FOREIGN_CHAIN = 2;
const FOREIGN_TOKEN_BRIDGE =
  "0000000000000000000000000290FB167208Af455bB137780163b7B7a9a10C16";
const GUARDIAN_SET_INDEX = 0;
const GUARDIAN_ADDRESS = Buffer.from(
  "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
  "hex"
).toString("base64");
const LUNA_ADDRESS =
  "01" + keccak256(Buffer.from("uluna", "utf-8")).substring(4); // cut off 0x56 (0x prefix - 1 byte)
// const INJ_ADDRESS =
//   "01" + keccak256(Buffer.from("inj", "utf-8")).substring(4); // cut off 0x56 (0x prefix - 1 byte)
const MCK_TOKEN_ID =
  "00" +
  keccak256(
    Buffer.from(
      "terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp",
      "utf-8"
    )
  ).substring(4);
const CONSISTENCY_LEVEL = 0;

const CHAIN_ID = 18;

const WASM_WORMHOLE = "../artifacts/wormhole.wasm";
const WASM_WRAPPED_ASSET = "../artifacts/cw20_wrapped_2.wasm";
const WASM_TOKEN_BRIDGE = "../artifacts/token_bridge_terra_2.wasm";
const WASM_MOCK_BRIDGE_INTEGRATION =
  "../artifacts/mock_bridge_integration_2.wasm";
const WASM_CW20 = "../artifacts/cw20_base.wasm";

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
test("MCK Hashing", () => {
  const knownMck =
    "007160d54c6dabbf464c2f9d95d42f4a30aa6ab2da6c71363917e418c4abe689";
  expect(MCK_TOKEN_ID).toBe(knownMck);
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
            chain_id: 18,
            fee_denom: "uluna"
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
            chain_id: 18,
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
        // test CW20
        const cw20 = await deploy(
          client,
          wallet,
          WASM_CW20,
          {
            name: "MOCK",
            symbol: "MCK",
            decimals: 6,
            initial_balances: [
              {
                address: wallet.key.accAddress,
                amount: "100000000",
              },
            ],
            mint: null,
          },
          "mock"
        );
        console.log("cw20 deployed at", cw20);
        contracts.set("wormhole", wormhole);
        contracts.set("tokenBridge", tokenBridge);
        contracts.set("mockBridgeIntegration", mockBridgeIntegration);
        contracts.set("cw20", cw20);
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
  test("Query Token", (done) => {
    (async () => {
      try {
        const [client] = await makeProviderAndWallet();

        const cw20 = contracts.get("cw20")!;
        const result: any = await client.wasm.contractQuery(cw20, {
          token_info: {},
        });
        console.log(result);
        done();
      } catch (e) {
        console.error(e);
        done("Failed to Query Token");
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
          walletAddress,
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
              payload: Buffer.from(myPayload, "ascii").toString("base64"),
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

        console.log(receipt);

        const jsonLog = JSON.parse(receipt.raw_log);
        let message = "";
        jsonLog.map((row: any) => {
          row.events.map((event: any) => {
            event.attributes.map((attribute: any) => {
              if (attribute.key === "message.message") {
                message = attribute.value;
              }
            });
          });
        });
        // payload type
        let last = 0;
        let len = 2;
        expect(message.substring(last, last + len)).toEqual("03");
        last += len;
        // amount
        len = 64;
        expect(message.substring(last, last + len)).toEqual(
          "0000000000000000000000000000000000000000000000000000000005f5e100"
        );
        last += len;
        // token address
        len = 64;
        expect(message.substring(last, last + len)).toEqual(
          "01fa6c6fbc36d8c245b0a852a43eb5d644e8b4c477b27bfab9537c10945939da"
        );
        last += len;
        // token chain
        len = 4;
        expect(message.substring(last, last + len)).toEqual("0012");
        last += len;
        // recipient address
        len = 64;
        expect(message.substring(last, last + len)).toEqual(
          "0000000000000000000000004206942069420694206942069420694206942069"
        );
        last += len;
        // recipient chain
        len = 4;
        expect(message.substring(last, last + len)).toEqual("0002");
        last += len;
        // sender address
        len = 64;
        expect(message.substring(last, last + len)).toEqual(
          await getEmitterAddressTerra(walletAddress)
        );
        last += len;
        // payload
        expect(message.substring(last)).toEqual(
          Buffer.from(myPayload, "ascii").toString("hex")
        );

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
          relayerFee, // now sender_address
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

        // check contract balance change
        const contractBalanceAfter = await getNativeBalance(
          client,
          mockBridgeIntegration,
          denom
        );
        const contractExpectedChange = new Int(amount);
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
        done();
      } catch (e) {
        console.error(e);
        done("Failed to Attest a Native Asset");
      }
    })();
  });
  test("Attest a CW20", (done) => {
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
                token: {
                  contract_addr: contracts.get("cw20"),
                },
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
        done("Failed to Attest a CW20");
      }
    })();
  });
  test("Register a Foreign Asset", (done) => {
    (async () => {
      try {
        const [client, wallet] = await makeProviderAndWallet();

        const signedVaa =
          "01000000000100efccb8a5d54162691095c88369b873d93ed4ba9365ed0f94adcf39743bb034be56ce0a94d9b163cb0c63be24b94e31b75e1a5889e3de03db147278d9d3eb7d260100000992abb6000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000d0f020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a000212544b4e0000000000000000000000000000000000000000000000000000000000457468657265756d205465737420546f6b656e00000000000000000000000000";

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
        console.log(receipt);
        done();
      } catch (e) {
        console.error(e);
        done("Failed to Register a Foreign Asset");
      }
    })();
  });
  test("Update a Foreign Asset", (done) => {
    (async () => {
      try {
        const [client, wallet] = await makeProviderAndWallet();

        const signedVaa =
          "01000000000100d1389731568d9816267accba90abb9db37dcd09738750fae421067f2f7f33f014c2d862e288e9a3149e2d8bcd2e53ffe2ed72dfc5e8eb50c740a0df34c60103f01000011ac2076010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000e0f020000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a000212544b4e0000000000000000000000000000000000000000000000000000000000457468657265756d205465737420546f6b656e00000000000000000000000000";

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
        console.log(receipt);
        done();
      } catch (e) {
        console.error(e);
        done("Failed to Update a Foreign Asset");
      }
    })();
  });
});

function nativeToHex(address: string) {
  return toHex(Bech32.decode(address).data).padStart(64, "0");
}
