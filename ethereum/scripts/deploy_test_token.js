// run this script with truffle exec

const ERC20 = artifacts.require("ERC20PresetMinterPauser")

module.exports = async function (callback) {
    try {
        const accounts = await web3.eth.getAccounts();

        // deploy token contract
        const tokenAddress = (await ERC20.new("Ethereum Test Token", "TKN")).address;
        const token = new web3.eth.Contract(ERC20.abi, tokenAddress);

        console.log("Token deployed at: " + tokenAddress);

        // mint 1000 units
        await token.methods.mint(accounts[0], "1000000000000000000000").send({
            from: accounts[0],
            gas: 1000000
        });

        callback();
    } catch (e) {
        callback(e);
    }
}