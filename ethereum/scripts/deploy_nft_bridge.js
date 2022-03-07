const BridgeImplementation = artifacts.require("NFTBridgeImplementation");
module.exports = async function(callback) {
  try {
    const bridge = (await BridgeImplementation.new());
    console.log('tx: ' + bridge.transactionHash);
    console.log('NFTBridge address: ' + bridge.address);
    callback();
  } catch (e) {
    callback(e);
  }
};
