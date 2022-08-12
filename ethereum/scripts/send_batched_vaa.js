const MockBatchedVAASender = artifacts.require("MockBatchedVAASender");

module.exports = async function(callback) {
  try {
    const accounts = await web3.eth.getAccounts();

    const batchedSender = await MockBatchedVAASender.deployed()

    const contract = new web3.eth.Contract(MockBatchedVAASender.abi, batchedSender.address);

    const nonce = Math.round(Date.now() / 1000);
    const nonceHex = nonce.toString(16)

    const res = await contract.methods.sendMultipleMessages(
      "0x" + nonceHex,
      "0x1",
      32
    ).send({
      value: 0,
      from: accounts[0]
    });

    console.log('sendMultipleMessages response', res)

    callback();
  } catch (e) {
    callback(e);
  }
};
