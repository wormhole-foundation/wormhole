import {
  Coin,
  Fee,
  LCDClient,
  MnemonicKey,
  MsgExecuteContract,
} from "@xpla/xpla.js";
import { fromUint8Array } from "js-base64";
import { impossible, Payload } from "./vaa";
import { NETWORKS } from "./networks";
import { CHAINS, CONTRACTS } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

export async function execute_xpla(
  payload: Payload,
  vaa: Buffer,
  network: "MAINNET" | "TESTNET" | "DEVNET"
) {
  const chain = "xpla";
  let n = NETWORKS[network][chain];
  let contracts = CONTRACTS[network][chain];

  const client = new LCDClient({
    URL: n.rpc,
    chainID: n.chain_id,
  });

  const wallet = client.wallet(
    new MnemonicKey({
      mnemonic: n.key,
    })
  );

  let target_contract: string;
  let execute_msg: object;

  switch (payload.module) {
    case "Core":
      target_contract = contracts.core;
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
          throw new Error("RecoverChainId not supported on XPLA")
        default:
          impossible(payload);
      }
      break;
    case "NFTBridge":
      if (contracts.nft_bridge === undefined) {
        // NOTE: this code can safely be removed once the terra NFT bridge is
        // released, but it's fine for it to stay, as the condition will just be
        // skipped once 'contracts.nft_bridge' is defined
        throw new Error("NFT bridge not supported yet for XPLA");
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
          throw new Error("RecoverChainId not supported on XPLA")
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
          throw new Error("RecoverChainId not supported on XPLA")
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
    { axpla: 1700000000000000000 }
  );

  const feeDenoms = ["axpla"];

  // const gasPrices = await axios
  //   .get("https://dimension-lcd.xpla.dev/v1/txs/gas_prices")
  //   .then((result) => result.data);

  const feeEstimate = await client.tx.estimateFee(
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
      // gasPrices,
    }
  );

  wallet
    .createAndSignTx({
      msgs: [transaction],
      memo: "",
      fee: new Fee(
        feeEstimate.gas_limit,
        feeEstimate.amount.add(new Coin("axpla", 18))
      ),
    })
    .then((tx) => client.tx.broadcast(tx))
    .then((result) => {
      console.log(result);
      console.log(`TX hash: ${result.txhash}`);
    });
}

export async function query_registrations_xpla(
  network: "MAINNET" | "TESTNET" | "DEVNET",
  module: "Core" | "NFTBridge" | "TokenBridge",
) {
  let chain = "xpla";
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
        results[c_name] = Buffer.from(queryResponse.address, 'base64').toString('hex');
    }
  }
  console.log(results);
}
