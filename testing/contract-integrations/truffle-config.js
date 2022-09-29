const HDWalletProvider = require("@truffle/hdwallet-provider");

module.exports = {
  networks: {
    development: {
      host: "127.0.0.1",
      port: 8545,
      network_id: "*",
    },
    ethereum_testnet: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://rpc.ankr.com/eth_goerli"
        );
      },
      network_id: "5",
    },
  },
  mocha: {
    // timeout: 100000
  },
  // Configure your compilers
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
};
