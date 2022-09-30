// MNEMONIC="" node src/immediatePublish.test.js
const { ethers } = require("ethers");

(async () => {
  const provider = new ethers.providers.JsonRpcProvider(
    "https://rpc.ankr.com/eth_goerli"
  );
  const signer = new ethers.Wallet(process.env.MNEMONIC, provider);
  const contract = new ethers.Contract(
    "0x1cd29DCf037769c2Fe6Ba2b1921e6E1D3d653FFC",
    ["function immediatePublish() public payable returns (uint64 sequence)"],
    signer
  );
  const tx = await contract.immediatePublish();
  const receipt = await tx.wait();
  console.log(receipt);
})();
