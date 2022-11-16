const Shutdown = artifacts.require("BridgeShutdown");
module.exports = async function(callback) {
  try {
    const contract = (await Shutdown.new());
    console.log('tx: ' + contract.transactionHash);
    console.log('Bridge address: ' + contract.address);
    callback();
  } catch (e) {
    callback(e);
  }
};
