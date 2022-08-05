// run this script with truffle exec

const ERC20 = artifacts.require("ERC20PresetMinterPauser");
const ERC721 = artifacts.require("ERC721PresetMinterPauserAutoId");

const interateToStandardTransactionCount = async () => {
  const accounts = await web3.eth.getAccounts();

  const transactionCount = await web3.eth.getTransactionCount(
    accounts[0],
    "latest"
  );
  console.log(
    "transaction count prior to test token deploys: ",
    transactionCount
  );

  const transactionsToBurn = 32 - transactionCount;
  const promises = [];
  for (let i = 0; i < transactionsToBurn; i++) {
    promises.push(
      web3.eth.sendTransaction({
        to: accounts[0],
        from: accounts[0],
        value: 530,
      })
    );
  }

  await Promise.all(promises);

  const burnCount = await web3.eth.getTransactionCount(accounts[0], "latest");

  console.log("transaction count after burn: ", burnCount);

  return Promise.resolve();
};

module.exports = async function(callback) {
  try {
    const accounts = await web3.eth.getAccounts();

    //Contracts deployed via this script deploy to an address which is determined by the number of transactions
    //which have been performed on the chain.
    //This is, however, variable. For example, if you optionally deploy contracts, more transactions are
    //performed than if you didn't.
    //In order to make sure the test contracts deploy to a location
    //which is deterministic with regard to other environment conditions, we fire bogus transactions up to a safe
    //count, currently 32, before deploying the test contracts.
    await interateToStandardTransactionCount();

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

    const MockWETH9 = await artifacts.require("MockWETH9");
    //WETH deploy
    // deploy token contract
    const wethAddress = (await MockWETH9.new()).address;
    const wethToken = new web3.eth.Contract(MockWETH9.abi, wethAddress);

    console.log("WETH token deployed at: " + wethAddress);

    for (let idx = 2; idx < 10; idx++) {
      await token.methods.mint(accounts[idx], "1000000000000000000000").send({
        from: accounts[0],
        gas: 1000000,
      });
    }

    // devnet WETH token address should be deterministic
    if (wethAddress !== "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E") {
      throw new Error("unexpected WETH token address");
    }

    callback();
  } catch (e) {
    callback(e);
  }
};
