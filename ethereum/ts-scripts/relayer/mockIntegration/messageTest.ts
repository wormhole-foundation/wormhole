import {
  ChainInfo,
  getWormholeRelayer,
  getOperatingChains,
  init,
  loadChains,
} from "../helpers/env";
import { sendMessage } from "./messageUtils";

init();
const chains = getOperatingChains();

async function run() {
  console.log(process.argv);
  const fetchSignedVaa = !!process.argv.find(
    (arg) => arg === "--fetchSignedVaa"
  );
  const queryMessageOnTarget = !process.argv.find(
    (arg) => arg === "--noQueryMessageOnTarget"
  );
  console.log(chains);
  if (process.argv[2] === "--from" && process.argv[4] === "--to") {
    await sendMessage(
      getChainById(process.argv[3]),
      getChainById(process.argv[5]),
      fetchSignedVaa,
      queryMessageOnTarget
    );
  } else if (process.argv[4] === "--from" && process.argv[2] === "--to") {
    await sendMessage(
      getChainById(process.argv[5]),
      getChainById(process.argv[3]),
      fetchSignedVaa,
      queryMessageOnTarget
    );
  } else if (process.argv[2] === "--per-chain") {
    for (let i = 0; i < chains.length; ++i) {
      await sendMessage(
        chains[i],
        chains[i === 0 ? chains.length - 1 : 0],
        fetchSignedVaa,
        queryMessageOnTarget
      );
    }
  } else if (process.argv[2] === "--matrix") {
    for (let i = 0; i < chains.length; ++i) {
      for (let j = 0; i < chains.length; ++i) {
        await sendMessage(
          chains[i],
          chains[j],
          fetchSignedVaa,
          queryMessageOnTarget
        );
      }
    }
  } else {
    await sendMessage(
      chains[0],
      chains[1],
      fetchSignedVaa,
      queryMessageOnTarget
    );
  }
}

function getChainById(id: number | string): ChainInfo {
  id = Number(id);
  const chain = chains.find((c) => c.chainId === id);
  if (!chain) {
    throw new Error("chainId not found, " + id);
  }
  return chain;
}

console.log("Start!");
run().then(() => console.log("Done!"));
