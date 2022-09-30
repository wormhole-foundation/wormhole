// MNEMONIC="" npm run truffle -- exec scripts/deploy_immediate_publish.js --network ethereum_testnet
const ImmediatePublish = artifacts.require("ImmediatePublish");
module.exports = async function (callback) {
  try {
    const GOERLI_TESTNET_WORMHOLE_CORE_ADDRESS =
      "0x706abc4E45D419950511e474C7B9Ed348A4a716c";
    const immediatePublish = await ImmediatePublish.new(
      GOERLI_TESTNET_WORMHOLE_CORE_ADDRESS
    );
    console.log("tx: " + immediatePublish.transactionHash);
    console.log("ImmediatePublish address: " + immediatePublish.address);
    callback();
  } catch (e) {
    callback(e);
  }
};
