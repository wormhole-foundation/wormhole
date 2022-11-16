const Shutdown = artifacts.require("Shutdown");
module.exports = async function(callback) {
  try {
    const contract = (await Shutdown.new());
    console.log('tx: ' + contract.transactionHash);
    console.log('Shutdown address: ' + contract.address);
    callback();
  } catch (e) {
    callback(e);
  }
};
