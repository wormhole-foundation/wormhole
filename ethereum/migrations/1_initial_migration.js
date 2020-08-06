const Schnorr = artifacts.require("Schnorr");
const WrappedAsset = artifacts.require("WrappedAsset");
const Wormhole = artifacts.require("Wormhole");

module.exports = async function (deployer) {
    await deployer.deploy(Schnorr);
    await deployer.deploy(WrappedAsset);
    await deployer.link(Schnorr, Wormhole);
    await deployer.deploy(Wormhole, {
        x: "15420174358166353706216094226583628565375637765325964030087969534155416299009",
        parity: 1,
        expiration_time: 0
    }, WrappedAsset.address, 1000);
};
