import {
  getNetworkInfo,
  Network as InjectiveNetwork,
} from "@injectivelabs/networks";
import {
  ChainGrpcWasmApi,
  ChainRestAuthApi,
  createTransaction,
  MsgExecuteContractCompat,
  Msgs,
  PrivateKey,
  TxGrpcApi,
} from "@injectivelabs/sdk-ts";
import { DEFAULT_STD_FEE, getStdFee } from "@injectivelabs/utils";
import { fromUint8Array } from "js-base64";
import { NETWORKS } from "./consts";
import { impossible, Payload } from "./vaa";
import { transferFromInjective } from "@certusone/wormhole-sdk/lib/esm/token_bridge/injective";
import {
  Chain,
  chainToChainId,
  contracts,
  Network,
} from "@wormhole-foundation/sdk-base";
import { chains } from "@wormhole-foundation/sdk";
import { tryNativeToUint8Array } from "./sdk/array";

export async function execute_injective(
  payload: Payload,
  vaa: Buffer,
  network: Network
) {
  if (network === "Devnet") {
    throw new Error("Injective is not supported in DEVNET");
  }
  const chain = "Injective";
  let { key } = NETWORKS[network][chain];
  if (!key) {
    throw Error(`No ${network} key defined for Injective`);
  }

  const endPoint =
    network === "Mainnet"
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
      target_contract = contracts.coreBridge(network, "Injective");
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
      const nftContract = contracts.nftBridge.get(network, "Injective");
      if (!nftContract) {
        // NOTE: this code can safely be removed once the injective NFT bridge is
        // released, but it's fine for it to stay, as the condition will just be
        // skipped once 'contracts.nft_bridge' is defined
        throw new Error("NFT bridge not supported yet for injective");
      }

      target_contract = nftContract;
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
      const tbContract = contracts.tokenBridge.get(network, "Injective");
      if (!tbContract) {
        throw new Error("contracts.token_bridge is undefined");
      }

      target_contract = tbContract;
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
    case "WormholeRelayer":
      throw Error("Wormhole Relayer not supported on Injective");
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

  await signAndSendTx(walletPK, network, transaction);
}

export async function transferInjective(
  dstChain: Chain,
  dstAddress: string,
  tokenAddress: string,
  amount: string,
  network: Network,
  rpc: string
) {
  if (network === "Devnet") {
    throw new Error("Injective is not supported in DEVNET");
  }
  const chain = "Injective";
  const { key } = NETWORKS[network][chain];
  if (!key) {
    throw Error(`No ${network} key defined for Injective`);
  }
  const token_bridge = contracts.tokenBridge.get(network, "Injective");
  if (token_bridge == undefined) {
    throw Error(`Unknown token bridge contract on ${network} for ${chain}`);
  }

  const walletPK = PrivateKey.fromMnemonic(key);
  const walletInjAddr = walletPK.toBech32();

  const msgs = await transferFromInjective(
    walletInjAddr,
    token_bridge,
    tokenAddress,
    amount,
    chainToChainId(dstChain),
    tryNativeToUint8Array(dstAddress, chainToChainId(dstChain))
  );

  await signAndSendTx(walletPK, network, msgs);
}

async function signAndSendTx(
  walletPK: PrivateKey,
  network: string,
  msgs: Msgs | Msgs[]
) {
  const endPoint =
    network === "MAINNET"
      ? InjectiveNetwork.MainnetK8s
      : InjectiveNetwork.TestnetK8s;
  const networkInfo = getNetworkInfo(endPoint);
  const walletPublicKey = walletPK.toPublicKey().toBase64();
  const accountDetails = await new ChainRestAuthApi(
    networkInfo.rest
  ).fetchAccount(walletPK.toBech32());
  const { signBytes, txRaw } = createTransaction({
    message: msgs,
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
  const chain = "Injective";
  const n = NETWORKS[network][chain];

  let targetContract: string | undefined;

  switch (module) {
    case "TokenBridge":
      targetContract = contracts.tokenBridge.get(network, "Injective");
      break;
    case "NFTBridge":
      targetContract = contracts.nftBridge.get(network, "Injective");
      break;
    default:
      throw new Error(`Invalid module: ${module}`);
  }

  if (targetContract === undefined) {
    throw new Error(`Contract for ${module} on ${network} does not exist`);
  }

  if (n === undefined || n.rpc === undefined) {
    throw new Error(`RPC for ${module} on ${network} does not exist`);
  }

  const client = new ChainGrpcWasmApi(n.rpc);

  // Query the bridge registration for all the chains in parallel.
  const registrations: (any | null)[][] = await Promise.all(
    chains
      .filter((cname) => cname !== chain)
      .map(async (cname) => [
        cname,
        await (async () => {
          let query_msg = {
            chain_registration: {
              chain: chainToChainId(cname),
            },
          };

          let result = null;
          try {
            result = await client.fetchSmartContractState(
              targetContract as string,
              Buffer.from(JSON.stringify(query_msg)).toString("base64")
            );
          } catch {
            // Not logging anything because a chain not registered returns an error.
          }

          return result;
        })(),
      ])
  );

  const results: { [key: string]: string } = {};
  for (let [cname, queryResponse] of registrations) {
    if (queryResponse) {
      results[cname] = Buffer.from(queryResponse.address, "base64").toString(
        "hex"
      );
    }
  }
  console.log(results);
}
