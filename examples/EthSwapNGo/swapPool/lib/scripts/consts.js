"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.SOLANA_TEST_WALLET_PUBLIC_KEY = exports.SOLANA_TEST_TOKEN = exports.ETH_TEST_WALLET_PUBLIC_KEY = exports.getSignerForChain = exports.SOLANA_PRIVATE_KEY = exports.ETH_PRIVATE_KEY = exports.BSC_NODE_URL = exports.POLYGON_NODE_URL = exports.ETH_NODE_URL = exports.WORMHOLE_RPC_HOSTS = exports.getTokenBridgeAddressForChain = exports.getNFTBridgeAddressForChain = exports.getBridgeAddressForChain = exports.TERRA_TOKEN_BRIDGE_ADDRESS = exports.TERRA_BRIDGE_ADDRESS = exports.SOL_TOKEN_BRIDGE_ADDRESS = exports.SOL_NFT_BRIDGE_ADDRESS = exports.SOL_BRIDGE_ADDRESS = exports.POLYGON_TOKEN_BRIDGE_ADDRESS = exports.POLYGON_NFT_BRIDGE_ADDRESS = exports.POLYGON_BRIDGE_ADDRESS = exports.BSC_TOKEN_BRIDGE_ADDRESS = exports.BSC_NFT_BRIDGE_ADDRESS = exports.BSC_BRIDGE_ADDRESS = exports.ETH_TOKEN_BRIDGE_ADDRESS = exports.ETH_NFT_BRIDGE_ADDRESS = exports.ETH_BRIDGE_ADDRESS = exports.TERRA_HOST = exports.SOLANA_HOST = exports.CLUSTER = void 0;
var wormhole_sdk_1 = require("@certusone/wormhole-sdk");
var web3_js_1 = require("@solana/web3.js");
var ethers_1 = require("ethers");
var utils_1 = require("ethers/lib/utils");
exports.CLUSTER = "devnet"; //This is the currently selected environment.
exports.SOLANA_HOST = process.env.REACT_APP_SOLANA_API_URL
    ? process.env.REACT_APP_SOLANA_API_URL
    : exports.CLUSTER === "mainnet"
        ? web3_js_1.clusterApiUrl("mainnet-beta")
        : exports.CLUSTER === "testnet"
            ? web3_js_1.clusterApiUrl("testnet")
            : "http://localhost:8899";
exports.TERRA_HOST = exports.CLUSTER === "mainnet"
    ? {
        URL: "https://lcd.terra.dev",
        chainID: "columbus-5",
        name: "mainnet",
    }
    : exports.CLUSTER === "testnet"
        ? {
            URL: "https://bombay-lcd.terra.dev",
            chainID: "bombay-12",
            name: "testnet",
        }
        : {
            URL: "http://localhost:1317",
            chainID: "columbus-5",
            name: "localterra",
        };
