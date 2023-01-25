import { getNetworkInfo, Network } from "@injectivelabs/networks";
import { getStdFee, DEFAULT_STD_FEE } from "@injectivelabs/utils";
import {
  PrivateKey,
  TxGrpcApi,
  ChainRestAuthApi,
  createTransaction,
  MsgExecuteContract,
} from "@injectivelabs/sdk-ts";
import { fromUint8Array } from "js-base64";
import { impossible, Payload } from "./vaa";
import { NETWORKS } from "./networks";
import { CONTRACTS } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

export async function execute_injective(
  payload: Payload,
  vaa: Buffer,
  environment: "MAINNET" | "TESTNET" | "DEVNET"
) {
  if (environment === "DEVNET") {
    throw new Error("Injective is not supported in DEVNET");
  }
  const chainName = "injective";
  let n = NETWORKS[environment][chainName];
  if (!n.key) {
    throw Error(`No ${environment} key defined for Injective`);
  }
  let contracts = CONTRACTS[environment][chainName];
  const endPoint =
    environment === "MAINNET" ? Network.MainnetK8s : Network.TestnetK8s;

  const network = getNetworkInfo(endPoint);
  const walletPKHash = n.key;
  const walletPK = PrivateKey.fromMnemonic(walletPKHash);
  const walletInjAddr = walletPK.toBech32();
  const walletPublicKey = walletPK.toPublicKey().toBase64();

  let target_contract: string;
  let action: string;
  let execute_msg: object;

  switch (payload.module) {
    case "Core":
      target_contract = contracts.core;
      action = "submit_v_a_a";
      execute_msg = {
        [action]: {
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
          throw new Error("RecoverChainId not supported on injective")
        default:
          impossible(payload);
      }
      break;
    case "NFTBridge":
      if (contracts.nft_bridge === undefined) {
        // NOTE: this code can safely be removed once the injective NFT bridge is
        // released, but it's fine for it to stay, as the condition will just be
        // skipped once 'contracts.nft_bridge' is defined
        throw new Error("NFT bridge not supported yet for injective");
      }
      target_contract = contracts.nft_bridge;
      action = "submit_vaa";
      execute_msg = {
        [action]: {
          data: fromUint8Array(vaa),
        },
      };
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on injective")
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
      console.log("contracts:", contracts);
      if (contracts.token_bridge === undefined) {
        throw new Error("contracts.token_bridge is undefined");
      }
      target_contract = contracts.token_bridge;
      action = "submit_vaa";
      execute_msg = {
        [action]: {
          data: fromUint8Array(vaa),
        },
      };
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on injective")
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

  console.log("execute_msg", execute_msg);
  const transaction = MsgExecuteContract.fromJSON({
    sender: walletInjAddr,
    contractAddress: target_contract,
    exec: {
      action,
      msg: {
        ...execute_msg[action],
      },
    },
  });
  console.log("transaction:", transaction);

  const accountDetails = await new ChainRestAuthApi(network.rest).fetchAccount(
    walletInjAddr
  );
  const { signBytes, txRaw } = createTransaction({
    message: transaction.toDirectSign(),
    memo: "",
    fee: getStdFee((parseInt(DEFAULT_STD_FEE.gas, 10) * 2.5).toString()),
    pubKey: walletPublicKey,
    sequence: parseInt(accountDetails.account.base_account.sequence, 10),
    accountNumber: parseInt(
      accountDetails.account.base_account.account_number,
      10
    ),
    chainId: network.chainId,
  });
  console.log("txRaw", txRaw);

  console.log("sign transaction...");
  /** Sign transaction */
  const sig = await walletPK.sign(Buffer.from(signBytes));

  /** Append Signatures */
  txRaw.setSignaturesList([sig]);

  const txService = new TxGrpcApi(network.grpc);

  console.log("simulate transaction...");
  /** Simulate transaction */
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

  console.log("broadcast transaction...");
  /** Broadcast transaction */
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
