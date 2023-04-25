import {
  ChainId,
  ChainName,
  isCosmWasmChain,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

export async function getEmitterAddress(chain: ChainId | ChainName, addr: string) {
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
    if (addr === "0x576410486a2da45eee6c949c995670112ddf2fbeedab20350d506328eefc9d4f") {
      addr = "0000000000000000000000000000000000000000000000000000000000000001";
    } else if (addr === "0x1bdffae984043833ed7fe223f7af7a3f8902d04129b14f801823e64827da7130") {
      addr = "0000000000000000000000000000000000000000000000000000000000000005";
    } else {
      throw Error(`Unsupported Aptos address: ${addr}`);
    }
  } else {
    addr = emitter.getEmitterAddressEth(addr);
  }

  return addr;
}
