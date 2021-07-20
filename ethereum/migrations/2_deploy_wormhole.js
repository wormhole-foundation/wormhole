require('dotenv').config({ path: "../.env" });

const Implementation = artifacts.require("Implementation");
const Wormhole = artifacts.require("Wormhole");

// CONFIG
const initialSigners = JSON.parse(process.env.INIT_SIGNERS);
const chainId = process.env.INIT_CHAIN_ID;
const governanceChainId = process.env.INIT_GOV_CHAIN_ID;
const governanceContract = process.env.INIT_GOV_CONTRACT; // bytes32

module.exports = async function (deployer) {
    // deploy implementation
    await deployer.deploy(Implementation);

    // encode initialisation data
    const impl = new web3.eth.Contract(Implementation.abi, Implementation.address);
    const initData = impl.methods.initialize(
        initialSigners,
        chainId,
        governanceChainId,
        governanceContract
    ).encodeABI();

    // console.log(initData)

    // deploy proxy
    await deployer.deploy(Wormhole, Implementation.address, initData);
};
