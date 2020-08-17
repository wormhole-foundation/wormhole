const Wormhole = artifacts.require("Wormhole");
const WrappedAsset = artifacts.require("WrappedAsset");
const ERC20 = artifacts.require("ERC20PresetMinterPauser");

// Taken from https://medium.com/fluidity/standing-the-time-of-test-b906fcc374a9
advanceTimeAndBlock = async (time) => {
    await advanceTime(time);
    await advanceBlock();

    return Promise.resolve(web3.eth.getBlock('latest'));
}

advanceTime = (time) => {
    return new Promise((resolve, reject) => {
        web3.currentProvider.send({
            jsonrpc: "2.0",
            method: "evm_increaseTime",
            params: [time],
            id: new Date().getTime()
        }, (err, result) => {
            if (err) {
                return reject(err);
            }
            return resolve(result);
        });
    });
}

advanceBlock = () => {
    return new Promise((resolve, reject) => {
        web3.currentProvider.send({
            jsonrpc: "2.0",
            method: "evm_mine",
            id: new Date().getTime()
        }, (err, result) => {
            if (err) {
                return reject(err);
            }
            const newBlockHash = web3.eth.getBlock('latest').hash;

            return resolve(newBlockHash)
        });
    });
}

contract("Wormhole", function () {
    it("should use master wrapped asset", async function () {
        let bridge = await Wormhole.deployed();
        let wa = await bridge.wrappedAssetMaster.call();
        assert.equal(wa, WrappedAsset.address)
    });

    it("should transfer tokens in on valid VAA", async function () {
        let bridge = await Wormhole.deployed();

        // User locked an asset on the foreign chain and the VAA proving this is transferred in.
        await bridge.submitVAA("0x0100000000010092737a1504f3b3df8c93cb85c64a4860bb270e26026b6e37f095356a406f6af439c6b2e9775fa1c6669525f06edab033ba5d447308f4e3bdb33c0f361dc32ec3015f3700081087000000350102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e99880000000000000000000000000000000000000000000000004563918244f40000")

        // Expect user to have a balance of a new wrapped asset

        // submitVAA has automatically created a new WrappedAsset for the foreign asset that has been transferred in.
        // We know the address because deterministic network. A user would see the address in the submitVAA tx log.
        let wa = new WrappedAsset("0x79183957Be84C0F4dA451E534d5bA5BA3FB9c696");
        assert.equal(await wa.assetChain(), 1)
        // Remote asset's contract address.
        assert.equal(await wa.assetAddress(), "0x0000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988")
        // Account that the user requests the transfer to.
        let balance = await wa.balanceOf("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1");
        assert.equal(balance, "5000000000000000000");
    });

    it("should not accept the same VAA twice", async function () {
        let bridge = await Wormhole.deployed();
        try {
            await bridge.submitVAA("0x0100000000010092737a1504f3b3df8c93cb85c64a4860bb270e26026b6e37f095356a406f6af439c6b2e9775fa1c6669525f06edab033ba5d447308f4e3bdb33c0f361dc32ec3015f3700081087000000350102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e99880000000000000000000000000000000000000000000000004563918244f40000");
        } catch (e) {
            assert.equal(e.reason, "VAA was already executed")
            return
        }
        assert.fail("did not fail")
    });

    it("should burn tokens on lock", async function () {
        let bridge = await Wormhole.deployed();
        // Expect user to have a balance
        let wa = new WrappedAsset("0x79183957Be84C0F4dA451E534d5bA5BA3FB9c696")

        await bridge.lockAssets(wa.address, "4000000000000000000", "0x0", 2);
        let balance = await wa.balanceOf("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1");

        // Expect user balance to decrease
        assert.equal(balance, "1000000000000000000");

        // Expect contract balance to be 0 since tokens have been burned
        balance = await wa.balanceOf(bridge.address);
        assert.equal(balance, "0");
    });

    it("should transfer tokens in and out", async function () {
        let bridge = await Wormhole.deployed();
        let token = await ERC20.new("Test Token", "TKN");

        await token.mint("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1", "1000000000000000000");
        // Expect user to have a balance
        assert.equal(await token.balanceOf("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"), "1000000000000000000");

        // Approve bridge
        await token.approve(bridge.address, "1000000000000000000");

        // Transfer of that token out of the contract should not work
        let threw = false;
        try {
            await bridge.submitVAA("0x01000000000100f0c5e4e6087c6af17ce51d6e51842a766834e252266fcccd9ad39222a262af4725ff3cd3d954fca7b9964c09f0290dfacefdcaa441f62b5128ec10dce888c0cc005f37017a1087000000350102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1020000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a70000000000000000000000000000000000000000000000000de0b6b3a7640000");
        } catch (e) {
            threw = true;
        }
        assert.isTrue(threw);

        // Lock assets
        let ev = await bridge.lockAssets(token.address, "1000000000000000000", "0x1230000000000000000000000000000000000000000000000000000000000000", 3);

        // Check that the lock event was emitted correctly
        assert.lengthOf(ev.logs, 1)
        assert.equal(ev.logs[0].event, "LogTokensLocked")
        assert.equal(ev.logs[0].args.target_chain, "3")
        assert.equal(ev.logs[0].args.token_chain, "2")
        assert.equal(ev.logs[0].args.token, "0x0000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a7")
        assert.equal(ev.logs[0].args.sender, "0x00000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1")
        assert.equal(ev.logs[0].args.recipient, "0x1230000000000000000000000000000000000000000000000000000000000000")
        assert.equal(ev.logs[0].args.amount, "1000000000000000000")

        // Check that the tokens were transferred to the bridge
        assert.equal(await token.balanceOf("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"), "0");
        assert.equal(await token.balanceOf(bridge.address), "1000000000000000000");

        // Transfer this token back
        await bridge.submitVAA("0x01000000000100f0c5e4e6087c6af17ce51d6e51842a766834e252266fcccd9ad39222a262af4725ff3cd3d954fca7b9964c09f0290dfacefdcaa441f62b5128ec10dce888c0cc005f37017a1087000000350102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1020000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a70000000000000000000000000000000000000000000000000de0b6b3a7640000");
        assert.equal(await token.balanceOf("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"), "1000000000000000000");
        assert.equal(await token.balanceOf(bridge.address), "0");
    });

    it("should accept validator set change", async function () {
        let bridge = await Wormhole.deployed();

        // Push time by 1000
        await advanceTimeAndBlock(1000);
        let ev = await bridge.submitVAA("0x010000000001003382c71a4c79e1518a6ce29c91569f6427a60a95696a3515b8c2340b6acffd723315bd1011aa779f22573882a4edfe1b8206548e134871a23f8ba0c1c7d0b5ed0100000bb801190000000101befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe")
        assert.lengthOf(ev.logs, 1)
        assert.equal(ev.logs[0].event, "LogGuardianSetChanged")

        // Expect guardian set to transition to 1
        assert.equal(await bridge.guardian_set_index(), 1);
    });

    it("should not accept guardian set change from old guardians", async function () {
        let bridge = await Wormhole.deployed();

        // Test update guardian set VAA from guardian set 0; timestamp 2000
        let threw = false;
        try {
            await bridge.submitVAA("0x01000000000100686c37a81f0895d0db88c5c348bba8df53dedd579116327c999dc0229157c04e0304f9f8223b4e7b538ccf140de112d456d88e040bce025c1022bb840acb88390100000bb801190000000201befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe")
        } catch (e) {
            threw = true;
            assert.equal(e.reason, "only the current guardian set can change the guardian set")
        }
        assert.isTrue(threw, "old guardian set could make changes")
    });

    it("should time out guardians", async function () {
        let bridge = await Wormhole.deployed();

        // Test VAA from guardian set 0; timestamp 1000
        await bridge.submitVAA("0x01000000000100a60fd865ceee4cf34048fec8edc540f257d05c186d1ac6904d959d35ab2b6c0518feeb01fc3927b44d92746461d0ddb5ea0008de529b8a4862e18acf1fea364c00000003e81087000000360102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a70000000000000000000000000000000000000000000000000de0b6b3a7640000")

        await advanceTimeAndBlock(1000);

        // Test VAA from guardian set 0; timestamp 2000 - should not work anymore
        let threw = false;
        try {
            await bridge.submitVAA("0x010000000001002a17cefb8242bc6865d3e38abd764359fcb4cb774637d483aa8690a223b334217e75d1e808dcc6999fa73fabdf20d28455fe4c3abcf565db351456df418f0b7900000007d01087000000360102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a70000000000000000000000000000000000000000000000000de0b6b3a7640000")
        } catch (e) {
            threw = true;
            assert.equal(e.reason, "guardian set has expired")
        }
        assert.isTrue(threw, "guardian set did not expire")

        // Test same transaction with guardian set 1; timestamp 2000
        await bridge.submitVAA("0x010000000101005cae5dc08ebab209640fb5b8051261a5cff25bd84a69f93ec36a4106fde6a53e7275267596a4833607aae8ae9426b7bd10d8062f06c96dc9c820e30516e32e0400000007d01087000000350102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a70000000000000000000000000000000000000000000000000de0b6b3a7640000")
    });

    it("should expire VAA", async function () {
        let bridge = await Wormhole.deployed();

        // Push time by 1000
        await advanceTimeAndBlock(1000);

        // Test same transaction with guardian set 1; timestamp 2000
        let threw = false;
        try {
            await bridge.submitVAA("0x01000000010100f69b3f6e31fbbe6ce9b9b1be8e8effded63b44ab8d7d2dc993c914d50d4bb6fe75cdf6ebb15e5bf209f2ea608e496283d8ff5a91a102f1cab42e9093cbb50b6201000007d01087000000360102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a70000000000000000000000000000000000000000000000000de0b6b3a7640000")
        } catch (e) {
            threw = true;
            assert.equal(e.reason, "VAA has expired")
        }
        assert.isTrue(threw, "VAA did not expire")
    });


    it("mismatching guardian set and signature should not work", async function () {
        let bridge = await Wormhole.deployed();

        // Test VAA signed by guardian set 0 but set guardian set index to 1
        let threw = false;
        try {
            await bridge.submitVAA("0x010000000101006f84df72f3f935543e9bda60d92f77e2e2c073655311f3fc00518bbe7e054ff87e5e6e3c9df9e5bd756ee033253d4513ddebf03ff844fdc0f48f7dcc1b3fd6e10000000fa01087000000370102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a70000000000000000000000000000000000000000000000000de0b6b3a7640000")
        } catch (e) {
            threw = true;
            assert.equal(e.reason, "VAA signature invalid")
        }
        assert.isTrue(threw, "invalid signature accepted")
    });

    it("quorum should be honored", async function () {
        let bridge = await Wormhole.deployed();

        // Update to validator set 2 with 6 signers
        await bridge.submitVAA("0x010000000101006ec1d2ab1b9c24fecfc43265366038ea06d465c422cb92348d757436846fe908068e92d0bdca740c583a717da7cd525e46d80b0b945a51baae72007e456b8a240100001388017d00000002067e5f4552091a69125d5dfcb7b8c2659029395bdfbefa429d57cd18b7f8a4d91a2da9ab4af05d0fbebefa429d57cd18b7f8a4d91a2da9ab4af05d0fbebefa429d57cd18b7f8a4d91a2da9ab4af05d0fbebefa429d57cd18b7f8a4d91a2da9ab4af05d0fbebefa429d57cd18b7f8a4d91a2da9ab4af05d0fbe")

        // Test VAA signed by only 3 signers
        let threw = false;
        try {
            await bridge.submitVAA("0x01000000020300d943f7e2f94fdb2d23d8ce270c1b981ce8058a94c46358e4fb486f9b80c685d234fe7fee41c054c5ed5aa26548e39bc23875549e72100b02aabe1e9bd4c7d9b601018ce763f32e07d8d2d69907575425ad9ffbda4be1917a64ea2f90172ae8212e2c2c04917956ab5a72b6ef4642f1673e28465567de3ad3197d1a773e28c03c40c100028ce763f32e07d8d2d69907575425ad9ffbda4be1917a64ea2f90172ae8212e2c2c04917956ab5a72b6ef4642f1673e28465567de3ad3197d1a773e28c03c40c10000000fa01087000000380102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a70000000000000000000000000000000000000000000000000de0b6b3a7640000")
        } catch (e) {
            threw = true;
            assert.equal(e.reason, "no quorum")
        }
        assert.isTrue(threw, "accepted only 3 signatures")

        // Test VAA signed by 5 signers (all except i=3)
        await bridge.submitVAA("0x01000000020500d943f7e2f94fdb2d23d8ce270c1b981ce8058a94c46358e4fb486f9b80c685d234fe7fee41c054c5ed5aa26548e39bc23875549e72100b02aabe1e9bd4c7d9b601018ce763f32e07d8d2d69907575425ad9ffbda4be1917a64ea2f90172ae8212e2c2c04917956ab5a72b6ef4642f1673e28465567de3ad3197d1a773e28c03c40c100028ce763f32e07d8d2d69907575425ad9ffbda4be1917a64ea2f90172ae8212e2c2c04917956ab5a72b6ef4642f1673e28465567de3ad3197d1a773e28c03c40c100048ce763f32e07d8d2d69907575425ad9ffbda4be1917a64ea2f90172ae8212e2c2c04917956ab5a72b6ef4642f1673e28465567de3ad3197d1a773e28c03c40c100058ce763f32e07d8d2d69907575425ad9ffbda4be1917a64ea2f90172ae8212e2c2c04917956ab5a72b6ef4642f1673e28465567de3ad3197d1a773e28c03c40c10000000fa01087000000380102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a70000000000000000000000000000000000000000000000000de0b6b3a7640000")
    });
});
