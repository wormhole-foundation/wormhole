const Implementation = artifacts.require("Implementation");
module.exports = async function(callback) {
  try {
    const bridge = (await Implementation.new());
    console.log('tx: ' + bridge.transactionHash);
    console.log('Implementation address: ' + bridge.address);
    callback();
  } catch (e) {
    callback(e);
  }
};
