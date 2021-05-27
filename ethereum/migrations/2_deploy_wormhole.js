const Implementation = artifacts.require("Implementation");
const Wormhole = artifacts.require("Wormhole");

const initialSigners = [
    // testSigner 1 & 2
    "0x7b6FA3F2bEb40eAf9Cefcb20505163C70d76f21c",
    "0x4ba0C2db9A26208b3bB1a50B01b16941c10D76db",
]
const chainId = "0x2";
const governanceChainId = "0x3";
const governanceContract = "0x0000000000000000000000000000000000000000000000000000000000000004"; // bytes32

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
