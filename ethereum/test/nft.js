const jsonfile = require('jsonfile');
const elliptic = require('elliptic');
const BigNumber = require('bignumber.js');

const Wormhole = artifacts.require("Wormhole");
const NFTBridge = artifacts.require("NFTBridgeEntrypoint");
const NFTBridgeImplementation = artifacts.require("NFTBridgeImplementation");
const NFTImplementation = artifacts.require("NFTImplementation");
const MockBridgeImplementation = artifacts.require("MockNFTBridgeImplementation");

const testSigner1PK = "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";
const testSigner2PK = "892330666a850761e7370376430bb8c2aa1494072d3bfeaed0c4fa3d5a9135fe";

const WormholeImplementationFullABI = jsonfile.readFileSync("build/contracts/Implementation.json").abi
const BridgeImplementationFullABI = jsonfile.readFileSync("build/contracts/NFTBridgeImplementation.json").abi
const NFTImplementationFullABI = jsonfile.readFileSync("build/contracts/NFTImplementation.json").abi

contract("NFT", function () {
    const testSigner1 = web3.eth.accounts.privateKeyToAccount(testSigner1PK);
    const testSigner2 = web3.eth.accounts.privateKeyToAccount(testSigner2PK);
    const testChainId = "2";
    const testFinality = "1";
    const testGovernanceChainId = "1";
    const testGovernanceContract = "0x0000000000000000000000000000000000000000000000000000000000000004";
    let WETH = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
    const testForeignChainId = "1";
    const testForeignBridgeContract = "0x000000000000000000000000000000000000000000000000000000000000ffff";
    const testBridgedAssetChain = "0003";
    const testBridgedAssetAddress = "000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e";


    it("should be initialized with the correct signers and values", async function () {
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);

        const tokenImplentation = await initialized.methods.tokenImplementation().call();
        assert.equal(tokenImplentation, NFTImplementation.address);

        // test beacon functionality
        const beaconImplementation = await initialized.methods.implementation().call();
        assert.equal(beaconImplementation, NFTImplementation.address);

        // chain id
        const chainId = await initialized.methods.chainId().call();
        assert.equal(chainId, testChainId);

        // finality
        const finality = await initialized.methods.finality().call();
        assert.equal(finality, testFinality);

        // governance
        const governanceChainId = await initialized.methods.governanceChainId().call();
        assert.equal(governanceChainId, testGovernanceChainId);
        const governanceContract = await initialized.methods.governanceContract().call();
        assert.equal(governanceContract, testGovernanceContract);
    })

    it("should register a foreign bridge implementation correctly", async function () {
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);
        const accounts = await web3.eth.getAccounts();

        let data = [
            "0x",
            "00000000000000000000000000000000000000000000004e4654427269646765",
            "01",
            "0000",
            web3.eth.abi.encodeParameter("uint16", testForeignChainId).substring(2 + (64 - 4)),
            web3.eth.abi.encodeParameter("bytes32", testForeignBridgeContract).substring(2),
        ].join('')

        const vm = await signAndEncodeVM(
            1,
            1,
            testGovernanceChainId,
            testGovernanceContract,
            0,
            data,
            [
                testSigner1PK
            ],
            0,
            0
        );


        let before = await initialized.methods.bridgeContracts(testForeignChainId).call();

        assert.equal(before, "0x0000000000000000000000000000000000000000000000000000000000000000");

        await initialized.methods.registerChain("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        let after = await initialized.methods.bridgeContracts(testForeignChainId).call();

        assert.equal(after, testForeignBridgeContract);
    })

    it("should accept a valid upgrade", async function () {
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);
        const accounts = await web3.eth.getAccounts();

        const mock = await MockBridgeImplementation.new();

        let data = [
            "0x",
            "00000000000000000000000000000000000000000000004e4654427269646765",
            "02",
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            web3.eth.abi.encodeParameter("address", mock.address).substring(2),
        ].join('')

        const vm = await signAndEncodeVM(
            1,
            1,
            testGovernanceChainId,
            testGovernanceContract,
            0,
            data,
            [
                testSigner1PK
            ],
            0,
            0
        );

        let before = await web3.eth.getStorageAt(NFTBridge.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(before.toLowerCase(), NFTBridgeImplementation.address.toLowerCase());

        await initialized.methods.upgrade("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        let after = await web3.eth.getStorageAt(NFTBridge.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(after.toLowerCase(), mock.address.toLowerCase());

        const mockImpl = new web3.eth.Contract(MockBridgeImplementation.abi, NFTBridge.address);

        let isUpgraded = await mockImpl.methods.testNewImplementationActive().call();

        assert.ok(isUpgraded);
    })

    it("bridged tokens should only be mint- and burn-able by owner", async function () {
        const accounts = await web3.eth.getAccounts();

        // initialize our template token contract
        const token = new web3.eth.Contract(NFTImplementation.abi, NFTImplementation.address);

        await token.methods.initialize(
            "TestToken",
            "TT",
            accounts[0],

            0,
            "0x0"
        ).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        await token.methods.mint(accounts[0], 10, "").send({
            from: accounts[0],
            gasLimit: 2000000
        });

        let failed = false
        try {
            await token.methods.mint(accounts[0], 11, "").send({
                from: accounts[1],
                gasLimit: 2000000
            });
        } catch (e) {
            failed = true
        }
        assert.ok(failed)

        failed = false
        try {
            await token.methods.burn(10).send({
                from: accounts[1],
                gasLimit: 2000000
            });
        } catch (e) {
            failed = true
        }
        assert.ok(failed)

        await token.methods.burn(10).send({
            from: accounts[0],
            gasLimit: 2000000
        });
    })

    it("should deposit and log transfers correctly", async function () {
        const accounts = await web3.eth.getAccounts();
        const tokenId = "1000000000000000000";

        // mint and approve tokens
        const token = new web3.eth.Contract(NFTImplementation.abi, NFTImplementation.address);
        await token.methods.mint(accounts[0], tokenId, "abcd").send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });
        await token.methods.approve(NFTBridge.address, tokenId).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // deposit tokens
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);

        const ownerBefore = await token.methods.ownerOf(tokenId).call();
        assert.equal(ownerBefore, accounts[0]);
        await initialized.methods.transferNFT(
            NFTImplementation.address,
            tokenId,
            "10",
            "0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e",
            "234"
        ).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        const ownerAfter = await token.methods.ownerOf(tokenId).call();
        assert.equal(ownerAfter, NFTBridge.address);

        // check transfer log
        const wormhole = new web3.eth.Contract(WormholeImplementationFullABI, Wormhole.address);
        const log = (await wormhole.getPastEvents('LogMessagePublished', {
            fromBlock: 'latest'
        }))[0].returnValues

        assert.equal(log.sender, NFTBridge.address)

        assert.equal(log.payload.length - 2, 340);

        // payload id
        assert.equal(log.payload.substr(2, 2), "01");

        // token
        assert.equal(log.payload.substr(4, 64), web3.eth.abi.encodeParameter("address", NFTImplementation.address).substring(2));

        // chain id
        assert.equal(log.payload.substr(68, 4), web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + 64 - 4))

        // symbol (TT)
        assert.equal(log.payload.substr(72, 64), "5454000000000000000000000000000000000000000000000000000000000000")

        // name (TestToken (Wormhole))
        assert.equal(log.payload.substr(136, 64), "54657374546f6b656e0000000000000000000000000000000000000000000000")

        // tokenID
        assert.equal(log.payload.substr(200, 64), web3.eth.abi.encodeParameter("uint256", new BigNumber(tokenId).toString()).substring(2));

        // url length
        assert.equal(log.payload.substr(264, 2), web3.eth.abi.encodeParameter("uint8", 4).substring(2 + 64 - 2))

        // url
        assert.equal(log.payload.substr(266, 8), "61626364")

        // to
        assert.equal(log.payload.substr(274, 64), "000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e");

        // to chain id
        assert.equal(log.payload.substr(338, 4), web3.eth.abi.encodeParameter("uint16", 10).substring(2 + 64 - 4))
    })

    it("should transfer out locked assets for a valid transfer vm", async function () {
        const accounts = await web3.eth.getAccounts();
        const tokenId = "1000000000000000000";

        const token = new web3.eth.Contract(NFTImplementation.abi, NFTImplementation.address);
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);

        const ownerBefore = await token.methods.ownerOf(tokenId).call();
        assert.equal(ownerBefore, NFTBridge.address);

        // PayloadID uint8 = 1
        // // Address of the NFT. Left-zero-padded if shorter than 32 bytes
        // NFTAddress [32]uint8
        // // Chain ID of the NFT
        // NFTChain uint16
        // // Name of the NFT
        // Name [32]uint8
        // // Symbol of the NFT
        // Symbol [10]uint8
        // // ID of the token (big-endian uint256)
        // TokenID [32]uint8
        // // URL of the NFT
        // URLLength u8
        // URL [n]uint8
        // // Address of the recipient. Left-zero-padded if shorter than 32 bytes
        // To [32]uint8
        // // Chain ID of the recipient
        // ToChain uint16
        const data = "0x" +
            "01" +
            // tokenaddress
            web3.eth.abi.encodeParameter("address", NFTImplementation.address).substr(2) +
            // tokenchain
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)) +
            // symbol
            "0000000000000000000000000000000000000000000000000000000000000000" +
            // name
            "0000000000000000000000000000000000000000000000000000000000000000" +
            // tokenID
            web3.eth.abi.encodeParameter("uint256", new BigNumber(tokenId).toString()).substring(2) +
            // url length
            "00" +
            // no URL
            "" +
            // receiver
            web3.eth.abi.encodeParameter("address", accounts[0]).substr(2) +
            // receiving chain
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4));

        const vm = await signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            0,
            data,
            [
                testSigner1PK
            ],
            0,
            0
        );

        await initialized.methods.completeTransfer("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        const ownerAfter = await token.methods.ownerOf(tokenId).call();
        assert.equal(ownerAfter, accounts[0]);
    })

    it("should mint bridged assets wrappers on transfer from another chain and handle fees correctly", async function () {
        const accounts = await web3.eth.getAccounts();
        let tokenId = "1000000000000000001";

        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);

        // we are using the asset where we created a wrapper in the previous test
        let data = "0x" +
            "01" +
            // tokenaddress
            testBridgedAssetAddress +
            // tokenchain
            testBridgedAssetChain +
            // symbol
            "464f520000000000000000000000000000000000000000000000000000000000" +
            // name
            "466f726569676e20436861696e204e4654000000000000000000000000000000" +
            // tokenID
            web3.eth.abi.encodeParameter("uint256", new BigNumber(tokenId).toString()).substring(2) +
            // url length
            "00" +
            // no URL
            "" +
            // receiver
            web3.eth.abi.encodeParameter("address", accounts[0]).substr(2) +
            // receiving chain
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4));

        let vm = await signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            0,
            data,
            [
                testSigner1PK
            ],
            0,
            0
        );

        await initialized.methods.completeTransfer("0x" + vm).send({
            value: 0,
            from: accounts[1],
            gasLimit: 2000000
        });

        const wrappedAddress = await initialized.methods.wrappedAsset("0x" + testBridgedAssetChain, "0x" + testBridgedAssetAddress).call();
        assert.ok(await initialized.methods.isWrappedAsset(wrappedAddress).call())
        const wrappedAsset = new web3.eth.Contract(NFTImplementation.abi, wrappedAddress);

        let ownerAfter = await wrappedAsset.methods.ownerOf(tokenId).call();
        assert.equal(ownerAfter, accounts[0]);

        const symbol = await wrappedAsset.methods.symbol().call();
        assert.equal(symbol, "FOR");

        const name = await wrappedAsset.methods.name().call();
        assert.equal(name, "Foreign Chain NFT");

        const chainId = await wrappedAsset.methods.chainId().call();
        assert.equal(chainId, Number(testBridgedAssetChain));

        const nativeContract = await wrappedAsset.methods.nativeContract().call();
        assert.equal(nativeContract, "0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e");

        // Transfer another tokenID of the same token address
        tokenId = "1000000000000000002"
        data = "0x" +
            "01" +
            // tokenaddress
            testBridgedAssetAddress +
            // tokenchain
            testBridgedAssetChain +
            // symbol
            "464f520000000000000000000000000000000000000000000000000000000000" +
            // name
            "466f726569676e20436861696e204e4654000000000000000000000000000000" +
            // tokenID
            web3.eth.abi.encodeParameter("uint256", new BigNumber(tokenId + 1).toString()).substring(2) +
            // url length
            "00" +
            // no URL
            "" +
            // receiver
            web3.eth.abi.encodeParameter("address", accounts[0]).substr(2) +
            // receiving chain
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4));

        vm = await signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            1,
            data,
            [
                testSigner1PK
            ],
            0,
            0
        );

        await initialized.methods.completeTransfer("0x" + vm).send({
            value: 0,
            from: accounts[1],
            gasLimit: 2000000
        });

        ownerAfter = await wrappedAsset.methods.ownerOf(tokenId + 1).call();
        assert.equal(ownerAfter, accounts[0]);
    })

    it("should mint bridged assets from solana under unified name, caching the original", async function () {
        const accounts = await web3.eth.getAccounts();
        let tokenId = "1000000000000000001";

        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);

        // we are using the asset where we created a wrapper in the previous test
        let data = "0x" +
            "01" +
            // tokenaddress
            testBridgedAssetAddress +
            // tokenchain
            "0001" +
            // symbol
            "464f520000000000000000000000000000000000000000000000000000000000" +
            // name
            "466f726569676e20436861696e204e4654000000000000000000000000000000" +
            // tokenID
            web3.eth.abi.encodeParameter("uint256", new BigNumber(tokenId).toString()).substring(2) +
            // url length
            "00" +
            // no URL
            "" +
            // receiver
            web3.eth.abi.encodeParameter("address", accounts[0]).substr(2) +
            // receiving chain
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4));

        let vm = await signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            0,
            data,
            [
                testSigner1PK
            ],
            0,
            0
        );

        await initialized.methods.completeTransfer("0x" + vm).send({
            value: 0,
            from: accounts[1],
            gasLimit: 2000000
        });

        const cache = await initialized.methods.splCache(tokenId).call()
        assert.equal(cache.symbol, "0x464f520000000000000000000000000000000000000000000000000000000000");
        assert.equal(cache.name, "0x466f726569676e20436861696e204e4654000000000000000000000000000000");

        const wrappedAddress = await initialized.methods.wrappedAsset("0x0001", "0x" + testBridgedAssetAddress).call();
        const wrappedAsset = new web3.eth.Contract(NFTImplementation.abi, wrappedAddress);

        const symbol = await wrappedAsset.methods.symbol().call();
        assert.equal(symbol, "WORMSPLNFT");

        const name = await wrappedAsset.methods.name().call();
        assert.equal(name, "Wormhole Bridged Solana-NFT");
    })

    it("cached SPL names are loaded when transferring out, cache is cleared", async function () {
        const accounts = await web3.eth.getAccounts();
        let tokenId = "1000000000000000001";

        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);

        const wrappedAddress = await initialized.methods.wrappedAsset("0x0001", "0x" + testBridgedAssetAddress).call();
        const wrappedAsset = new web3.eth.Contract(NFTImplementation.abi, wrappedAddress);

        await wrappedAsset.methods.approve(NFTBridge.address, tokenId).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        const transfer = await initialized.methods.transferNFT(
            wrappedAddress,
            tokenId,
            "10",
            "0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e",
            "2345"
        ).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // symbol
        assert.ok(transfer.events[4].raw.data.includes('464f520000000000000000000000000000000000000000000000000000000000'))
        // name
        assert.ok(transfer.events[4].raw.data.includes('466f726569676e20436861696e204e4654000000000000000000000000000000'))

        // check if cache is cleared
        const cache = await initialized.methods.splCache(tokenId).call()
        assert.equal(cache.symbol, "0x0000000000000000000000000000000000000000000000000000000000000000");
        assert.equal(cache.name, "0x0000000000000000000000000000000000000000000000000000000000000000");
    })

    it("should should fail deposit unapproved NFTs", async function () {
        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);
        const tokenId = "1000000000000000001";

        const wrappedAddress = await initialized.methods.wrappedAsset("0x" + testBridgedAssetChain, "0x" + testBridgedAssetAddress).call();

        // deposit tokens
        let failed = false
        try {
            await initialized.methods.transferNFT(
                wrappedAddress,
                tokenId,
                "10",
                "0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e",
                "234"
            ).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (e) {
            assert.equal(e.message, "Returned error: VM Exception while processing transaction: revert ERC721: transfer caller is not owner nor approved")
            failed = true
        }

        assert.ok(failed)
    })

    it("should refuse to burn wrappers not held by msg.sender", async function () {
        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);
        const tokenId = "1000000000000000001";

        const wrappedAddress = await initialized.methods.wrappedAsset("0x" + testBridgedAssetChain, "0x" + testBridgedAssetAddress).call();
        const wrappedAsset = new web3.eth.Contract(NFTImplementation.abi, wrappedAddress);

        // approve from 0
        await wrappedAsset.methods.approve(NFTBridge.address, tokenId).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // deposit tokens from 1
        let failed = false
        try {
            await initialized.methods.transferNFT(
                wrappedAddress,
                tokenId,
                "10",
                "0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e",
                "234"
            ).send({
                value: 0,
                from: accounts[1],
                gasLimit: 2000000
            });
        } catch (e) {
            assert.equal(e.message, "Returned error: VM Exception while processing transaction: revert ERC721: transfer of token that is not own")
            failed = true
        }

        assert.ok(failed)
    })

    it("should deposit and burn approved bridged assets wrappers on transfer to another chain", async function () {
        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, NFTBridge.address);
        const tokenId = "1000000000000000001";

        const wrappedAddress = await initialized.methods.wrappedAsset("0x" + testBridgedAssetChain, "0x" + testBridgedAssetAddress).call();
        const wrappedAsset = new web3.eth.Contract(NFTImplementation.abi, wrappedAddress);

        await wrappedAsset.methods.approve(NFTBridge.address, tokenId).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // deposit tokens

        const ownerBefore = await wrappedAsset.methods.ownerOf(tokenId).call();

        assert.equal(ownerBefore, accounts[0]);

        await initialized.methods.transferNFT(
            wrappedAddress,
            tokenId,
            "10",
            "0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e",
            "234"
        ).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        try {
            await wrappedAsset.methods.ownerOf(tokenId).call();
            assert.fail("burned token still exists")
        } catch (e) {
            assert.equal(e.data[Object.keys(e.data)[0]].reason, "ERC721: owner query for nonexistent token")
        }
    })
});

