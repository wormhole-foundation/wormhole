import {
  ChainId,
  ChainName,
  isCosmWasmChain,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

export async function getEmitterAddress(
  chain: ChainId | ChainName,
  addr: string
) {
  const emitter = require("@certusone/wormhole-sdk/lib/cjs/bridge/getEmitterAddress");
  if (chain === "solana" || chain === "pythnet") {
    // TODO: Create an isSolanaChain()
    addr = emitter.getEmitterAddressSolana(addr);
  } else if (isCosmWasmChain(chain)) {
    addr = await emitter.getEmitterAddressTerra(addr);
  } else if (chain === "algorand") {
    addr = emitter.getEmitterAddressAlgorand(BigInt(addr));
  } else if (chain === "near") {
    addr = emitter.getEmitterAddressNear(addr);
  } else if (chain === "aptos") {
    // TODO: There should be something in the SDK to do this.
    if (
      addr ===
      "0x576410486a2da45eee6c949c995670112ddf2fbeedab20350d506328eefc9d4f"
    ) {
      // Mainnet / Testnet TokenBridge
      addr = "0000000000000000000000000000000000000000000000000000000000000001";
    } else if (
      // Mainnet NFTBridge
      addr ===
      "0x1bdffae984043833ed7fe223f7af7a3f8902d04129b14f801823e64827da7130"
    ) {
      addr = "0000000000000000000000000000000000000000000000000000000000000005";
    } else {
      throw Error(`Unsupported Aptos address: ${addr}`);
    }
  } else if (chain === "sui") {
    // TODO: There should be something in the SDK to do this.
    if (
      addr ===
      "0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9"
    ) {
      // Mainnet TokenBridge
      addr = "ccceeb29348f71bdd22ffef43a2a19c1f5b5e17c5cca5411529120182672ade5";
    } else if (
      addr ===
      "0x32422cb2f929b6a4e3f81b4791ea11ac2af896b310f3d9442aa1fe924ce0bab4"
    ) {
      // Testnet TokenBridge
      addr =
        "0xb22cd218bb63da447ac2704c1cc72727df6b5e981ee17a22176fd7b84c114610";
    } else {
      throw Error(`Unsupported Sui address: ${addr}`);
    }
  } else {
    addr = emitter.getEmitterAddressEth(addr);
  }

  return addr;
}
