const Wormhole = artifacts.require("Wormhole");
const MockBatchedVAASender = artifacts.require("MockBatchedVAASender");

module.exports = async function(callback) {
  try {
    const accounts = await web3.eth.getAccounts();

    await MockBatchedVAASender.deploy();

    const batchedSender = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);
    await batchedSender.methods.setup(Wormhole.address).send({from: accounts[0]});

    callback();
  } catch (e) {
    callback(e);
  }
};
