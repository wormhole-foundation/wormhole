var WETH9 = artifacts.require("MockWETH9");

module.exports = function(deployer) {
    deployer.deploy(WETH9);
};
