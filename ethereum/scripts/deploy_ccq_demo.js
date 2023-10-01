const QueryDemo = artifacts.require("QueryDemo");
module.exports = async function(callback) {
  const accounts = await web3.eth.getAccounts();
  try {
    // const ccqDemo = await QueryDemo.new(
    //   accounts[0],
    //   "0x0CBE91CF822c73C2315FB05100C2F714765d5c20",
    //   5
    // );
    // const ccqDemo = await QueryDemo.new(
    //   accounts[0],
    //   "0xC7A204bDBFe983FCD8d8E61D02b475D4073fF97e",
    //   23
    // );
    const ccqDemo = await QueryDemo.new(
      accounts[0],
      "0x6b9C8671cdDC8dEab9c719bB87cBd3e782bA6a35",
      24
    );
    console.log("tx: " + ccqDemo.transactionHash);
    console.log("QueryDemo address: " + ccqDemo.address);
    callback();
  } catch (e) {
    callback(e);
  }
};
