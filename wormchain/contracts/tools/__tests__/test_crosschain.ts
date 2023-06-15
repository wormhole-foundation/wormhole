import "dotenv/config";
import * as os from "os";
import { describe, expect, jest, test } from "@jest/globals";
import {
  approveEth,
  attestFromEth,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_TERRA2,
  CHAIN_ID_WORMCHAIN,
  ChainId,
  CONTRACTS,
  createWrappedOnEth,
  createWrappedOnTerra,
  getEmitterAddressEth,
  getEmitterAddressTerra,
  getForeignAssetEth,
  getIsTransferCompletedTerra2,
  getSignedVAAWithRetry,
  hexToUint8Array,
  parseSequenceFromLogEth,
  parseSequenceFromLogTerra,
  parseTokenTransferVaa,
  parseVaa,
  redeemOnEth,
  redeemOnTerra,
  serialiseVAA,
  sign,
  TokenBridgeTransfer,
  transferFromEth,
  tryNativeToHexString,
  tryNativeToUint8Array,
  uint8ArrayToHex,
  VAA,
} from "@certusone/wormhole-sdk";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { ethers } from "ethers";
import * as devnetConsts from "../devnet-consts.json";
import { arrayify, parseUnits } from "ethers/lib/utils";
import { SigningCosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import { GasPrice } from "@cosmjs/stargate";
import { Secp256k1HdWallet } from "@cosmjs/amino";
import {
  LCDClient,
  MnemonicKey,
  Msg,
  MsgExecuteContract,
  TxInfo,
  Wallet,
  isTxError,
} from "@terra-money/terra.js";
import { fromUint8Array } from "js-base64";

jest.setTimeout(60000);

if (process.env.INIT_SIGNERS_KEYS_CSV === "undefined") {
  let msg = `.env is missing. run "make contracts-tools-deps" to fetch.`;
  console.error(msg);
  throw msg;
}

/*
 * Goals:
 *   1. Attempt to attest Eth on wormchain.
 *   2. Attempt to send a payload 1 transfer through wormchain and watch it fail.
 *   3. Attempt to send a payload 3 transfer through wormchain and watch it succeed.
 *
 */

const ci = !!process.env.CI;

const WORMCHAIN_ID = 3104;
const GUARDIAN_HOST = ci ? "guardian" : "localhost";
const GUARDIAN_RPCS = [`http://${GUARDIAN_HOST}:7071`];
const GUARDIAN_METRICS = `http://${GUARDIAN_HOST}:6060/metrics`;
const ETH_NODE_URL = ci ? "ws://eth-devnet:8545" : "ws://localhost:8545";
const BSC_NODE_URL = ci ? "ws://eth-devnet2:8545" : "ws://localhost:8546";
const ETH_PRIVATE_KEY9 =
  "0xb0057716d5917badaf911b193b12b910811c1497b5bada8d7711f758981c3773";
const ETH_GA_TEST_TOKEN =
  devnetConsts.chains[CHAIN_ID_ETH].addresses.testGA.address;
const DECIMALS = devnetConsts.chains[CHAIN_ID_ETH].addresses.testGA.decimals;
const VAA_SIGNERS = process.env.INIT_SIGNERS_KEYS_CSV.split(",");
const GOVERNANCE_CHAIN = Number(devnetConsts.global.governanceChainId);
const GOVERNANCE_EMITTER = devnetConsts.global.governanceEmitterAddress;
const TENDERMINT_URL = ci ? "http://wormchain:26657" : "http://localhost:26659";
const GA_ADDRESS =
  "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465";
const IBC_GATEWAY_ADDRESS =
  "wormhole1436kxs0w2es6xlqpp9rd35e3d0cjnw4sv8j3a7483sgks29jqwgsqyfker";
const IBC_TB_ADDRESS =
  "wormhole1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqhnev3f";
const CROSS_CHAIN_ADDRESS =
  "wormhole1mf6ptkssddfmxvhdx0ech0k03ktp6kf9yk59renau2gvht3nq2gq6n0sg2";
const CROSS_CHAIN_TOKEN_BRIDGE_ADDRESS =
  "wormhole1436kxs0w2es6xlqpp9rd35e3d0cjnw4sv8j3a7483sgks29jqwgsqyfker";
export const ETH_PRIVATE_KEY2 =
  "0x6370fd033278c143179d81c5526140625662b8daa446c22ee2d73db3707e620c"; // account 2 - terra2 tests
const TEST_ERC20 = "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A";
const WORMHOLE_RPC_HOSTS = ci
  ? ["http://guardian:7071"]
  : ["http://localhost:7071"];
export const TERRA_NODE_URL = ci
  ? "http://terra-terrad:1317"
  : "http://localhost:1317";
const TERRA2_NODE_URL = ci
  ? "http://terra2-terrad:1317"
  : "http://localhost:1318";
const TERRA_CHAIN_ID = "localterra";
const TERRA2_GAS_PRICES_URL = ci
  ? "http://terra2-fcd:3060/v1/txs/gas_prices"
  : "http://localhost:3061/v1/txs/gas_prices";
const TERRA2_PRIVATE_KEY =
  "symbol force gallery make bulk round subway violin worry mixture penalty kingdom boring survey tool fringe patrol sausage hard admit remember broken alien absorb"; // test3
const TERRA_PRIVATE_KEY2 =
  "quality vacuum heart guard buzz spike sight swarm shove special gym robust assume sudden deposit grid alcohol choice devote leader tilt noodle tide penalty"; // test2
function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
const wlcd = new LCDClient({
  URL: TENDERMINT_URL,
  chainID: "3104",
});
const lcd = new LCDClient({
  URL: TERRA2_NODE_URL,
  chainID: TERRA_CHAIN_ID,
});
const terraWallet = lcd.wallet(
  new MnemonicKey({ mnemonic: TERRA2_PRIVATE_KEY })
);
const terraWalletAddress = terraWallet.key.accAddress;

const lcdClassic = new LCDClient({
  URL: TERRA_NODE_URL,
  chainID: TERRA_CHAIN_ID,
  isClassic: true,
});
const terraClassicWallet = lcdClassic.wallet(
  new MnemonicKey({ mnemonic: TERRA_PRIVATE_KEY2 })
);
const terraClassicWalletAddress = terraClassicWallet.key.accAddress;

const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
const signer = new ethers.Wallet(ETH_PRIVATE_KEY2, provider);
const ethEmitterAddress = getEmitterAddressEth(
  CONTRACTS.DEVNET.ethereum.token_bridge
);
const ethTransferAmount = parseUnits("1", 18);

let ethWalletAddress: string;
let terraEmitterAddress: string;

type BasicTransfer = {
  basic_transfer: { chain_id: number; recipient: string };
};

type ParsedPayload = {
  payloadID: number;
  amount: Uint8Array;
  tokenAddress: string;
  tokenChain: number;
  to: Uint8Array;
  toChain: number;
  fromAddress: Uint8Array;
  payload: Uint8Array;
};

function parsePayload(pl: Buffer): ParsedPayload {
  const payloadID = pl.readUInt8(0);
  const amount = pl.subarray(1, 32);
  const tokenAddress = Buffer.from(pl.subarray(33, 32)).toString("hex");
  const tokenChain = pl.readUInt16BE(65);
  const to = pl.subarray(67, 32);
  const toChain = pl.readUInt16BE(99);
  const fromAddress = pl.subarray(101, 32);
  const payload = pl.subarray(133);
  return {
    payloadID,
    amount,
    tokenAddress,
    tokenChain,
    to,
    toChain,
    fromAddress,
    payload,
  };
}

beforeAll(async () => {
  ethWalletAddress = await signer.getAddress();
  terraEmitterAddress = await getEmitterAddressTerra(
    CONTRACTS.DEVNET.terra2.token_bridge
  );
});

afterAll(async () => {
  provider.destroy();
});

async function waitForTerraExecution(
  transaction: string,
  lcd: LCDClient
): Promise<TxInfo | undefined> {
  let done: boolean = false;
  let info;
  while (!done) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    try {
      info = await lcd.tx.txInfo(transaction);
      if (info) {
        done = true;
      }
    } catch (e) {
      console.error(e);
    }
  }
  if (info && info.code !== 0) {
    // error code
    throw new Error(
      `Tx ${transaction}: error code ${info.code}: ${info.raw_log}`
    );
  }
  return info;
}

