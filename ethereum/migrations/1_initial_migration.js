const Schnorr = artifacts.require("Schnorr");
const WrappedAsset = artifacts.require("WrappedAsset");
const Wormhole = artifacts.require("Wormhole");

module.exports = async function (deployer) {
    await deployer.deploy(Schnorr);
    await deployer.deploy(WrappedAsset);
    await deployer.link(Schnorr, Wormhole);
    await deployer.deploy(Wormhole, {
        keys: ["0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf"],
        expiration_time: 0
    }, WrappedAsset.address, 1000);
};
