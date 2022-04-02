require("../helpers/loadConfig");
process.env.LOG_DIR = ".";

import { CHAIN_ID_TERRA } from "@certusone/wormhole-sdk";
import { jest, test } from "@jest/globals";
import { LCDClient } from "@terra-money/terra.js";
import { ChainConfigInfo } from "../configureEnv";
import { calcLocalAddressesTerra, pullTerraBalance } from "./walletMonitor";
// import { pullEVMBalance } from "./walletMonitor";

jest.setTimeout(300000);

// const bscChainConfig: ChainConfigInfo = {
//   chainId: CHAIN_ID_BSC,
//   chainName: "BSC",
//   nativeCurrencySymbol: "BNB",
//   nodeUrl: "https://bsc-dataseed.binance.org",
//   tokenBridgeAddress: "0xB6F6D86a8f9879A9c87f643768d9efc38c1Da6E7",
//   wrappedAsset: "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c",
// };
// const bscPublicKey = "0xB6F6D86a8f9879A9c87f643768d9efc38c1Da6E7"; // Token Bridge
// const bscTokens = [
//   "0xfA54fF1a158B5189Ebba6ae130CEd6bbd3aEA76e", // SOL
//   "0x4DB5a66E937A9F4473fA95b1cAF1d1E1D62E29EA", // WETH
//   "0x156ab3346823B651294766e23e6Cf87254d68962", // LUNA
//   "0x3d4350cD54aeF9f9b2C29435e0fa809957B3F30a", // UST
//   "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c", // WBNB
//   "0xc836d8dC361E44DbE64c4862D55BA041F88Ddd39", // WMATIC
//   "0x96412902aa9aFf61E13f085e70D3152C6ef2a817", // WAVAX
//   "0x6c6D604D3f07aBE287C1A3dF0281e999A83495C0", // wROSE
//   "0xbF8413EE8612E0E4f66Aa63B5ebE27f3C5883d47", // WFTM
//   "0xB04906e95AB5D797aDA81508115611fee694c2b3", // USDC
//   "0x524bC91Dc82d6b90EF29F76A3ECAaBAffFD490Bc", // USDT
// ];

// test("should pull EVM token balances", async () => {
//   for (let address of bscTokens) {
//     const balance = await pullEVMBalance(bscChainConfig, bscPublicKey, address);
//     console.log(balance);
//     expect(balance).toBeTruthy();
//   }
// });

const terraChainConfig: ChainConfigInfo = {
  chainId: CHAIN_ID_TERRA,
  chainName: "Terra",
  nativeCurrencySymbol: "UST",
  nodeUrl: "https://fcd.terra.dev",
  tokenBridgeAddress: "terra10nmmwe8r3g99a9newtqa7a75xfgs2e8z87r2sf",
  terraName: "mainnet",
  terraChainId: "columbus-5",
  terraCoin: "uluna",
  terraGasPriceUrl: "https://fcd.terra.dev/v1/txs/gas_prices",
};

const supportedTokens = require("../../config/mainnet/supportedTokens.json");

test("should pull Terra token balances", async () => {
  if (
    !(
      terraChainConfig.terraChainId &&
      terraChainConfig.terraCoin &&
      terraChainConfig.terraGasPriceUrl &&
      terraChainConfig.terraName
    )
  ) {
    throw new Error("Terra relay was called without proper instantiation.");
  }
  const lcdConfig = {
    URL: terraChainConfig.nodeUrl,
    chainID: terraChainConfig.terraChainId,
    name: terraChainConfig.terraName,
  };
  const lcd = new LCDClient(lcdConfig);
  const localAddresses = await calcLocalAddressesTerra(
    lcd,
    supportedTokens,
    terraChainConfig
  );
  expect(localAddresses.length).toBeGreaterThan(0);
  for (const tokenAddress of localAddresses) {
    const balance = await pullTerraBalance(
      lcd,
      terraChainConfig.tokenBridgeAddress,
      tokenAddress
    );
    console.log(balance);
    expect(balance).toBeDefined();
  }
});