async function waitForWormExecution(
  transaction: string,
  lcd: LCDClient
): Promise<TxInfo | undefined> {
  let done: boolean = false;
  let info;
  while (!done) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    try {
      info = await wlcd.tx.txInfo(transaction);
      if (info) {
        done = true;
      }
    } catch (e) {
      console.error(e);
    }
  }
  if (info && info.code !== 0) {
    // error code
    throw new Error(
      `Tx ${transaction}: error code ${info.code}: ${info.raw_log}`
    );
  }
  return info;
}

async function getSignedVAABySequence(
  chainId: ChainId,
  sequence: string,
  emitterAddress: string
): Promise<Uint8Array> {
  //Note, if handed a sequence which doesn't exist or was skipped for consensus this will retry until the timeout.
  const { vaaBytes } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    chainId,
    emitterAddress,
    sequence,
    {
      transport: NodeHttpTransport(), //This should only be needed when running in node.
    },
    1000, //retryTimeout
    1000 //Maximum retry attempts
  );

  return vaaBytes;
}

const terraBroadcastAndWaitForExecution = async (
  msgs: Msg[],
  wallet: Wallet,
  isClassic = false
) => {
  const tx = await wallet.createAndSignTx({
    msgs,
  });
  const _lcd = isClassic ? lcdClassic : lcd;
  const txResult = await _lcd.tx.broadcast(tx);
  if (isTxError(txResult)) {
    throw new Error("tx error");
  }
  const txInfo = await waitForTerraExecution(txResult.txhash, _lcd);
  if (!txInfo) {
    throw new Error("tx info not found");
  }
  return txInfo;
};

