require("dotenv").config({ path: ".env" });
const HDWalletProvider = require("@truffle/hdwallet-provider");

const MNEMONIC="myth like bonus scare over problem client lizard pioneer submit female collect";

module.exports = {
  networks: {  
    development: {
      provider: () =>
        new HDWalletProvider(
          MNEMONIC,
          "http://127.0.0.1:8545"
        ),
      network_id: '595', // mandala
      gas: 42032000,
      gasPrice: 200786445289, // storage_limit = 64001, validUntil = 360001, gasLimit = 10000000
      timeoutBlocks: 25,
      confirmations: 0,
    },
  },

  compilers: {
    solc: {
      version: "0.8.4",
      settings: {
        optimizer: {
          enabled: true,
          runs: 200,
        },
      },
    },
  },

  plugins: ["@chainsafe/truffle-plugin-abigen", "truffle-plugin-verify"],

  api_keys: {
    etherscan: process.env.ETHERSCAN_KEY,
  },
};
