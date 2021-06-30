const Implementation = artifacts.require("Implementation");
const Wormhole = artifacts.require("Wormhole");

const initialSigners = [
    // testSigner 1 & 2
    "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
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
