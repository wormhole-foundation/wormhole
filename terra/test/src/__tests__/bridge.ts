import { describe, expect, jest, test } from "@jest/globals";
import { Bech32, toHex } from "@cosmjs/encoding";
import { Int, MsgExecuteContract } from "@terra-money/terra.js";

import { makeProviderAndWallet, transactWithoutMemo } from "../helpers/client";
import {
  makeGovernanceVaaPayload,
  makeTransferVaaPayload,
  signAndEncodeVaa,
  TEST_SIGNER_PKS,
} from "../helpers/vaa";
import { storeCode, deploy } from "../instantiate";

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
const WASM_TOKEN_BRIDGE = "../artifacts/token_bridge.wasm";

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
    > should mint bridged assets wrappers on transfer from another chain and handle fees correctly
    > should burn bridged assets wrappers on transfer to another chain
    > should handle ETH deposits correctly (uusd)
    > should handle ETH withdrawals and fees correctly (uusd)
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
        const wormhole = await deploy(client, wallet, WASM_WORMHOLE, {
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
        });

        // token bridge
        const wrappedAssetCodeId = await storeCode(
          client,
          wallet,
          WASM_WRAPPED_ASSET
        );
        const tokenBridge = await deploy(client, wallet, WASM_TOKEN_BRIDGE, {
          gov_chain: GOVERNANCE_CHAIN,
          gov_address: governanceAddress,
          wormhole_contract: wormhole,
          wrapped_asset_code_id: wrappedAssetCodeId,
        });

        contracts.set("wormhole", wormhole);
        contracts.set("tokenBridge", tokenBridge);
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
        let balanceBefore = new Int(0);
        {
          const [balance] = await client.bank.balance(tokenBridge);
          const coin = balance.get(denom);
          if (coin !== undefined) {
            balanceBefore = new Int(coin.amount);
          }
        }

        // execute outbound transfer
        const receipt = await transactWithoutMemo(client, wallet, [
          deposit,
          initiateTransfer,
        ]);
        console.info("receipt", receipt.txhash);

        let balanceAfter: Int;
        {
          const [balance] = await client.bank.balance(tokenBridge);
          const coin = balance.get(denom);
          expect(!coin).toBeFalsy();

          balanceAfter = new Int(coin!.amount);
        }
        expect(
          balanceBefore.add(new Int(amount)).eq(balanceAfter)
        ).toBeTruthy();

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
          const recipient = "terra17lmam6zguazs5q5u6z5mmx76uj63gldnse2pdp";
          
        // check balances
        let balanceBefore = new Int(0);
        {
          const [balance] = await client.bank.balance(recipient);
          const coin = balance.get(denom);
          if (coin !== undefined) {
            balanceBefore = new Int(coin.amount);
          }
        }

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

        const submitVaa = new MsgExecuteContract(walletAddress, tokenBridge, {
          submit_vaa: {
            data: Buffer.from(signedVaa, "hex").toString("base64"),
          },
        });

        // execute inbound transfer with signed vaa
        const receipt = await transactWithoutMemo(client, wallet, [submitVaa]);
        console.info("receipt", receipt.txhash);

        let balanceAfter: Int;
        {
          const [balance] = await client.bank.balance(recipient);
          const coin = balance.get(denom);
          expect(!coin).toBeFalsy();

          balanceAfter = new Int(coin!.amount);
        }          
          const expectedAmount = (new Int(amount)).sub(relayerFee);
        expect(
          //balanceBefore.add(new Int(expectedAmount)).eq(balanceAfter)
          balanceBefore.add(expectedAmount).eq(balanceAfter)
        ).toBeTruthy();

        done();
      } catch (e) {
        console.error(e);
        done("Failed to Complete Transfer (native denom)");
      }
    })();
  });
});

function nativeToHex(address: string) {
  return toHex(Bech32.decode(address).data).padStart(64, "0");
}
