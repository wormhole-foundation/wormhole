const Schnorr = artifacts.require("Schnorr");
const Wormhole = artifacts.require("Wormhole");
const WrappedAsset = artifacts.require("WrappedAsset");
const ERC20 = artifacts.require("ERC20PresetMinterPauser");

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

        await bridge.submitVAA("0x0100000000008df1ef2b367213cf591e6f6a8de37dd5a4ca771590f6f964a2c4a63b44c1e8532c0e595f4e6e0e784314724c85038af6576de0000007d01087000000330102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e99880000000000000000000000000000000000000000000000004563918244f40000")
        // Expect user to have a balance of a new wrapped asset
        let wa = new WrappedAsset("0x79183957Be84C0F4dA451E534d5bA5BA3FB9c696");
        assert.equal(await wa.assetChain(), 1)
        assert.equal(await wa.assetAddress(), "0x0000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e9988")
        let balance = await wa.balanceOf("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1");
        assert.equal(balance, "5000000000000000000");
    });

    it("should not accept the same VAA twice", async function () {
        let bridge = await Wormhole.deployed();
        try {
            await bridge.submitVAA("0x0100000000008df1ef2b367213cf591e6f6a8de37dd5a4ca771590f6f964a2c4a63b44c1e8532c0e595f4e6e0e784314724c85038af6576de0000007d01087000000330102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e99880000000000000000000000000000000000000000000000004563918244f40000");
        } catch (e) {
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
            await bridge.submitVAA("0x0100000000636e71c9cb08d64b6388a39d28779fab9dd42edad20331d022c9e90a43b78b1bfc737f2973136230a9e323fbd5d2f7d6cb599c2bfffff82f1087000000310102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1020000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a70000000000000000000000000000000000000000000000000de0b6b3a7640000");
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
        await bridge.submitVAA("0x0100000000636e71c9cb08d64b6388a39d28779fab9dd42edad20331d022c9e90a43b78b1bfc737f2973136230a9e323fbd5d2f7d6cb599c2bfffff82f1087000000310102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1020000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a70000000000000000000000000000000000000000000000000de0b6b3a7640000");
        assert.equal(await token.balanceOf("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"), "1000000000000000000");
        assert.equal(await token.balanceOf(bridge.address), "0");
    });

    it("should accept validator set change", async function () {
        let bridge = await Wormhole.deployed();

        // Push time by 1000
        await advanceTimeAndBlock(1000);
        await bridge.submitVAA("0x0100000000fe60d5766a84300effedd5362dcf6ff8f4ed75ab3dbe4c1ae07151ab48bc8cbf767b4aa42cf768477dc5bb45367044bd2de6d6b3000003e801253e2f87d126ef42ac22d284de7619d2c87437198a32887efeddb4debfd016747f0000000001")
        // Expect user to have a balance of a new wrapped asset
        assert.equal(await bridge.guardian_set_index(), 1);
        assert.equal((await bridge.guardian_sets(1)).x, "28127375798693063422362909717576839343810687066240716944661469189277081826431");

        // Test VAA from guardian set 0; timestamp 1000
        await bridge.submitVAA("0x01000000004f871da18c25af540bf7ea0ef28df13ff8945903fa1b82aa5d11ff749f33dba57b6064666dfe07b627e5e1da1f4bf620f92c15c2000003e81087000000340102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e99880000000000000000000000000000000000000000000000004563918244f40000")

        await advanceTimeAndBlock(1000);

        // Test VAA from guardian set 0; timestamp 2000 - should not work anymore
        let threw = false;
        try {
            await bridge.submitVAA("0x01000000004629dc39ea4b284d31f9c7d5350013aeed4b1c38a80fc65fb21e6c7da5ebd0eb13b46039f40a0ddd7c94c3e974b51cacf9eaa1bb000007d01087000000340102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e99880000000000000000000000000000000000000000000000004563918244f40000")
        } catch (e) {
            threw = true;
            assert.equal(e.reason, "guardian set has expired")
        }
        assert.isTrue(threw, "guardian set did not expire")

        // Test same transaction with guardian set 1; timestamp 2000
        await bridge.submitVAA("0x01000000011322402df3ec812a145aa2d9b0f627ff3654c9b3ca471622a1439e81da62ec384ad14db65ae4bee55a23b8082628590902e3d778000007d01087000000340102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e99880000000000000000000000000000000000000000000000004563918244f40000")
    });

    it("should expire VAA", async function () {
        let bridge = await Wormhole.deployed();

        // Push time by 1000
        await advanceTimeAndBlock(1000);

        // Test same transaction with guardian set 1; timestamp 2000
        let threw = false;
        try {
            await bridge.submitVAA("0x01000000013faebdc02d6427d1e8d33919fbaa519ca402323723922c772e4e2da7fedc820c15b24aa5e4c99bec6a9f4c9b612970590ea3acd1000007d01087000000350102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e99880000000000000000000000000000000000000000000000004563918244f40000")
        } catch (e) {
            threw = true;
            assert.equal(e.reason, "VAA has expired")
        }
        assert.isTrue(threw, "VAA did not expire")
    });
});
