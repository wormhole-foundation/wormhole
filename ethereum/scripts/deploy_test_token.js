// run this script with truffle exec

const TokenImplementation = artifacts.require("TokenImplementation")

module.exports = async function(callback) {
    const accounts = await web3.eth.getAccounts();

    // deploy token contract
    const tokenAddress = (await TokenImplementation.new()).address;
    const token = new web3.eth.Contract(TokenImplementation.abi, tokenAddress);

    console.log("Token deployed at: "+tokenAddress);

    // initialize token contract
    await token.methods.initialize(
        "Test Token",
        "TKN",
        "18",        // decimals
        accounts[0], // owner
        "0",
        "0x00000000000000000000000000000000"
    ).send({
        from:accounts[0],
        gas:1000000
    });

    // mint 1000 units
    await token.methods.mint(accounts[0], "1000000000000000000000").send({
        from:accounts[0],
        gas:1000000
    });

    callback();
}