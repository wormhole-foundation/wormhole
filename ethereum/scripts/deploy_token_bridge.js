const BridgeImplementation = artifacts.require("BridgeImplementation");
module.exports = async function(callback) {
  try {
    const bridge = (await BridgeImplementation.new());
    console.log('tx: ' + bridge.transactionHash);
    console.log('Bridge address: ' + bridge.address);
    callback();
  } catch (e) {
    callback(e);
  }
};
