import {
  ChainId,
  ChainName,
  isCosmWasmChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";

import {
  getEmitterAddressAlgorand,
  getEmitterAddressEth,
  getEmitterAddressNear,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
} from "@certusone/wormhole-sdk/lib/esm/bridge/getEmitterAddress";

export async function getEmitterAddress(
  chain: ChainId | ChainName,
  addr: string
) {
  if (chain === "solana" || chain === "pythnet") {
    // TODO: Create an isSolanaChain()
    addr = getEmitterAddressSolana(addr);
  } else if (isCosmWasmChain(chain)) {
    addr = await getEmitterAddressTerra(addr);
  } else if (chain === "algorand") {
    addr = getEmitterAddressAlgorand(BigInt(addr));
  } else if (chain === "near") {
    addr = getEmitterAddressNear(addr);
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
      "0x6fb10cdb7aa299e9a4308752dadecb049ff55a892de92992a1edbd7912b3d6da"
    ) {
      // Testnet TokenBridge
      addr =
        "0x40440411a170b4842ae7dee4f4a7b7a58bc0a98566e998850a7bb87bf5dc05b9";
    } else {
      throw Error(`Unsupported Sui address: ${addr}`);
    }
  } else {
    addr = getEmitterAddressEth(addr);
  }

  return addr;
}
