const { ethers } = require("ethers");

(async () => {
  TESTNET_RPC = process.env.TESTNET_RPC;
  if (TESTNET_RPC === "") {
    console.error("Please set \"TESTNET_RPC\"")
    return
  }

  PUBLISH_MSG_ADDRESS = process.env.PUBLISH_MSG_ADDRESS;
  if (PUBLISH_MSG_ADDRESS === "") {
    console.error("Please set \"PUBLISH_MSG_ADDRESS\"")
    return
  }
  
  CONSISTENCY_LEVEL = process.env.CONSISTENCY_LEVEL;
  if (CONSISTENCY_LEVEL === "") {
    console.error("Please set \"CONSISTENCY_LEVEL\"")
    return
  }
  consistencyLevel = parseInt(CONSISTENCY_LEVEL, 10)

  console.log("publishing a message to contract " +
    PUBLISH_MSG_ADDRESS + 
    ", using rpc " + 
    TESTNET_RPC + 
    " and consistency level " +
    consistencyLevel);

  const provider = new ethers.providers.JsonRpcProvider(TESTNET_RPC);
  const signer = new ethers.Wallet(process.env.MNEMONIC, provider);
  const contract = new ethers.Contract(
    PUBLISH_MSG_ADDRESS,
    ["function publishMsg(uint8 consistencyLevel) public payable returns (uint64 sequence)"],
    signer
  );
  const tx = await contract.publishMsg(consistencyLevel);
  const receipt = await tx.wait();
  console.log(receipt);
})();
