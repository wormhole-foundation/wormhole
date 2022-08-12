require('dotenv').config({ path: "../.env" });

const Wormhole = artifacts.require("Wormhole");
const MockBatchedVAASender = artifacts.require("MockBatchedVAASender");

module.exports = async function (deployer, network, accounts) {

    await deployer.deploy(MockBatchedVAASender)

    const contract = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

    await contract.methods.setup(
        Wormhole.address
    ).send({from: accounts[0]})
};
