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
import { NETWORKS } from "./consts";
import { Payload, impossible } from "./vaa";
import { transferFromTerra } from "@certusone/wormhole-sdk/lib/esm/token_bridge/transfer";
import { tryNativeToUint8Array } from "@certusone/wormhole-sdk/lib/esm/utils";
import { Chain, Network, contracts } from "@wormhole-foundation/sdk-base";

export async function execute_terra(
  payload: Payload,
  vaa: Buffer,
  network: Network,
  chain: "Terra" | "Terra2"
): Promise<void> {
  const { rpc, key, chain_id } = NETWORKS[network][chain];

  const terra = new LCDClient({
    URL: rpc,
    chainID: chain_id,
    isClassic: chain === "Terra",
  });

  const wallet = terra.wallet(
    new MnemonicKey({
      mnemonic: key,
    })
  );

  let target_contract: string;
  let execute_msg: object;

  switch (payload.module) {
    case "Core": {
      const coreContract = contracts.coreBridge(network, chain);
      if (!coreContract) {
        throw new Error(
          `Core bridge address not defined for ${chain} ${network}`
        );
      }

      target_contract = coreContract;
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
          throw new Error("RecoverChainId not supported on terra");
        default:
          impossible(payload);
      }

      break;
    }
    case "NFTBridge": {
      const nftContract = contracts.nftBridge.get(network, chain);
      if (!nftContract) {
        // NOTE: this code can safely be removed once the terra NFT bridge is
        // released, but it's fine for it to stay, as the condition will just be
        // skipped once 'contracts.nft_bridge' is defined
        throw new Error(`NFT bridge not supported yet for ${chain}`);
      }

      target_contract = nftContract;
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
          throw new Error("RecoverChainId not supported on terra");
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
      const tbContract = contracts.tokenBridge.get(network, chain);
      if (!tbContract) {
        throw new Error(
          `Token bridge address not defined for ${chain} ${network}`
        );
      }

      target_contract = tbContract;
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
          throw new Error("RecoverChainId not supported on terra");
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
      throw Error("Wormhole Relayer not supported on Terra");
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

export async function transferTerra(
  srcChain: "Terra" | "Terra2",
  dstChain: Chain,
  dstAddress: string,
  tokenAddress: string,
  amount: string,
  network: Network,
  rpc: string
) {
  const n = NETWORKS[network][srcChain];
  if (!n.key) {
    throw Error(`No ${network} key defined for ${srcChain} (see networks.ts)`);
  }
  const token_bridge = contracts.tokenBridge.get(network, srcChain);
  if (!token_bridge) {
    throw Error(`Unknown token bridge contract on ${network} for ${srcChain}`);
  }

  const terra = new LCDClient({
    URL: rpc,
    chainID: n.chain_id,
    isClassic: srcChain === "Terra",
  });

  const wallet = terra.wallet(
    new MnemonicKey({
      mnemonic: n.key,
    })
  );

  const msgs = await transferFromTerra(
    wallet.key.accAddress,
    token_bridge,
    tokenAddress,
    amount,
    dstChain,
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
  const gasPrices = await axios
    .get("https://terra-classic-fcd.publicnode.com/v1/txs/gas_prices")
    .then((result) => result.data);
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

export async function queryRegistrationsTerra(
  network: Network,
  chain: "Terra" | "Terra2",
  module: "Core" | "NFTBridge" | "TokenBridge"
): Promise<Object> {
  const n = NETWORKS[network][chain];

  let targetContract: string | undefined;

  switch (module) {
    case "TokenBridge":
      targetContract = contracts.tokenBridge(network, chain);
      break;
    case "NFTBridge":
      targetContract = contracts.nftBridge.get(network, chain);
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

  if (n === undefined || n.chain_id === undefined) {
    throw new Error(`Chain id for ${module} on ${network} does not exist`);
  }

  const client = new LCDClient({
    URL: n.rpc,
    chainID: n.chain_id,
    isClassic: chain === "Terra",
  });

  // Query the bridge registration for all the chains in parallel.
  const registrations: (string | null)[][] = await Promise.all(
    Object.entries(CHAINS)
      .filter(([cname, _]) => cname !== chain && cname !== "unset")
      .map(async ([cname, cid]) => [
        cname,
        await (async () => {
          let query_msg = {
            chain_registration: {
              chain: cid,
            },
          };

          let result = null;
          try {
            const resp: { address: string } = await client.wasm.contractQuery(
              targetContract as string,
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
