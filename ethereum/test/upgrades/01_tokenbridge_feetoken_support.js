const jsonfile = require('jsonfile');
const elliptic = require('elliptic');
const BigNumber = require('bignumber.js');

const Wormhole = artifacts.require("Wormhole");
const TokenBridge = artifacts.require("TokenBridge");
const BridgeSetup = artifacts.require("BridgeSetup");
const BridgeImplementation = artifacts.require("BridgeImplementation");
const MockBridgeImplementation = artifacts.require("MockBridgeImplementation");
const TokenImplementation = artifacts.require("TokenImplementation");
const FeeToken = artifacts.require("FeeToken");

const testSigner1PK = "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";

const WormholeImplementationFullABI = jsonfile.readFileSync("build/contracts/Implementation.json").abi
const BridgeImplementationFullABI = jsonfile.readFileSync("build/contracts/BridgeImplementation.json").abi

// needs to run on a mainnet fork

contract("Update Bridge", function (accounts) {
    const testChainId = "2";
    const testGovernanceChainId = "1";
    const testGovernanceContract = "0x0000000000000000000000000000000000000000000000000000000000000004";
    let WETH = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
    const testForeignChainId = "1";
    const testForeignBridgeContract = "0x000000000000000000000000000000000000000000000000000000000000ffff";

    const currentImplementation = "0x6c4c12987303b2c94b2C76c612Fc5F4D2F0360F7";
    let bridgeProxy;

    it("create bridge instance with current implementation", async function () {
        // encode initialisation data
        const setup = new web3.eth.Contract(BridgeSetup.abi, BridgeSetup.address);
        const initData = setup.methods.setup(
            currentImplementation,
            testChainId,
            (await Wormhole.deployed()).address,
            testGovernanceChainId,
            testGovernanceContract,
            TokenImplementation.address,
            WETH
        ).encodeABI();

        const deploy = await TokenBridge.new(BridgeSetup.address, initData);

        bridgeProxy = new web3.eth.Contract(BridgeImplementationFullABI, deploy.address);
    })

    it("register a foreign bridge implementation", async function () {
        let data = [
            "0x",
            "000000000000000000000000000000000000000000546f6b656e427269646765",
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


        let before = await bridgeProxy.methods.bridgeContracts(testForeignChainId).call();

        assert.equal(before, "0x0000000000000000000000000000000000000000000000000000000000000000");

        await bridgeProxy.methods.registerChain("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        let after = await bridgeProxy.methods.bridgeContracts(testForeignChainId).call();

        assert.equal(after, testForeignBridgeContract);
    })

    it("mimic previous deposits (deposit some ETH)", async function () {
        const amount = "100000000000000000";
        const fee = "10000000000000000";

        await bridgeProxy.methods.wrapAndTransferETH(
            "10",
            "0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e",
            fee,
            "234"
        ).send({
            value: amount,
            from: accounts[0],
            gasLimit: 2000000
        });

        // check transfer log
        const wormhole = new web3.eth.Contract(WormholeImplementationFullABI, Wormhole.address);
        const log = (await wormhole.getPastEvents('LogMessagePublished', {
            fromBlock: 'latest'
        }))[0].returnValues

        assert.equal(log.payload.length - 2, 266);

        // payload id
        assert.equal(log.payload.substr(2, 2), "01");

        // amount
        assert.equal(log.payload.substr(4, 64), web3.eth.abi.encodeParameter("uint256", new BigNumber(amount).div(1e10).toString()).substring(2));

        // token
        assert.equal(log.payload.substr(68, 64), web3.eth.abi.encodeParameter("address", WETH).substring(2));

        // chain id
        assert.equal(log.payload.substr(132, 4), web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + 64 - 4))

        // to
        assert.equal(log.payload.substr(136, 64), "000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e");

        // to chain id
        assert.equal(log.payload.substr(200, 4), web3.eth.abi.encodeParameter("uint16", 10).substring(2 + 64 - 4))

        // fee
        assert.equal(log.payload.substr(204, 64), web3.eth.abi.encodeParameter("uint256", new BigNumber(fee).div(1e10).toString()).substring(2))
    })

    let upgradeDeployedAt;
    it("apply upgrade", async function () {
        const deploy = await BridgeImplementation.new();
        upgradeDeployedAt = deploy.address;

        let data = [
            "0x",
            "000000000000000000000000000000000000000000546f6b656e427269646765",
            "02",
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            web3.eth.abi.encodeParameter("address", deploy.address).substring(2),
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

        let before = await web3.eth.getStorageAt(bridgeProxy.options.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(before.toLowerCase(), currentImplementation.toLowerCase());

        await bridgeProxy.methods.upgrade("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        let after = await web3.eth.getStorageAt(bridgeProxy.options.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(after.toLowerCase(), deploy.address.toLowerCase());
    })

    it("test withdrawing existing assets (deposited ETH)", async function () {
        const amount = "100000000000000000";

        const accountBalanceBefore = await web3.eth.getBalance(accounts[1]);

        // we are using the asset where we created a wrapper in the previous test
        const data = "0x" +
            "01" +
            // amount
            web3.eth.abi.encodeParameter("uint256", new BigNumber(amount).div(1e10).toString()).substring(2) +
            // tokenaddress
            web3.eth.abi.encodeParameter("address", WETH).substr(2) +
            // tokenchain
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)) +
            // receiver
            web3.eth.abi.encodeParameter("address", accounts[1]).substr(2) +
            // receiving chain
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)) +
            // fee
            web3.eth.abi.encodeParameter("uint256", 0).substring(2);

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

        const transferTX = await bridgeProxy.methods.completeTransferAndUnwrapETH("0x" + vm).send({
            from: accounts[0],
            gasLimit: 2000000
        });

        const accountBalanceAfter = await web3.eth.getBalance(accounts[1]);

        assert.equal((new BigNumber(accountBalanceAfter)).minus(accountBalanceBefore).toString(10), (new BigNumber(amount)).toString(10))
    })

    it("test new functionality (fee token transfers)", async function () {
        const accounts = await web3.eth.getAccounts();
        const mintAmount = "10000000000000000000";
        const amount = "1000000000000000000";
        const fee = "100000000000000000";

        // mint and approve tokens
        const deployFeeToken = await FeeToken.new();
        const token = new web3.eth.Contract(FeeToken.abi, deployFeeToken.address);
        await token.methods.initialize(
            "Test",
            "TST",
            "18",
            "123",
            accounts[0],
            "0",
            "0x0000000000000000000000000000000000000000000000000000000000000000"
        ).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });
        await token.methods.mint(accounts[0], mintAmount).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });
        await token.methods.approve(bridgeProxy.options.address, mintAmount).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        const bridgeBalanceBefore = await token.methods.balanceOf(bridgeProxy.options.address).call();

        assert.equal(bridgeBalanceBefore.toString(10), "0");

        await bridgeProxy.methods.transferTokens(
            deployFeeToken.address,
            amount,
            "10",
            "0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e",
            fee,
            "234"
        ).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        const bridgeBalanceAfter = await token.methods.balanceOf(bridgeProxy.options.address).call();

        let feeAmount = new BigNumber(amount).times(9).div(10)

        assert.equal(bridgeBalanceAfter.toString(10), feeAmount);

        // check transfer log
        const wormhole = new web3.eth.Contract(WormholeImplementationFullABI, Wormhole.address);
        const log = (await wormhole.getPastEvents('LogMessagePublished', {
            fromBlock: 'latest'
        }))[0].returnValues

        assert.equal(log.sender, bridgeProxy.options.address)

        assert.equal(log.payload.length - 2, 266);

        // payload id
        assert.equal(log.payload.substr(2, 2), "01");

        // amount
        assert.equal(log.payload.substr(4, 64), web3.eth.abi.encodeParameter("uint256", feeAmount.div(1e10).toString()).substring(2));

        // token
        assert.equal(log.payload.substr(68, 64), web3.eth.abi.encodeParameter("address", deployFeeToken.address).substring(2));

        // chain id
        assert.equal(log.payload.substr(132, 4), web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + 64 - 4))

        // to
        assert.equal(log.payload.substr(136, 64), "000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e");

        // to chain id
        assert.equal(log.payload.substr(200, 4), web3.eth.abi.encodeParameter("uint16", 10).substring(2 + 64 - 4))

        // fee
        assert.equal(log.payload.substr(204, 64), web3.eth.abi.encodeParameter("uint256", new BigNumber(fee).div(1e10).toString()).substring(2))
    })

    it("should accept a further upgrade", async function () {
        const mock = await MockBridgeImplementation.new();

        let data = [
            "0x",
            "000000000000000000000000000000000000000000546f6b656e427269646765",
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

        let before = await web3.eth.getStorageAt(bridgeProxy.options.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(before.toLowerCase(), upgradeDeployedAt.toLowerCase());

        await bridgeProxy.methods.upgrade("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        let after = await web3.eth.getStorageAt(bridgeProxy.options.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(after.toLowerCase(), mock.address.toLowerCase());

        const mockImpl = new web3.eth.Contract(MockBridgeImplementation.abi, bridgeProxy.options.address);

        let isUpgraded = await mockImpl.methods.testNewImplementationActive().call();

        assert.ok(isUpgraded);
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