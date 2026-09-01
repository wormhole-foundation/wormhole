import {
  Coin,
  Fee,
  LCDClient,
  MnemonicKey,
  MsgExecuteContract,
  Wallet,
} from "@terra-money/terra.js";
import axios from "axios";
import { fromUint8Array } from "js-base64";
import {
  Chain,
  Network,
  chains,
  toChainId,
} from "@wormhole-foundation/sdk-base";
import { transferFromTerra } from "@certusone/wormhole-sdk/lib/esm/token_bridge/transfer";
import { Payload, impossible } from "../../vaa";
import { toLegacyChainId, tryNativeToUint8Array } from "../../sdk/array";
import { TERRA2_CONNECTIONS, Terra2, terra2Contracts } from "./consts";

const GAS_PRICES_URL = "https://phoenix-fcd.terra.dev/v1/txs/gas_prices";

export const getTerra2Client = (network: Network, rpc?: string): LCDClient => {
  const connection = TERRA2_CONNECTIONS[network];
  const url = rpc ?? connection.rpc;
  if (!url) {
    throw new Error(`No ${network} rpc defined for Terra2`);
  }
  return new LCDClient({
    URL: url,
    chainID: connection.chain_id,
    isClassic: false,
  });
};

const getWallet = (terra: LCDClient, network: Network): Wallet => {
  const { key } = TERRA2_CONNECTIONS[network];
  if (!key) {
    throw new Error(`No ${network} key defined for Terra2`);
  }
  return terra.wallet(new MnemonicKey({ mnemonic: key }));
};

export async function execute_terra2(
  payload: Payload,
  vaa: Buffer,
  network: Network
): Promise<void> {
  const terra = getTerra2Client(network);
  const wallet = getWallet(terra, network);
  const { core, tokenBridge } = terra2Contracts(network);

  let target_contract: string;
  let execute_msg: object;

  switch (payload.module) {
    case "Core": {
      target_contract = core;
      // sigh...
      execute_msg = {
        submit_v_a_a: {
          vaa: fromUint8Array(vaa),
        },
      };
      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set");
          break;
        case "ContractUpgrade":
          console.log("Upgrading core contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on terra2");
        case "TransferFees":
          console.log("Transferring core bridge fees");
          break;
        default:
          impossible(payload);
      }

      break;
    }
    case "NFTBridge":
      throw new Error("NFT bridge not supported on terra2");
    case "TokenBridge": {
      target_contract = tokenBridge;
      execute_msg = {
        submit_vaa: {
          data: fromUint8Array(vaa),
        },
      };
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on terra2");
        case "RegisterChain":
          console.log("Registering chain");
          break;
        case "Transfer":
          console.log("Completing transfer");
          break;
        case "AttestMeta":
          console.log("Creating wrapped token");
          break;
        case "TransferWithPayload":
          throw Error("Can't complete payload 3 transfer from CLI");
        default:
          impossible(payload);
      }

      break;
    }
    case "WormholeRelayer":
      throw Error("Wormhole Relayer not supported on Terra2");
    default:
      target_contract = impossible(payload);
      execute_msg = impossible(payload);
  }

  const transaction = new MsgExecuteContract(
    wallet.key.accAddress,
    target_contract,
    execute_msg,
    { uluna: 1000 }
  );

  await signAndSendTx(terra, wallet, [transaction]);
}

export async function transferTerra2(
  dstChain: Chain | Terra2,
  dstAddress: string,
  tokenAddress: string,
  amount: string,
  network: Network,
  rpc: string
) {
  const terra = getTerra2Client(network, rpc);
  const wallet = getWallet(terra, network);
  const { tokenBridge } = terra2Contracts(network);

  const msgs = await transferFromTerra(
    wallet.key.accAddress,
    tokenBridge,
    tokenAddress,
    amount,
    toLegacyChainId(dstChain),
    tryNativeToUint8Array(dstAddress, dstChain)
  );
  await signAndSendTx(terra, wallet, msgs);
}

async function signAndSendTx(
  terra: LCDClient,
  wallet: Wallet,
  msgs: MsgExecuteContract[]
) {
  const feeDenoms = ["uluna"];
  // A devnet LCD has no FCD gas-price oracle; fall back to a sane default.
  const gasPrices = await axios
    .get(GAS_PRICES_URL)
    .then((result) => result.data)
    .catch(() => ({ uluna: "0.015" }));
  const feeEstimate = await terra.tx.estimateFee(
    [
      {
        sequenceNumber: await wallet.sequence(),
        publicKey: wallet.key.publicKey,
      },
    ],
    {
      msgs,
      memo: "",
      feeDenoms,
      gasPrices,
    }
  );

  return wallet
    .createAndSignTx({
      msgs,
      memo: "",
      fee: new Fee(
        feeEstimate.gas_limit,
        feeEstimate.amount.add(new Coin("uluna", 12))
      ),
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((result) => {
      console.log(result);
      console.log(`TX hash: ${result.txhash}`);
    });
}

export async function queryRegistrationsTerra2(
  network: Network,
  module: "Core" | "NFTBridge" | "TokenBridge"
): Promise<Object> {
  if (module !== "TokenBridge") {
    throw new Error(`Invalid module: ${module}`);
  }
  const { tokenBridge } = terra2Contracts(network);
  const client = getTerra2Client(network);

  // Query the bridge registration for all the chains in parallel.
  const registrations: (string | null)[][] = await Promise.all(
    chains.map(async (cname) => [
      cname,
      await (async () => {
        let query_msg = {
          chain_registration: {
            chain: toChainId(cname),
          },
        };

        let result = null;
        try {
          const resp: { address: string } = await client.wasm.contractQuery(
            tokenBridge,
            query_msg
          );
          if (resp) {
            result = resp.address;
          }
        } catch {
          // Not logging anything because a chain not registered returns an error.
        }

        return result;
      })(),
    ])
  );

  const results: { [key: string]: string } = {};
  for (let [cname, queryResponse] of registrations) {
    if (cname && queryResponse) {
      results[cname] = Buffer.from(queryResponse, "base64").toString("hex");
    }
  }
  return results;
}
