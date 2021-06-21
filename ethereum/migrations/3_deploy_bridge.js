require('dotenv').config({ path: "../.env" });

const TokenBridge = artifacts.require("TokenBridge");
const BridgeImplementation = artifacts.require("BridgeImplementation");
const TokenImplementation = artifacts.require("TokenImplementation");
const Wormhole = artifacts.require("Wormhole");

const chainId = process.env.BRIDGE_INIT_CHAIN_ID;
const governanceChainId = process.env.BRIDGE_INIT_GOV_CHAIN_ID;
const governanceContract = process.env.BRIDGE_INIT_GOV_CONTRACT; // bytes32
const WETH = process.env.BRIDGE_INIT_WETH;

module.exports = async function (deployer) {
    // deploy token implementation
    await deployer.deploy(TokenImplementation);

    // deploy implementation
    await deployer.deploy(BridgeImplementation);

    // encode initialisation data
    const impl = new web3.eth.Contract(BridgeImplementation.abi, BridgeImplementation.address);
    const initData = impl.methods.initialize(
        chainId,
        (await Wormhole.deployed()).address,
        governanceChainId,
        governanceContract,
        TokenImplementation.address,
        WETH
    ).encodeABI();

    // deploy proxy
    await deployer.deploy(TokenBridge, BridgeImplementation.address, initData);
};
