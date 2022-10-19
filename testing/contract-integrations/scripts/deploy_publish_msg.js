const PublishMsg = artifacts.require("PublishMsg");
module.exports = async function (callback) {
  try {
    const TESTNET_WORMHOLE_CORE_ADDRESS = process.env.TESTNET_WORMHOLE_CORE_ADDRESS;
    if (TESTNET_WORMHOLE_CORE_ADDRESS === "") {
      console.error("Please set \"TESTNET_WORMHOLE_CORE_ADDRESS\"")
      return
    }
    const publishMsg = await PublishMsg.new(
      TESTNET_WORMHOLE_CORE_ADDRESS
    );
    console.log("tx: " + publishMsg.transactionHash);
    console.log("PublishMsg address: " + publishMsg.address);
    callback();
  } catch (e) {
    callback(e);
  }
};
