import {
  CHAINS,
  CONTRACTS,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import {
  getNetworkInfo,
  Network as InjectiveNetwork,
} from "@injectivelabs/networks";
import {
  ChainGrpcWasmApi,
  ChainRestAuthApi,
  createTransaction,
  MsgExecuteContractCompat,
  PrivateKey,
  TxGrpcApi,
} from "@injectivelabs/sdk-ts";
import { DEFAULT_STD_FEE, getStdFee } from "@injectivelabs/utils";
import { fromUint8Array } from "js-base64";
import { NETWORKS } from "./consts";
import { Network } from "./utils";
import { impossible, Payload } from "./vaa";

export async function execute_injective(
  payload: Payload,
  vaa: Buffer,
  network: Network
) {
  if (network === "DEVNET") {
    throw new Error("Injective is not supported in DEVNET");
  }
  const chain = "injective";
  let { key } = NETWORKS[network][chain];
  if (!key) {
    throw Error(`No ${network} key defined for Injective`);
  }

  let contracts = CONTRACTS[network][chain];
  const endPoint =
    network === "MAINNET"
      ? InjectiveNetwork.MainnetK8s
      : InjectiveNetwork.TestnetK8s;

  const networkInfo = getNetworkInfo(endPoint);
  const walletPK = PrivateKey.fromMnemonic(key);
  const walletInjAddr = walletPK.toBech32();
  const walletPublicKey = walletPK.toPublicKey().toBase64();

  let target_contract: string;
  let action: "submit_v_a_a" | "submit_vaa";
  let execute_msg: { vaa: string } | { data: string };

  switch (payload.module) {
    case "Core": {
      target_contract = contracts.core;
      action = "submit_v_a_a";
      execute_msg = {
        vaa: fromUint8Array(vaa),
      };
      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set");
          break;
        case "ContractUpgrade":
          console.log("Upgrading core contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on injective");
        default:
          impossible(payload);
      }

      break;
    }
    case "NFTBridge": {
      if (!contracts.nft_bridge) {
        // NOTE: this code can safely be removed once the injective NFT bridge is
        // released, but it's fine for it to stay, as the condition will just be
        // skipped once 'contracts.nft_bridge' is defined
        throw new Error("NFT bridge not supported yet for injective");
      }

      target_contract = contracts.nft_bridge;
      action = "submit_vaa";
      execute_msg = {
        data: fromUint8Array(vaa),
      };
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on injective");
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
    }
    case "TokenBridge": {
      console.log("contracts:", contracts);
      if (!contracts.token_bridge) {
        throw new Error("contracts.token_bridge is undefined");
      }

      target_contract = contracts.token_bridge;
      action = "submit_vaa";
      execute_msg = {
        data: fromUint8Array(vaa),
      };
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on injective");
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
    default:
      action = impossible(payload);
      target_contract = impossible(payload);
      execute_msg = impossible(payload);
  }

  console.log("execute_msg", { [action]: execute_msg });
  const transaction = MsgExecuteContractCompat.fromJSON({
    sender: walletInjAddr,
    contractAddress: target_contract,
    exec: {
      action,
      msg: {
        ...execute_msg,
      },
    },
  });
  console.log("transaction:", transaction);

  const accountDetails = await new ChainRestAuthApi(
    networkInfo.rest
  ).fetchAccount(walletInjAddr);
  const { signBytes, txRaw } = createTransaction({
    message: transaction,
    memo: "",
    fee: getStdFee((parseInt(DEFAULT_STD_FEE.gas, 10) * 2.5).toString()),
    pubKey: walletPublicKey,
    sequence: parseInt(accountDetails.account.base_account.sequence, 10),
    accountNumber: parseInt(
      accountDetails.account.base_account.account_number,
      10
    ),
    chainId: networkInfo.chainId,
  });
  console.log("txRaw", txRaw);

  // Sign transaction
  console.log("sign transaction...");
  const sig = await walletPK.sign(Buffer.from(signBytes));

  // Append Signatures
  txRaw.signatures = [sig];

  // Simulate transaction
  console.log("simulate transaction...");
  const txService = new TxGrpcApi(networkInfo.grpc);
  try {
    const simulationResponse = await txService.simulate(txRaw);
    console.log(
      `Transaction simulation response: ${JSON.stringify(
        simulationResponse.gasInfo
      )}`
    );
  } catch (e) {
    console.log("Failed to simulate:", e);
    return;
  }

  // Broadcast transaction
  console.log("broadcast transaction...");
  const txResponse = await txService.broadcast(txRaw);
  console.log("txResponse", txResponse);

  if (txResponse.code !== 0) {
    console.log(`Transaction failed: ${txResponse.rawLog}`);
  } else {
    console.log(
      `Broadcasted transaction hash: ${JSON.stringify(txResponse.txHash)}`
    );
  }
}

export async function queryRegistrationsInjective(
  network: Network,
  module: "Core" | "NFTBridge" | "TokenBridge"
) {
  let chain = "injective";
  let n = NETWORKS[network][chain];
  let contracts = CONTRACTS[network][chain];

  let targetContract: string;

  switch (module) {
    case "TokenBridge":
      targetContract = contracts.token_bridge;
      break;
    case "NFTBridge":
      targetContract = contracts.nft_bridge;
      break;
    default:
      throw new Error(`Invalid module: ${module}`);
  }

  if (!targetContract) {
    throw new Error(`Contract for ${module} on ${network} does not exist`);
  }

  const client = new ChainGrpcWasmApi(n.rpc);

  // Query the bridge registration for all the chains in parallel.
  const registrationsPromise = Promise.all(
    Object.entries(CHAINS)
      .filter(([c_name, _]) => c_name !== chain && c_name !== "unset")
      .map(async ([c_name, c_id]) => [
        c_name,
        await (async () => {
          let query_msg = {
            chain_registration: {
              chain: c_id,
            },
          };

          let result = null;
          try {
            result = await client.fetchSmartContractState(
              targetContract,
              Buffer.from(JSON.stringify(query_msg)).toString("base64")
            );
          } catch {
            // Not logging anything because a chain not registered returns an error.
          }

          return result;
        })(),
      ])
  );

  const registrations = await registrationsPromise;

  let results = {};
  for (let [c_name, queryResponse] of registrations) {
    if (queryResponse) {
      results[c_name] = Buffer.from(queryResponse.address, "base64").toString(
        "hex"
      );
    }
  }
  console.log(results);
}