exports.ETH_BRIDGE_ADDRESS = utils_1.getAddress(exports.CLUSTER === "mainnet"
    ? "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
    : exports.CLUSTER === "testnet"
        ? "0x44F3e7c20850B3B5f3031114726A9240911D912a"
        : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550");
exports.ETH_NFT_BRIDGE_ADDRESS = utils_1.getAddress(exports.CLUSTER === "mainnet"
    ? "0x6FFd7EdE62328b3Af38FCD61461Bbfc52F5651fE"
    : exports.CLUSTER === "testnet"
        ? "0x26b4afb60d6c903165150c6f0aa14f8016be4aec" // TODO: test address
        : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec");
exports.ETH_TOKEN_BRIDGE_ADDRESS = utils_1.getAddress(exports.CLUSTER === "mainnet"
    ? "0x3ee18B2214AFF97000D974cf647E7C347E8fa585"
    : exports.CLUSTER === "testnet"
        ? "0xa6CDAddA6e4B6704705b065E01E52e2486c0FBf6"
        : "0x0290FB167208Af455bB137780163b7B7a9a10C16");
exports.BSC_BRIDGE_ADDRESS = utils_1.getAddress(exports.CLUSTER === "mainnet"
    ? "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
    : exports.CLUSTER === "testnet"
        ? "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550" // TODO: test address
        : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550");
exports.BSC_NFT_BRIDGE_ADDRESS = utils_1.getAddress(exports.CLUSTER === "mainnet"
    ? "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE"
    : exports.CLUSTER === "testnet"
        ? "0x26b4afb60d6c903165150c6f0aa14f8016be4aec" // TODO: test address
        : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec");
exports.BSC_TOKEN_BRIDGE_ADDRESS = utils_1.getAddress(exports.CLUSTER === "mainnet"
    ? "0xB6F6D86a8f9879A9c87f643768d9efc38c1Da6E7"
    : exports.CLUSTER === "testnet"
        ? "0x0290FB167208Af455bB137780163b7B7a9a10C16" // TODO: test address
        : "0x0290FB167208Af455bB137780163b7B7a9a10C16");
exports.POLYGON_BRIDGE_ADDRESS = utils_1.getAddress(exports.CLUSTER === "mainnet"
    ? "0x7A4B5a56256163F07b2C80A7cA55aBE66c4ec4d7"
    : exports.CLUSTER === "testnet"
        ? "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550" // TODO: test address
        : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550");
exports.POLYGON_NFT_BRIDGE_ADDRESS = utils_1.getAddress(exports.CLUSTER === "mainnet"
    ? "0x90BBd86a6Fe93D3bc3ed6335935447E75fAb7fCf"
    : exports.CLUSTER === "testnet"
        ? "0x26b4afb60d6c903165150c6f0aa14f8016be4aec" // TODO: test address
        : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec");
exports.POLYGON_TOKEN_BRIDGE_ADDRESS = utils_1.getAddress(exports.CLUSTER === "mainnet"
    ? "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE"
    : exports.CLUSTER === "testnet"
        ? "0x0290FB167208Af455bB137780163b7B7a9a10C16" // TODO: test address
        : "0x0290FB167208Af455bB137780163b7B7a9a10C16");
exports.SOL_BRIDGE_ADDRESS = exports.CLUSTER === "mainnet"
    ? "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
    : exports.CLUSTER === "testnet"
        ? "Brdguy7BmNB4qwEbcqqMbyV5CyJd2sxQNUn6NEpMSsUb"
        : "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o";
exports.SOL_NFT_BRIDGE_ADDRESS = exports.CLUSTER === "mainnet"
    ? "WnFt12ZrnzZrFZkt2xsNsaNWoQribnuQ5B5FrDbwDhD"
    : exports.CLUSTER === "testnet"
        ? "NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA" // TODO: test address
        : "NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA";
exports.SOL_TOKEN_BRIDGE_ADDRESS = exports.CLUSTER === "mainnet"
    ? "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb"
    : exports.CLUSTER === "testnet"
        ? "A4Us8EhCC76XdGAN17L4KpRNEK423nMivVHZzZqFqqBg"
        : "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE";
exports.TERRA_BRIDGE_ADDRESS = exports.CLUSTER === "mainnet"
    ? "terra1dq03ugtd40zu9hcgdzrsq6z2z4hwhc9tqk2uy5"
    : exports.CLUSTER === "testnet"
        ? "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5"
        : "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5";
exports.TERRA_TOKEN_BRIDGE_ADDRESS = exports.CLUSTER === "mainnet"
    ? "terra10nmmwe8r3g99a9newtqa7a75xfgs2e8z87r2sf"
    : exports.CLUSTER === "testnet"
        ? "terra10pyejy66429refv3g35g2t7am0was7ya7kz2a4"
        : "terra10pyejy66429refv3g35g2t7am0was7ya7kz2a4";
var getBridgeAddressForChain = function (chainId) {
    return chainId === wormhole_sdk_1.CHAIN_ID_SOLANA
        ? exports.SOL_BRIDGE_ADDRESS
        : chainId === wormhole_sdk_1.CHAIN_ID_ETH
            ? exports.ETH_BRIDGE_ADDRESS
            : chainId === wormhole_sdk_1.CHAIN_ID_BSC
                ? exports.BSC_BRIDGE_ADDRESS
                : chainId === wormhole_sdk_1.CHAIN_ID_TERRA
                    ? exports.TERRA_BRIDGE_ADDRESS
                    : chainId === wormhole_sdk_1.CHAIN_ID_POLYGON
                        ? exports.POLYGON_BRIDGE_ADDRESS
                        : "";
};
exports.getBridgeAddressForChain = getBridgeAddressForChain;
var getNFTBridgeAddressForChain = function (chainId) {
    return chainId === wormhole_sdk_1.CHAIN_ID_SOLANA
        ? exports.SOL_NFT_BRIDGE_ADDRESS
        : chainId === wormhole_sdk_1.CHAIN_ID_ETH
            ? exports.ETH_NFT_BRIDGE_ADDRESS
            : chainId === wormhole_sdk_1.CHAIN_ID_BSC
                ? exports.BSC_NFT_BRIDGE_ADDRESS
                : chainId === wormhole_sdk_1.CHAIN_ID_POLYGON
                    ? exports.POLYGON_NFT_BRIDGE_ADDRESS
                    : "";
};
exports.getNFTBridgeAddressForChain = getNFTBridgeAddressForChain;
var getTokenBridgeAddressForChain = function (chainId) {
    return chainId === wormhole_sdk_1.CHAIN_ID_SOLANA
        ? exports.SOL_TOKEN_BRIDGE_ADDRESS
        : chainId === wormhole_sdk_1.CHAIN_ID_ETH
            ? exports.ETH_TOKEN_BRIDGE_ADDRESS
            : chainId === wormhole_sdk_1.CHAIN_ID_BSC
                ? exports.BSC_TOKEN_BRIDGE_ADDRESS
                : chainId === wormhole_sdk_1.CHAIN_ID_TERRA
                    ? exports.TERRA_TOKEN_BRIDGE_ADDRESS
                    : chainId === wormhole_sdk_1.CHAIN_ID_POLYGON
                        ? exports.POLYGON_TOKEN_BRIDGE_ADDRESS
                        : "";
};
exports.getTokenBridgeAddressForChain = getTokenBridgeAddressForChain;
exports.WORMHOLE_RPC_HOSTS = exports.CLUSTER === "mainnet"
    ? [
        "https://wormhole-v2-mainnet-api.certus.one",
        "https://wormhole.inotel.ro",
        "https://wormhole-v2-mainnet-api.mcf.rocks",
        "https://wormhole-v2-mainnet-api.chainlayer.network",
        "https://wormhole-v2-mainnet-api.staking.fund",
        "https://wormhole-v2-mainnet.01node.com",
    ]
    : exports.CLUSTER === "testnet"
        ? ["https://wormhole-v2-testnet-api.certus.one"]
        : ["http://localhost:7071"];
exports.ETH_NODE_URL = "ws://localhost:8545"; //TODO testnet
exports.POLYGON_NODE_URL = "ws:localhost:0000"; //TODO
exports.BSC_NODE_URL = "ws://localhost:8546"; //TODO testnet
exports.ETH_PRIVATE_KEY = "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d";
exports.SOLANA_PRIVATE_KEY = new Uint8Array([
    14, 173, 153, 4, 176, 224, 201, 111, 32, 237, 183, 185, 159, 247, 22, 161, 89,
    84, 215, 209, 212, 137, 10, 92, 157, 49, 29, 192, 101, 164, 152, 70, 87, 65,
    8, 174, 214, 157, 175, 126, 98, 90, 54, 24, 100, 177, 247, 77, 19, 112, 47,
    44, 165, 109, 233, 102, 14, 86, 109, 29, 134, 145, 132, 141,
]);
function getSignerForChain(chainId) {
    var provider = new ethers_1.ethers.providers.WebSocketProvider(chainId === wormhole_sdk_1.CHAIN_ID_POLYGON
        ? exports.POLYGON_NODE_URL
        : chainId === wormhole_sdk_1.CHAIN_ID_BSC
            ? exports.BSC_NODE_URL
            : exports.ETH_NODE_URL);
    var signer = new ethers_1.ethers.Wallet(exports.ETH_PRIVATE_KEY, provider);
    return signer;
}
exports.getSignerForChain = getSignerForChain;
exports.ETH_TEST_WALLET_PUBLIC_KEY = "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
exports.SOLANA_TEST_TOKEN = "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ"; //SOLT on devnet
exports.SOLANA_TEST_WALLET_PUBLIC_KEY = "6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J";
