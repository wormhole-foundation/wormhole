const jsonfile = require('jsonfile');
const BigNumber = require('bignumber.js');

const Migrator = artifacts.require("Migrator");
const TokenImplementation = artifacts.require("TokenImplementation");

contract("Migrator", function (accounts) {
    var migrator,
        fromToken,
        toToken,
        fromDecimals = 8,
        toDecimals = 18;

    it("should deploy with the correct values", async function () {
        fromToken = await TokenImplementation.new();
        await fromToken.initialize(
            "TestFrom",
            "FROM",
            fromDecimals,
            0,
            accounts[0],
            0,
            "0x00"
        )
        toToken = await TokenImplementation.new();
        await toToken.initialize(
            "TestTo",
            "TO",
            toDecimals,
            0,
            accounts[0],
            0,
            "0x00"
        )

        migrator = await Migrator.new(
            fromToken.address,
            toToken.address,
        );

        assert.equal(await migrator.fromAsset(), fromToken.address)
        assert.equal(await migrator.toAsset(), toToken.address)
        assert.equal((await migrator.fromDecimals()).toNumber(), fromDecimals)
        assert.equal((await migrator.toDecimals()).toNumber(), toDecimals)
    })

    it("should give out LP tokens 1:1 for a toToken deposit", async function () {
        await toToken.mint(accounts[0], "1000000000000000000")
        await toToken.approve(migrator.address, "1000000000000000000")
        await migrator.add("1000000000000000000")


        assert.equal((await toToken.balanceOf(migrator.address)).toString(), "1000000000000000000")
        assert.equal((await migrator.balanceOf(accounts[0])).toString(), "1000000000000000000")
    })

    it("should refund toToken for LP tokens", async function () {
        await migrator.remove("500000000000000000")

        assert.equal((await toToken.balanceOf(migrator.address)).toString(), "500000000000000000")
        assert.equal((await toToken.balanceOf(accounts[0])).toString(), "500000000000000000")
        assert.equal((await migrator.balanceOf(accounts[0])).toString(), "500000000000000000")
    })

    it("should redeem fromToken to toToken adjusting for decimals", async function () {
        await fromToken.mint(accounts[1], "50000000")
        await fromToken.approve(migrator.address, "50000000", {
            from : accounts[1]
        })
        await migrator.migrate("50000000", {
            from : accounts[1]
        })

        assert.equal((await toToken.balanceOf(accounts[1])).toString(), "500000000000000000")
        assert.equal((await fromToken.balanceOf(accounts[1])).toString(), "0")
        assert.equal((await fromToken.balanceOf(migrator.address)).toString(), "50000000")
        assert.equal((await toToken.balanceOf(migrator.address)).toString(), "0")
    })

    it("fromToken should be claimable for LP tokens, adjusting for decimals", async function () {
        await migrator.claim("500000000000000000")

        assert.equal((await fromToken.balanceOf(migrator.address)).toString(), "0")
        assert.equal((await fromToken.balanceOf(accounts[0])).toString(), "50000000")
        assert.equal((await migrator.balanceOf(accounts[0])).toString(), "0")
    })
})