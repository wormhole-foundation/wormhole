const WrappedAsset = artifacts.require("WrappedAsset");
const Wormhole = artifacts.require("Wormhole");
const ERC20 = artifacts.require("ERC20PresetMinterPauser");

module.exports = async function (deployer) {
    let bridge = await Wormhole.deployed();
    let token = await ERC20.deployed();

    console.log("Token:", token.address);

    // Create example ERC20 and mint a generous amount of it.
    await token.mint("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1", "1000000000000000000");
    await token.approve(bridge.address, "1000000000000000000");
};
