const Shutdown = artifacts.require("NFTBridgeShutdown");
module.exports = async function(callback) {
  try {
    const contract = (await Shutdown.new());
    console.log('tx: ' + contract.transactionHash);
    console.log('NFTBridgeShutdown address: ' + contract.address);
    callback();
  } catch (e) {
    callback(e);
  }
};
