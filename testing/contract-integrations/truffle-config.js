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
    neon_testnet: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://proxy.devnet.neonlabs.org/solana"
        );
      },
      network_id: "*",
      gas: 3000000000,
      gasPrice: 443065000000,
    },
    arbitrum_testnet: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://goerli-rollup.arbitrum.io/rpc"
        );
      },
      network_id: 421613,
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
