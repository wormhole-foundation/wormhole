require("dotenv").config({ path: ".env" });
const HDWalletProvider = require("@truffle/hdwallet-provider");
const KLAYHDWalletProvider = require("truffle-hdwallet-provider-klaytn");
const Caver = require("caver-js");

module.exports = {
  networks: {
    development: {
      host: "127.0.0.1",
      port: 8545,
      network_id: "*",
    },
    // test network is the same as development but allows us to omit certain migrations
    test: {
      host: "127.0.0.1",
      port: 8545,
      network_id: "*",
    },
    mainnet: {
      provider: () =>
        new HDWalletProvider(
          process.env.MNEMONIC,
          `https://mainnet.infura.io/v3/` + process.env.INFURA_KEY
        ),
      network_id: 1,
      gas: 10000000,
      gasPrice: 191000000000,
      confirmations: 1,
      timeoutBlocks: 200,
      skipDryRun: false,
    },
    rinkeby: {
      provider: () =>
        new HDWalletProvider(
          process.env.MNEMONIC,
          `https://rinkeby.infura.io/v3/` + process.env.INFURA_KEY
        ),
      network_id: 4,
      gas: 5500000,
      confirmations: 2,
      timeoutBlocks: 200,
      skipDryRun: true,
    },
    goerli: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://goerli.infura.io/v3/" + process.env.INFURA_KEY
        );
      },
      network_id: "5",
    },
    binance: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://bsc-dataseed.binance.org/"
        );
      },
      network_id: "56",
      gas: 70000000,
      gasPrice: 8000000000,
    },
    binance_testnet: {
      provider: () =>
        new HDWalletProvider(
          process.env.MNEMONIC,
          "https://data-seed-prebsc-1-s1.binance.org:8545/"
        ),
      network_id: "97",
    },
    polygon: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://polygon-rpc.com"
        );
      },
      network_id: "137",
      gas: 10000000,
      gasPrice: 700000000000,
    },
    mumbai: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://polygon-mumbai.infura.io/v3/" + process.env.INFURA_KEY
        );
      },
      network_id: "80001",
    },
    avalanche: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://api.avax.network/ext/bc/C/rpc"
        );
      },
      network_id: "43114",
      gas: 8000000,
      gasPrice: 26000000000,
    },
    fuji: {
      provider: () =>
        new HDWalletProvider(
          process.env.MNEMONIC,
          "https://api.avax-test.network/ext/bc/C/rpc"
        ),
      network_id: "43113",
    },
    oasis: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://emerald.oasis.dev/"
        );
      },
      network_id: 42262,
      gas: 4465030,
      gasPrice: 30000000000,
    },
    oasis_testnet: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://testnet.emerald.oasis.dev"
        );
      },
      network_id: 42261,
    },
    aurora: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://mainnet.aurora.dev"
        );
      },
      network_id: 0x4e454152,
      from: "DEPLOYER_PUBLIC_KEY_HERE",
    },
    aurora_testnet: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://testnet.aurora.dev"
        );
      },
      network_id: 0x4e454153,
      gas: 10000000,
      from: "0x3bC7f2e458aC4E55F941C458cfD8c6851a591B4F", // public key
    },
    fantom: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://rpc.ftm.tools/"
        );
      },
      network_id: 250,
      gas: 8000000,
      gasPrice: 3000000000000,
      timeoutBlocks: 15000,
    },
    fantom_testnet: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://rpc.testnet.fantom.network/"
        );
      },
      network_id: 0xfa2,
    },
    karura: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          // To use this local host, needed to run this: ENDPOINT_URL=wss://karura-rpc-1.aca-api.network npx @acala-network/eth-rpc-adapter@latest
          "http://localhost:8545"
          //"https://eth-rpc-karura.aca-api.network/"
        );
      },
      network_id: 686,
      gasPrice: "0x2f7e8803ea",
      gasLimit: "0x329b140",
      gas: "0x329b140",
    },
    karura_testnet: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://karura-dev.aca-dev.network/eth/http"
        );
      },
      network_id: 596,
      gasPrice: "0x2f7e8805de",
      gasLimit: "0x329b140",
      gas: "0x329b0dc",
    },
    acala: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          // To use this local host, needed to run this: ENDPOINT_URL=wss://acala-rpc-0.aca-api.network npx @acala-network/eth-rpc-adapter@latest
          //"http://localhost:8545"
          "https://eth-rpc-acala.aca-api.network/"
        );
      },
      network_id: 787,
      gasPrice: "0x2f25eb03ea",
      gas: "0x6fc3540",
    },
    acala_testnet: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://acala-dev.aca-dev.network/eth/http"
        );
      },
      network_id: 597,
      gasPrice: "0x2ed51903ea",
      gasLimit: "0x6fc3540",
      gas: "0x329b0dc",
    },
    klaytn: {
      // Note that Klaytn works with version 5.3.14 of truffle, but not some of the newer versions.
      provider: () => {
        const option = {
          headers: [
            {
              name: "Authorization",
              value:
                "Basic " +
                Buffer.from(
                  process.env.KLAY_ACCESS_ID +
                    ":" +
                    process.env.KLAY_SECURITY_KEY
                ).toString("base64"),
            },
            { name: "x-chain-id", value: "8217" },
          ],
          keepAlive: false,
        };
        return new KLAYHDWalletProvider(
          process.env.MNEMONIC,
          new Caver.providers.HttpProvider(
            "https://node-api.klaytnapi.com/v1/klaytn",
            option
          )
        );
      },
      network_id: 8217, //Klaytn mainnet's network id
      gas: "8000000",
      gasPrice: "750000000000",
      disableConfirmationListener: true,
      pollingInterval: 1800000,
    },
    klaytn_testnet: {
      // Note that Klaytn works with version 5.3.14 of truffle, but not some of the newer versions.
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://api.baobab.klaytn.net:8651/"
        );
      },
      network_id: "1001",
      gas: "8500000",
      gasPrice: null,
    },
    celo: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://forno.celo.org"
        );
      },
      network_id: 42220,
      gas: 8000000,
      gasPrice: null,
    },
    celo_alfajores_testnet: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://alfajores-forno.celo-testnet.org"
        );
      },
      network_id: 44787,
    },
    moonbeam_testnet: {
      provider: () => {
        return new HDWalletProvider(
          process.env.MNEMONIC,
          "https://rpc.api.moonbase.moonbeam.network"
        );
      },
      network_id: 1287,
      gasPrice: 3000000000, // 3.0 gwei
      timeoutBlocks: 15000,
    },
    neon_devnet: {
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
