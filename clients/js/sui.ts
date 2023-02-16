import { Ed25519Keypair, JsonRpcProvider, RawSigner } from "@mysten/sui.js";
import { execSync } from "child_process";
import { NETWORKS } from "./networks";

type Network = "MAINNET" | "TESTNET" | "DEVNET";

export function loadSigner(network: Network, rpc: string | undefined) {
  let private_key_str_base_64: string | undefined =
    NETWORKS[network]["sui"].key;
  if (private_key_str_base_64 === undefined) {
    throw new Error("No key for Sui");
  }
  let priv_key_bytes = new Uint8Array(
    Buffer.from(private_key_str_base_64, "base64")
  );
  let keypair = Ed25519Keypair.fromSeed(priv_key_bytes.slice(1));
  if (typeof rpc != "undefined") {
    rpc = NETWORKS[network]["sui"].rpc;
  }
  let provider = new JsonRpcProvider(rpc);
  const signer = new RawSigner(keypair, provider);
  return signer;
}

export async function publishPackage(
  network: Network,
  rpc: string | undefined,
  packagePath: string
) {
  console.log("publish package network: ", network);
  console.log("publish package rpc: ", rpc);
  console.log("package path is: ", packagePath);
  let signer = loadSigner(network, rpc);
  console.log("signer pubkey is: ", signer.getAddress());
  const compiledModules: string[] = JSON.parse(
    execSync(`sui move build --dump-bytecode-as-base64 --path ${packagePath}`, {
      encoding: "utf-8",
    })
  );
  console.log("here in pub package");
  console.log("compiled modules: ", compiledModules);
  const publishTxn = await signer.publish({
    compiledModules: compiledModules,
    gasBudget: 150000,
  });
  console.log("publishTxn", publishTxn);
  console.log(
    "effects: ",
    JSON.stringify(publishTxn["effects"]["effects"])
  );
  //console.log('publishTxn effects', publishTxn["EffectsCert"]["effects"]["effects"]);
}

export async function callEntryFunc(
  network: Network,
  rpc: string | undefined,
  packageObjectId: string,
  module: string,
  func: string,
  type_args: Array<string>,
  args: Array<any>
) {
  let signer = loadSigner(network, rpc);
  console.log("network: ", network);
  console.log("rpc: ", rpc);
  console.log("package object id: ", packageObjectId);
  console.log("module: ", module);
  const moveCallTxn = await signer.executeMoveCall({
    packageObjectId: packageObjectId,
    module: module,
    function: func,
    typeArguments: type_args,
    arguments: args,
    gasBudget: 50000,
  });
  console.log("moveCallTxn: ", moveCallTxn);
  console.log(
    "effects: ",
    JSON.stringify(moveCallTxn["effects"]["effects"])
  );
}
