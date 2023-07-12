import {
  CONTRACTS,
  ChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import {
  Coin,
  Fee,
  LCDClient,
  MnemonicKey,
  MsgExecuteContract,
  Wallet,
} from "@xpla/xpla.js";
import { fromUint8Array } from "js-base64";
import { NETWORKS } from "./consts";
import { Network } from "./utils";
import { Payload, impossible } from "./vaa";
import { transferFromXpla } from "@certusone/wormhole-sdk/lib/esm/token_bridge/transfer";
import { tryNativeToUint8Array } from "@certusone/wormhole-sdk/lib/esm/utils";

export async function execute_xpla(
  payload: Payload,
  vaa: Buffer,
  network: Network
) {
  const { rpc, key, chain_id } = NETWORKS[network].xpla;
  const contracts = CONTRACTS[network].xpla;
  if (!key) {
    throw Error(`No ${network} key defined for XPLA`);
  }

  if (!rpc) {
    throw Error(`No ${network} rpc defined for XPLA`);
  }

  const client = new LCDClient({
    URL: rpc,
    chainID: chain_id,
  });

  const wallet = client.wallet(
    new MnemonicKey({
      mnemonic: key,
    })
  );

  let target_contract: string;
  let execute_msg: object;
  switch (payload.module) {
    case "Core": {
      if (!contracts.core) {
        throw new Error(`Core bridge address not defined for XPLA ${network}`);
      }

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
          throw new Error("RecoverChainId not supported on XPLA");
        default:
          impossible(payload);
      }

      break;
    }
    case "NFTBridge": {
      if (!contracts.nft_bridge) {
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
          throw new Error("RecoverChainId not supported on XPLA");
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
      if (!contracts.token_bridge) {
        throw new Error(`Token bridge address not defined for XPLA ${network}`);
      }

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
          throw new Error("RecoverChainId not supported on XPLA");
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
      throw Error("Wormhole Relayer not supported on Xpla");
    default:
      target_contract = impossible(payload);
      execute_msg = impossible(payload);
  }

  const transaction = new MsgExecuteContract(
    wallet.key.accAddress,
    target_contract,
    execute_msg,
    { axpla: "1700000000000000000" }
  );

  await signAndSendTx(client, wallet, [transaction]);
}

export async function transferXpla(
  dstChain: ChainName,
  dstAddress: string,
  tokenAddress: string,
  amount: string,
  network: Network,
  rpc: string
) {
  const { key, chain_id } = NETWORKS[network].xpla;
  if (!key) {
    throw Error(`No ${network} key defined for XPLA`);
  }
  const { token_bridge } = CONTRACTS[network].xpla;
  if (token_bridge == undefined) {
    throw Error(`Unknown token bridge contract on ${network} for XPLA`);
  }
  const client = new LCDClient({
    URL: rpc,
    chainID: chain_id,
  });
  const wallet = client.wallet(
    new MnemonicKey({
      mnemonic: key,
    })
  );
  const msgs = transferFromXpla(
    wallet.key.accAddress,
    token_bridge,
    tokenAddress,
    amount,
    dstChain,
    tryNativeToUint8Array(dstAddress, dstChain)
  );
  await signAndSendTx(client, wallet, msgs);
}

async function signAndSendTx(
  client: LCDClient,
  wallet: Wallet,
  msgs: MsgExecuteContract[]
) {
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
      msgs,
      memo: "",
      feeDenoms,
      // gasPrices,
    }
  );

  wallet
    .createAndSignTx({
      msgs,
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