const terraBroadcastTxAndGetSignedVaa = async (msgs: Msg[], wallet: Wallet) => {
  const txInfo = await terraBroadcastAndWaitForExecution(msgs, wallet);
  const txSequence = parseSequenceFromLogTerra(txInfo);
  if (!txSequence) {
    throw new Error("tx sequence not found");
  }
  return await getSignedVAABySequence(
    CHAIN_ID_TERRA2,
    txSequence,
    terraEmitterAddress
  );
};

const wormBroadcastAndWaitForExecution = async (
  msgs: Msg[],
  wallet: Wallet
) => {
  console.log("createAndSignTx...");
  const tx = await wallet.createAndSignTx({
    msgs,
  });
  console.log("broadcast...");
  const txResult = await wlcd.tx.broadcast(tx);
  console.log("txResult", txResult);
  if (isTxError(txResult)) {
    throw new Error("tx error");
  }
  console.log("waitForWormExecution...");
  const txInfo = await waitForWormExecution(txResult.txhash, wlcd);
  if (!txInfo) {
    throw new Error("tx info not found");
  }
  return txInfo;
};

const ethParseLogAndGetSignedVaa = async (receipt: ethers.ContractReceipt) => {
  const sequence = parseSequenceFromLogEth(
    receipt,
    CONTRACTS.DEVNET.ethereum.core
  );
  return await getSignedVAABySequence(
    CHAIN_ID_ETH,
    sequence,
    ethEmitterAddress
  );
};

const createWrappedOnWorm = async (
  tbAddr: string,
  wallet: string,
  signedVAA: Uint8Array
) => {
  const execute_msg = {
    submit_vaa: {
      data: fromUint8Array(signedVAA),
    },
  };

  const transaction = new MsgExecuteContract(wallet, tbAddr, execute_msg, {
    uworm: 1000,
  });
  return transaction;
};

