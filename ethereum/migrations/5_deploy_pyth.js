require('dotenv').config({ path: "../.env" });

const PythDataBridge = artifacts.require("PythDataBridge");
const PythImplementation = artifacts.require("PythImplementation");
const PythSetup = artifacts.require("PythSetup");
const Wormhole = artifacts.require("Wormhole");

const chainId = process.env.PYTH_INIT_CHAIN_ID;
const governanceChainId = process.env.PYTH_INIT_GOV_CHAIN_ID;
const governanceContract = process.env.PYTH_INIT_GOV_CONTRACT; // bytes32
const pyth2WormholeChainId = process.env.PYTH_TO_WORMHOLE_CHAIN_ID;
const pyth2WormholeContract = process.env.PYTH_TO_WORMHOLE_CONTRACT; // bytes32

module.exports = async function (deployer) {
    // deploy implementation
    await deployer.deploy(PythImplementation);
    // deploy implementation
    await deployer.deploy(PythSetup);

    // encode initialisation data
    const setup = new web3.eth.Contract(PythSetup.abi, PythSetup.address);
    const initData = setup.methods.setup(
        PythImplementation.address,

        chainId,
        (await Wormhole.deployed()).address,

        governanceChainId,
        governanceContract,

        pyth2WormholeChainId,
        pyth2WormholeContract,
    ).encodeABI();

    // deploy proxy
    await deployer.deploy(PythDataBridge, PythSetup.address, initData);
};
