/**
 * @type import('hardhat/config').HardhatUserConfig
 */

require('@nomicfoundation/hardhat-toolbox')
require('@quai/quais-upgrades');
require("@quai/hardhat-deploy-metadata");

const dotenv = require('dotenv')
dotenv.config({ path: '.env' })

module.exports = {
  defaultNetwork: 'cyprus1',
  networks: {
    cyprus1: {
      url: process.env.RPC_URL,
      accounts: [process.env.CYPRUS1_PK],
      chainId: Number(process.env.CHAIN_ID),
    },
    cyprus1_fullpath: {
      url: "https://orchard.rpc.quai.network/cyprus1",
      accounts: [process.env.CYPRUS1_PK],
      chainId: Number(process.env.CHAIN_ID),
    },
  },

  solidity: {
    compilers: [
      {
      version: '0.8.17',
      settings: {
        optimizer: {
          enabled: true,
          runs: 200,
        },
        metadata: {
          bytecodeHash: 'ipfs',
          useLiteralContent: true, // Include the source code in the metadata
        },
        evmVersion: 'london',
      },
    },
  ]
  },

  paths: {
    sources: './contracts',
    cache: './cache',
    artifacts: './artifacts',
  },
  mocha: {
    timeout: 20000,
  },
}