describe("Crosschain Tests", () => {
  test.skip("Attest Eth on Wormchain", async () => {
    console.log("attestFromEth...");
    const attestReceipt = await attestFromEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      signer,
      TEST_ERC20
    );
    console.log("getting signed vaa...");
    const attestSignedVaa = await ethParseLogAndGetSignedVaa(attestReceipt);
    console.log(
      "attestSignedVaa",
      Buffer.from(attestSignedVaa).toString("hex")
    );
    console.log("createWrappedOnTerra...");
    // const createWrappedMsg = await createWrappedOnTerra(
    //   // CONTRACTS.DEVNET.terra2.token_bridge,
    //   CROSS_CHAIN_TOKEN_BRIDGE_ADDRESS,
    //   terraWalletAddress,
    //   attestSignedVaa
    // );
    // console.log("createWrappedOnTerra done", createWrappedMsg);
    const raw_msg = {
      submit_vaa: {
        data: fromUint8Array(attestSignedVaa),
      },
    };

    // Set up signer, wallet, etc.
    const addressPrefix = "wormhole";
    const denom = devnetConsts.chains[WORMCHAIN_ID].addresses.native.denom;
    const mnemonic =
      devnetConsts.chains[WORMCHAIN_ID].accounts.wormchainNodeOfGuardian0
        .mnemonic;
    const w = await Secp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: addressPrefix,
    });
    const gas = GasPrice.fromString(`0${denom}`);
    let host = devnetConsts.chains[3104].tendermintUrlLocal;
    if (os.hostname().includes("wormchain-deploy")) {
      // running in tilt devnet
      host = devnetConsts.chains[3104].tendermintUrlTilt;
    }
    let cwc = await SigningCosmWasmClient.connectWithSigner(host, w, {
      prefix: addressPrefix,
      gasPrice: gas,
    });
    const signers = await w.getAccounts();
    const cw_signer = signers[0].address;
    console.log("cw_signer", cw_signer);
    // End setup

    try {
      const inst = await cwc.execute(
        cw_signer,
        CROSS_CHAIN_TOKEN_BRIDGE_ADDRESS,
        raw_msg,
        "auto"
      );
      console.log("Result inst:", inst);
    } catch (e) {
      console.error("Error:", e);
    }
  });

  test("Create Payload 3 VAA and send to crosschain", async () => {
    console.log("Calling approveEth...");
    await approveEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      TEST_ERC20,
      signer,
      ethTransferAmount
    );
    console.log("Calling transferFromEth...");
    const payload: string = "terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v";
    const btPayload: BasicTransfer = {
      basic_transfer: {
        chain_id: CHAIN_ID_WORMCHAIN,
        recipient: payload,
      },
    };
    console.log(
      "Payload stuff:",
      JSON.stringify(btPayload),
      Buffer.from(JSON.stringify(btPayload))
    );
    const transferReceipt = await transferFromEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      signer,
      TEST_ERC20,
      ethTransferAmount,
      CHAIN_ID_WORMCHAIN,
      tryNativeToUint8Array(CROSS_CHAIN_ADDRESS, CHAIN_ID_WORMCHAIN),
      ethTransferAmount,
      null,
      Buffer.from(JSON.stringify(btPayload))
    );
    console.log("transferReceipt", transferReceipt);
    const transferSignedVaa = await ethParseLogAndGetSignedVaa(transferReceipt);
    console.log(
      "transferSignedVaa",
      Buffer.from(transferSignedVaa).toString("hex")
    );
    // Send VAA to crosschain contract
    const parsed = parseVaa(transferSignedVaa);
    console.log("parsed", parsed);
    const pl = parsePayload(parsed.payload);
    console.log("pl", pl);

    // complete_transfer_and_convert
    const raw_msg = {
      complete_transfer_and_convert: {
        vaa: fromUint8Array(transferSignedVaa),
      },
    };

    // Set up signer, wallet, etc.
    const addressPrefix = "wormhole";
    const denom = devnetConsts.chains[WORMCHAIN_ID].addresses.native.denom;
    const mnemonic =
      devnetConsts.chains[WORMCHAIN_ID].accounts.wormchainNodeOfGuardian0
        .mnemonic;
    const w = await Secp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: addressPrefix,
    });
    const gas = GasPrice.fromString(`0${denom}`);
    let host = devnetConsts.chains[3104].tendermintUrlLocal;
    if (os.hostname().includes("wormchain-deploy")) {
      // running in tilt devnet
      host = devnetConsts.chains[3104].tendermintUrlTilt;
    }
    let cwc = await SigningCosmWasmClient.connectWithSigner(host, w, {
      prefix: addressPrefix,
      gasPrice: gas,
    });
    const signers = await w.getAccounts();
    const cw_signer = signers[0].address;
    console.log("cw_signer", cw_signer);
    // End setup

    try {
      const inst = await cwc.execute(
        cw_signer,
        CROSS_CHAIN_ADDRESS,
        raw_msg,
        "auto"
      );
      console.log("Result inst:", inst);
      const numEvents: number = inst.events.length;
      for (let i = 0; i < numEvents; i++) {
        console.log("Event:", inst.events[i]);
        if (inst.events[i].type === "wasm") {
          for (let j = 0; j < inst.events[i].attributes.length; j++) {
            if (inst.events[i].attributes[j].key === "recipient") {
              const recipient = Buffer.from(
                inst.events[i].attributes[j].value,
                "base64"
              ).toString("hex");
              console.log("Recipient:", recipient);
            }
          }
        }
      }
    } catch (e) {
      console.error("Error:", e);
    }
  });

  test.skip("Attest and transfer token from Ethereum to Wormchain", async () => {
    // Attest
    console.log("Calling attestFromEth...");
    const attestReceipt = await attestFromEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      signer,
      TEST_ERC20
    );
    console.log("Calling ethParseLogAndGetSignedVaa...");
    const attestSignedVaa = await ethParseLogAndGetSignedVaa(attestReceipt);
    console.log("Calling createWrappedOnTerra...");
    const createWrappedMsg = await createWrappedOnWorm(
      // CONTRACTS.DEVNET.terra2.token_bridge,
      CROSS_CHAIN_ADDRESS,
      terraWalletAddress,
      attestSignedVaa
    );
    console.log("This is the createWrappedMsg", createWrappedMsg);
    let host = devnetConsts.chains[3104].tendermintUrlLocal;
    if (os.hostname().includes("wormchain-deploy")) {
      // running in tilt devnet
      host = devnetConsts.chains[3104].tendermintUrlTilt;
    }
    const denom = devnetConsts.chains[WORMCHAIN_ID].addresses.native.denom;
    const mnemonic =
      devnetConsts.chains[WORMCHAIN_ID].accounts.wormchainNodeOfGuardian0
        .mnemonic;
    const addressPrefix = "wormhole";
    const signerPk = devnetConsts.devnetGuardians[0].private;
    const accountingAddress =
      devnetConsts.chains[WORMCHAIN_ID].contracts.accountingNativeAddress;

    const w = await Secp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: addressPrefix,
    });

    const gas = GasPrice.fromString(`0${denom}`);
    let cwc = await SigningCosmWasmClient.connectWithSigner(host, w, {
      prefix: addressPrefix,
      gasPrice: gas,
    });

    // there is no danger here, just several Cosmos chains in devnet, so check for config issues
    let id = await cwc.getChainId();
    console.log("id:", id);
    if (id !== "wormchain") {
      throw new Error(
        `Wormchain CosmWasmClient connection produced an unexpected chainID: ${id}`
      );
    }

    const signers = await w.getAccounts();
    const cw_signer = signers[0].address;
    console.log("wormchain wallet pubkey: ", cw_signer);
    const raw_msg = {
      submit_vaa: {
        data: fromUint8Array(attestSignedVaa),
      },
    };
    const exec_msg = {
      execute_msg: {
        submit_vaa: {
          data: fromUint8Array(attestSignedVaa),
        },
      },
    };
    let inst;
    try {
      inst = await cwc.execute(
        cw_signer,
        CROSS_CHAIN_ADDRESS,
        // exec_msg,
        raw_msg,
        // createWrappedMsg,
        "auto"
      );
      console.log("Result inst:", inst);
    } catch (e) {
      console.error("Error:", e);
    }
    let txHash = inst.transactionHash;
    console.log(`executed submit_observation! txHash: ${txHash}`);

    console.log("Calling terraBroadcastAndWaitForExecution...");
    await terraBroadcastAndWaitForExecution([createWrappedMsg], terraWallet);
    // Transfer
    console.log("Calling approveEth...");
    await approveEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      TEST_ERC20,
      signer,
      ethTransferAmount
    );
    console.log("Calling transferFromEth...");
    const transferReceipt = await transferFromEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      signer,
      TEST_ERC20,
      ethTransferAmount,
      CHAIN_ID_TERRA2,
      tryNativeToUint8Array(terraWalletAddress, CHAIN_ID_TERRA2)
    );
    const transferSignedVaa = await ethParseLogAndGetSignedVaa(transferReceipt);
    const redeemMsg = await redeemOnTerra(
      CONTRACTS.DEVNET.terra2.token_bridge,
      terraWalletAddress,
      transferSignedVaa
    );
    expect(
      await getIsTransferCompletedTerra2(
        CONTRACTS.DEVNET.terra2.token_bridge,
        transferSignedVaa,
        lcd
      )
    ).toBe(false);
    await terraBroadcastAndWaitForExecution([redeemMsg], terraWallet);
    expect(
      await getIsTransferCompletedTerra2(
        CONTRACTS.DEVNET.terra2.token_bridge,
        transferSignedVaa,
        lcd
      )
    ).toBe(true);
  });

  test.skip("Attest and transfer token from Ethereum to Terra2", async () => {
    // Attest
    console.log("attestFromEth...");
    const attestReceipt = await attestFromEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      signer,
      TEST_ERC20
    );
    console.log("getting signed vaa...");
    const attestSignedVaa = await ethParseLogAndGetSignedVaa(attestReceipt);
    console.log("createWrappedOnTerra...");
    const createWrappedMsg = await createWrappedOnTerra(
      // CONTRACTS.DEVNET.terra2.token_bridge,
      CROSS_CHAIN_TOKEN_BRIDGE_ADDRESS,
      terraWalletAddress,
      attestSignedVaa
    );
    console.log("createWrappedOnTerra done", createWrappedMsg);
    await wormBroadcastAndWaitForExecution([createWrappedMsg], terraWallet);
    console.log("wormBroadcastAndWaitForExecution done");
    // Transfer
    // console.log("approveEth...");
    // await approveEth(
    //   CONTRACTS.DEVNET.ethereum.token_bridge,
    //   TEST_ERC20,
    //   signer,
    //   ethTransferAmount
    // );
    // const transferReceipt = await transferFromEth(
    //   CONTRACTS.DEVNET.ethereum.token_bridge,
    //   signer,
    //   TEST_ERC20,
    //   ethTransferAmount,
    //   CHAIN_ID_TERRA2,
    //   tryNativeToUint8Array(terraWalletAddress, CHAIN_ID_TERRA2)
    // );
    // const transferSignedVaa = await ethParseLogAndGetSignedVaa(transferReceipt);
    // const redeemMsg = await redeemOnTerra(
    //   CONTRACTS.DEVNET.terra2.token_bridge,
    //   terraWalletAddress,
    //   transferSignedVaa
    // );
    // expect(
    //   await getIsTransferCompletedTerra2(
    //     CONTRACTS.DEVNET.terra2.token_bridge,
    //     transferSignedVaa,
    //     lcd
    //   )
    // ).toBe(false);
    // await terraBroadcastAndWaitForExecution([redeemMsg], terraWallet);
    // expect(
    //   await getIsTransferCompletedTerra2(
    //     CONTRACTS.DEVNET.terra2.token_bridge,
    //     transferSignedVaa,
    //     lcd
    //   )
    // ).toBe(true);
  });

  test.skip("Test Parse VAA", async () => {
    const Payload3VAA =
      "0100000000010007e1a3ff6398cad8da78d1c7b402258c73a39682dd0523e84d846d1b36fb1de14b621366ea9402d62b901fc6453cf4e4fd6f80f7869119003e50e8ad88ce0cab01000015a81e56010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000001030000000000000000000000000000000000000000000000000000000005f5e1000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002da7415da106b53b332ed33f38bbecf8d961d592525a851e67de290cbae3302900c2000000000000000000000000022d491bde2303f2f43325b2108d26f1eaba1e32b7b2262617369635f7472616e73666572223a7b22636861696e5f6964223a333130342c22726563697069656e74223a2274657272613178343672716179346433637373713867787876717a387874366e776c7a34746432306b333876227d7d";
    // need a VAA with a payload 3 payload
    // Run it through the parser
    // check the results
  });
});
