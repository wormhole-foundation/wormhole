require('dotenv').config({ path: "../.env" });

const TokenBridge = artifacts.require("NFTBridgeEntrypoint");
const BridgeImplementation = artifacts.require("NFTBridgeImplementation");
const BridgeSetup = artifacts.require("NFTBridgeSetup");
const TokenImplementation = artifacts.require("NFTImplementation");
const Wormhole = artifacts.require("Wormhole");

const chainId = process.env.BRIDGE_INIT_CHAIN_ID;
const governanceChainId = process.env.BRIDGE_INIT_GOV_CHAIN_ID;
const governanceContract = process.env.BRIDGE_INIT_GOV_CONTRACT; // bytes32
const finality = process.env.BRIDGE_INIT_FINALITY;
const evmChainId = process.env.INIT_EVM_CHAIN_ID;

module.exports = async function (deployer) {
    // deploy token implementation
    await deployer.deploy(TokenImplementation);

    // deploy setup
    await deployer.deploy(BridgeSetup);

    // deploy implementation
    await deployer.deploy(BridgeImplementation);

    // encode initialisation data
    const setup = new web3.eth.Contract(BridgeSetup.abi, BridgeSetup.address);
    const initData = setup.methods.setup(
        BridgeImplementation.address,
        chainId,
        (await Wormhole.deployed()).address,
        governanceChainId,
        governanceContract,
        TokenImplementation.address,
        finality,
        evmChainId
    ).encodeABI();

    // deploy proxy
    await deployer.deploy(TokenBridge, BridgeSetup.address, initData);
};
