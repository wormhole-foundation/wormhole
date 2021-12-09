import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  getForeignAssetEth,
  nativeToHexString,
  hexToUint8Array,
  approveEth,
} from "@certusone/wormhole-sdk";
import { setDefaultWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import { getAddress } from "@ethersproject/address";
import { parseUnits } from "@ethersproject/units";
import { ethers } from "ethers";
import * as fs from "fs";
import { exit } from "process";
import { SimpleDex__factory } from "../ethers-contracts/abi/factories/SimpleDex__factory";
import {
  ETH_TEST_WALLET_PUBLIC_KEY,
  getSignerForChain,
  getTokenBridgeAddressForChain,
} from "./consts";
import {
  fullAttestation,
  basicTransfer,
} from "@certusone/wormhole-examples/lib/commonWorkflows";

setDefaultWasm("node");

//This script is reliant on core examples, and the wormhole SDK.
//It is meant to be run against a fresh devnet / tilt environment.
function getPrice(chain: ChainId) {
  if (chain === CHAIN_ID_ETH) {
    return 4400;
  }
  if (chain === CHAIN_ID_BSC) {
    return 630;
  }
}

async function main() {
  await configWormhole();

  const { ethAddress, wbnbOnEth, bscAddress, wethOnBsc } = await seedPools();
  console.log("Pools seeded");

  await createSwapPoolFile(ethAddress, wbnbOnEth, bscAddress, wethOnBsc);

  console.log("Job done");
  return Promise.resolve();
}

async function configWormhole() {
  const basisTransferAmount = "5000";
  const WETH = "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";

  console.log("Doing WETH Attest");
  await fullAttestation(CHAIN_ID_ETH, getAddress(WETH));
  console.log("Doing WBNB Attest");
  await fullAttestation(CHAIN_ID_BSC, getAddress(WETH));

  console.log("Bridging over WETH to bsc");
  await basicTransfer(
    CHAIN_ID_ETH,
    basisTransferAmount,
    CHAIN_ID_BSC,
    ETH_TEST_WALLET_PUBLIC_KEY,
    ETH_TEST_WALLET_PUBLIC_KEY,
    true
  );
  console.log("Bridging over WBNB to eth");
  await basicTransfer(
    CHAIN_ID_BSC,
    basisTransferAmount,
    CHAIN_ID_ETH,
    ETH_TEST_WALLET_PUBLIC_KEY,
    ETH_TEST_WALLET_PUBLIC_KEY,
    true
  );
}

async function createSwapPoolFile(
  ethAddress: string,
  wbnbAddress: string,
  bscAddress: string,
  wethAddress: string
) {
  const literal: any = {
    [CHAIN_ID_ETH]: {
      [CHAIN_ID_BSC]: { poolAddress: ethAddress, tokenAddress: wbnbAddress },
    },
    [CHAIN_ID_BSC]: {
      [CHAIN_ID_ETH]: { poolAddress: bscAddress, tokenAddress: wethAddress },
    },
  };
  const content = JSON.stringify(literal);

  //TODO not this
  await fs.writeFileSync("../react/src/swapPools.json", content, {
    flag: "w+",
  });
}

//TODO, in a for loop for all the EVM chains
const seedPools = async () => {
  const ethSigner = getSignerForChain(CHAIN_ID_ETH);
  const bscSigner = getSignerForChain(CHAIN_ID_BSC);
  const currentEthPrice = getPrice(CHAIN_ID_ETH);
  const currentSolPrice = getPrice(CHAIN_ID_BSC);
  const ratio = currentEthPrice / currentSolPrice;
  const ethBasis = 500;
  const bnbBasis = Math.ceil(ethBasis * ratio);
  const WETH = "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";

  const wbnbOnEth = await getForeignAssetEth(
    getTokenBridgeAddressForChain(CHAIN_ID_ETH),
    ethSigner.provider as any,
    CHAIN_ID_BSC,
    hexToUint8Array(nativeToHexString(WETH, CHAIN_ID_BSC))
  );

  console.log("WBNB on ETH address", wbnbOnEth);

  const wethOnBsc = await getForeignAssetEth(
    getTokenBridgeAddressForChain(CHAIN_ID_BSC),
    bscSigner.provider as any,
    CHAIN_ID_ETH,
    hexToUint8Array(nativeToHexString(WETH, CHAIN_ID_ETH))
  );

  console.log("WETH on BSC address", wethOnBsc);

  console.log("about to deploy ETH contract");
  const contractInterface = SimpleDex__factory.createInterface();
  const bytecode = SimpleDex__factory.bytecode;
  const ethfactory = new ethers.ContractFactory(
    contractInterface,
    bytecode,
    ethSigner
  );
  const contract = await ethfactory.deploy(getAddress(wbnbOnEth));
  const ethAddress = await contract.deployed().then(
    (result) => {
      console.log("Successfully deployed contract at " + result.address);
      return result.address;
    },
    (error) => {
      console.error(error);
      exit(1);
    }
  );
  console.log("about to deploy bsc contract");
  const bscfactory = new ethers.ContractFactory(
    contractInterface,
    bytecode,
    bscSigner
  );
  const bscContract = await bscfactory.deploy(getAddress(wethOnBsc));
  const bscAddress = await bscContract.deployed().then(
    (result) => {
      console.log("Successfully deployed contract at " + result.address);
      return result.address;
    },
    (error) => {
      console.error(error);
      exit(1);
    }
  );

  console.log("Doing WBNB on ETH Approve");
  await approveEth(
    ethAddress,
    getAddress(wbnbOnEth),
    ethSigner,
    "10000000000000000000000"
  );
  console.log("Doing WETH on BSC Approve");
  await approveEth(
    bscAddress,
    getAddress(wethOnBsc),
    bscSigner,
    "10000000000000000000000"
  );

  const ethDex = SimpleDex__factory.connect(ethAddress, ethSigner);
  const bscDex = SimpleDex__factory.connect(bscAddress, bscSigner);

  console.log("Initializing eth pool");
  const ethInit = await ethDex.init(parseUnits(bnbBasis.toString(), 18), {
    value: parseUnits(ethBasis.toString(), 18),
    gasLimit: 500000,
  });
  await ethInit.wait();

  console.log("Initializing bsc pool");
  const bscInit = await bscDex.init(parseUnits(ethBasis.toString(), 18), {
    value: parseUnits(bnbBasis.toString(), 18),
    gasLimit: 500000,
  });
  await bscInit.wait();
  console.log("pools initialized");

  const ethLiq = await ethDex.totalLiquidity();
  console.log("Eth liquidity", ethLiq);

  const bscLiq = await bscDex.totalLiquidity();
  console.log("bsc liquidity", bscLiq);
  //Pool should now be seeded with a small amount of ETH and Wormhole-Wrapped SOL.
  return { ethAddress, wbnbOnEth, bscAddress, wethOnBsc };
};

let done = false;
main().then(
  () => (done = true),
  (error) => {
    console.error(error);
    done = true;
  }
);
function wait() {
  if (!done) {
    setTimeout(wait, 1000);
  } else {
    exit(0);
  }
}
wait();
