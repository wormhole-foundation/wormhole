const Wormhole = artifacts.require("Wormhole");
const MockBatchedVAASender = artifacts.require("MockBatchedVAASender");

module.exports = async function(callback) {
  try {
    const accounts = await web3.eth.getAccounts();

    await MockBatchedVAASender.deploy();

    // devnet contract address should be deterministic
    if (MockBatchedVAASender.address !== "0xf19a2a01b70519f67adb309a994ec8c69a967e8b") {
      throw new Error("unexpected batched-VAA contract address");
    }

    const batchedSender = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);
    await batchedSender.methods.setup(Wormhole.address).send({from: accounts[0]});

    callback();
  } catch (e) {
    callback(e);
  }
};
