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
import { CONTRACTS, TerraChainName } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

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