const signAndEncodeVM = async function (
    timestamp,
    nonce,
    emitterChainId,
    emitterAddress,
    sequence,
    data,
    signers,
    guardianSetIndex,
    consistencyLevel
) {
    const body = [
        web3.eth.abi.encodeParameter("uint32", timestamp).substring(2 + (64 - 8)),
        web3.eth.abi.encodeParameter("uint32", nonce).substring(2 + (64 - 8)),
        web3.eth.abi.encodeParameter("uint16", emitterChainId).substring(2 + (64 - 4)),
        web3.eth.abi.encodeParameter("bytes32", emitterAddress).substring(2),
        web3.eth.abi.encodeParameter("uint64", sequence).substring(2 + (64 - 16)),
        web3.eth.abi.encodeParameter("uint8", consistencyLevel).substring(2 + (64 - 2)),
        data.substr(2)
    ]

    const hash = web3.utils.soliditySha3(web3.utils.soliditySha3("0x" + body.join("")))

    let signatures = "";

    for (let i in signers) {
        const ec = new elliptic.ec("secp256k1");
        const key = ec.keyFromPrivate(signers[i]);
        const signature = key.sign(hash.substr(2), {canonical: true});

        const packSig = [
            web3.eth.abi.encodeParameter("uint8", i).substring(2 + (64 - 2)),
            zeroPadBytes(signature.r.toString(16), 32),
            zeroPadBytes(signature.s.toString(16), 32),
            web3.eth.abi.encodeParameter("uint8", signature.recoveryParam).substr(2 + (64 - 2)),
        ]

        signatures += packSig.join("")
    }

    const vm = [
        web3.eth.abi.encodeParameter("uint8", 1).substring(2 + (64 - 2)),
        web3.eth.abi.encodeParameter("uint32", guardianSetIndex).substring(2 + (64 - 8)),
        web3.eth.abi.encodeParameter("uint8", signers.length).substring(2 + (64 - 2)),

        signatures,
        body.join("")
    ].join("");

    return vm
}

function zeroPadBytes(value, length) {
    while (value.length < 2 * length) {
        value = "0" + value;
    }
    return value;
}
