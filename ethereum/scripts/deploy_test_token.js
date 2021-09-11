// run this script with truffle exec

const ERC20 = artifacts.require("ERC20PresetMinterPauser");
const ERC721 = artifacts.require("ERC721PresetMinterPauserAutoId");

module.exports = async function(callback) {
  try {
    const accounts = await web3.eth.getAccounts();

    // deploy token contract
    const tokenAddress = (await ERC20.new("Ethereum Test Token", "TKN"))
      .address;
    const token = new web3.eth.Contract(ERC20.abi, tokenAddress);

    console.log("Token deployed at: " + tokenAddress);

    // mint 1000 units
    await token.methods.mint(accounts[0], "1000000000000000000000").send({
      from: accounts[0],
      gas: 1000000,
    });

    const nftAddress = (
      await ERC721.new(
        "Not an APE 🐒",
        "APE🐒",
        "https://cloudflare-ipfs.com/ipfs/QmeSjSinHpPnmXmspMjwiXyN6zS4E9zccariGR3jxcaWtq/"
      )
    ).address;
    const nft = new web3.eth.Contract(ERC721.abi, nftAddress);
    await nft.methods.mint(accounts[0]).send({
      from: accounts[0],
      gas: 1000000,
    });
    await nft.methods.mint(accounts[0]).send({
      from: accounts[0],
      gas: 1000000,
    });

    console.log("NFT deployed at: " + nftAddress);

    callback();
  } catch (e) {
    callback(e);
  }
};
