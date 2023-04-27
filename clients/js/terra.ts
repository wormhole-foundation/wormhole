import {
  Coin,
  Fee,
  LCDClient,
  MnemonicKey,
  MsgExecuteContract,
} from "@terra-money/terra.js";
import { fromUint8Array } from "js-base64";
import { impossible, Payload } from "./vaa";
import { NETWORKS } from "./networks";
import axios from "axios";
import { CHAINS, CONTRACTS, TerraChainName } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

export async function execute_terra(
  payload: Payload,
  vaa: Buffer,
  network: "MAINNET" | "TESTNET" | "DEVNET",
  chain: TerraChainName
) {
  let n = NETWORKS[network][chain];
  let contracts = CONTRACTS[network][chain];

  const terra = new LCDClient({
    URL: n.rpc,
    chainID: n.chain_id,
    isClassic: chain === "terra",
  });

  const wallet = terra.wallet(
    new MnemonicKey({
      mnemonic: n.key,
    })
  );

  let target_contract: string;
  let execute_msg: object;

  switch (payload.module) {
    case "Core":
      target_contract = contracts.core;
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
          throw new Error("RecoverChainId not supported on terra")
        default:
          impossible(payload);
      }
      break;
    case "NFTBridge":
      if (contracts.nft_bridge === undefined) {
        // NOTE: this code can safely be removed once the terra NFT bridge is
        // released, but it's fine for it to stay, as the condition will just be
        // skipped once 'contracts.nft_bridge' is defined
        throw new Error("NFT bridge not supported yet for terra");
      }
      target_contract = contracts.nft_bridge;
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
          throw new Error("RecoverChainId not supported on terra")
        case "RegisterChain":
          console.log("Registering chain");
          break;
        case "Transfer":
          console.log("Completing transfer");
          break;
        default:
          impossible(payload);
      }
      break;
    case "TokenBridge":
      target_contract = contracts.token_bridge;
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
          throw new Error("RecoverChainId not supported on terra")
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
          break;
      }
      break;
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

  const feeDenoms = ["uluna"];

  const gasPrices = await axios
    .get("https://fcd.terra.dev/v1/txs/gas_prices")
    .then((result) => result.data);

  const feeEstimate = await terra.tx.estimateFee(
    [
      {
        sequenceNumber: await wallet.sequence(),
        publicKey: wallet.key.publicKey,
      },
    ],
    {
      msgs: [transaction],
      memo: "",
      feeDenoms,
      gasPrices,
    }
  );

  wallet
    .createAndSignTx({
      msgs: [transaction],
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

export async function query_registrations_terra(
  network: "MAINNET" | "TESTNET" | "DEVNET",
  chain: string,
  module: "Core" | "NFTBridge" | "TokenBridge",
): Promise<Object> {
  let n = NETWORKS[network][chain];
  let contracts = CONTRACTS[network][chain];

  let target_contract: string;

  switch (module) {
    case "TokenBridge":
      target_contract = contracts.token_bridge;
      break;
    case "NFTBridge":
      target_contract = contracts.nft_bridge;
      break;
    default:
      throw new Error(`Invalid module: ${module}`);
  }
  
  if (!target_contract) {
    throw new Error(`Contract for ${module} on ${network} does not exist`);
  }

  console.log(`Querying the ${module} on ${network} ${chain} for registered chains.`);

  const client = new LCDClient({
    URL: n.rpc,
    chainID: n.chain_id,
    isClassic: chain === "terra",
  });

  // Query the bridge registration for all the chains in parallel.
  const registrationsPromise = Promise.all(
    Object.entries(CHAINS)
      .filter(([c_name, _]) => c_name !== chain && c_name !== "unset")
      .map(async ([c_name, c_id]) => [c_name, await (async () => {
          let query_msg = {
            chain_registration: {
              chain: c_id,
            },
          };

          let result = null;
          try {
            result = await client.wasm.contractQuery(target_contract, query_msg);
          } catch {
            // Not logging anything because a chain not registered returns an error.
          }
        
          return result;
        })()
      ])
  )

  const registrations = await registrationsPromise;

  let results = {}
  for (let [c_name, queryResponse] of registrations) {
    if (queryResponse) {
        results[c_name] = `0x` + Buffer.from(queryResponse.address, 'base64').toString('hex');
    }
  }
  return results;
}
