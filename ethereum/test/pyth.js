const jsonfile = require('jsonfile');
const elliptic = require('elliptic');
const BigNumber = require('bignumber.js');

const Wormhole = artifacts.require("Wormhole");
const PythDataBridge = artifacts.require("PythDataBridge");
const PythImplementation = artifacts.require("PythImplementation");
const MockPythImplementation = artifacts.require("MockPythImplementation");

const testSigner1PK = "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";
const testSigner2PK = "892330666a850761e7370376430bb8c2aa1494072d3bfeaed0c4fa3d5a9135fe";

const WormholeImplementationFullABI = jsonfile.readFileSync("build/contracts/Implementation.json").abi
const P2WImplementationFullABI = jsonfile.readFileSync("build/contracts/PythImplementation.json").abi

contract("Pyth", function () {
    const testSigner1 = web3.eth.accounts.privateKeyToAccount(testSigner1PK);
    const testSigner2 = web3.eth.accounts.privateKeyToAccount(testSigner2PK);
    const testChainId = "2";
    const testGovernanceChainId = "3";
    const testGovernanceContract = "0x0000000000000000000000000000000000000000000000000000000000000004";
    const testPyth2WormholeChainId = "5";
    const testPyth2WormholeEmitter = "0x0000000000000000000000000000000000000000000000000000000000000006";


    it("should be initialized with the correct signers and values", async function(){
        const initialized = new web3.eth.Contract(P2WImplementationFullABI, PythDataBridge.address);

        // chain id
        const chainId = await initialized.methods.chainId().call();
        assert.equal(chainId, testChainId);

        // governance
        const governanceChainId = await initialized.methods.governanceChainId().call();
        assert.equal(governanceChainId, testGovernanceChainId);
        const governanceContract = await initialized.methods.governanceContract().call();
        assert.equal(governanceContract, testGovernanceContract);

        // pyth2wormhole
        const pyth2wormChain = await initialized.methods.pyth2WormholeChainId().call();
        assert.equal(pyth2wormChain, testPyth2WormholeChainId);
        const pyth2wormEmitter = await initialized.methods.pyth2WormholeEmitter().call();
        assert.equal(pyth2wormEmitter, testPyth2WormholeEmitter);
    })

    it("should accept a valid upgrade", async function() {
        const initialized = new web3.eth.Contract(P2WImplementationFullABI, PythDataBridge.address);
        const accounts = await web3.eth.getAccounts();

        const mock = await MockPythImplementation.new();

        let data = [
            "0x0000000000000000000000000000000000000000000000000000000050797468",
            "01",
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

        let before = await web3.eth.getStorageAt(PythDataBridge.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(before.toLowerCase(), PythImplementation.address.toLowerCase());

        await initialized.methods.upgrade("0x" + vm).send({
            value : 0,
            from : accounts[0],
            gasLimit : 2000000
        });

        let after = await web3.eth.getStorageAt(PythDataBridge.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(after.toLowerCase(), mock.address.toLowerCase());

        const mockImpl = new web3.eth.Contract(MockPythImplementation.abi, PythDataBridge.address);

        let isUpgraded = await mockImpl.methods.testNewImplementationActive().call();

        assert.ok(isUpgraded);
    })

    let testUpdate = "0x"+
        "503257480001011515151515151515151515151515151515151515151515151515151515151515DEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDEDE01DEADBEEFDEADBABEFFFFFFFDFFFFFFFFFFFFFFD6000000000000000F0000000000000025000000000000002A000000000000045700000000000008AE0000000000000065010000000000075BCD15";

    it("should parse price update correctly", async function() {
        const initialized = new web3.eth.Contract(P2WImplementationFullABI, PythDataBridge.address);

        let parsed = await initialized.methods.parsePriceAttestation(testUpdate).call();

        assert.equal(parsed.magic, 1345476424);
        assert.equal(parsed.version, 1);
        assert.equal(parsed.payloadId, 1);
        assert.equal(parsed.productId, "0x1515151515151515151515151515151515151515151515151515151515151515");
        assert.equal(parsed.priceId, "0xdededededededededededededededededededededededededededededededede");
        assert.equal(parsed.priceType, 1);
        assert.equal(parsed.price, -2401053088876217666);
        assert.equal(parsed.exponent, -3);

        assert.equal(parsed.twap.value, -42);
        assert.equal(parsed.twap.numerator, 15);
        assert.equal(parsed.twap.denominator, 37);

        assert.equal(parsed.twac.value, 42);
        assert.equal(parsed.twac.numerator, 1111);
        assert.equal(parsed.twac.denominator, 2222);

        assert.equal(parsed.confidenceInterval, 101);

        assert.equal(parsed.status, 1);
        assert.equal(parsed.corpAct, 0);

        assert.equal(parsed.timestamp, 123456789);
    })

    it("should attest price updates over wormhole", async function() {
        const initialized = new web3.eth.Contract(P2WImplementationFullABI, PythDataBridge.address);
        const accounts = await web3.eth.getAccounts();

        const vm = await signAndEncodeVM(
            1,
            1,
            testPyth2WormholeChainId,
            testPyth2WormholeEmitter,
            0,
            testUpdate,
            [
                testSigner1PK
            ],
            0,
            0
        );

        let result = await initialized.methods.attestPrice("0x"+vm).send({
            value : 0,
            from : accounts[0],
            gasLimit : 2000000
        });
    })

    it("should cache price updates", async function() {
        const initialized = new web3.eth.Contract(P2WImplementationFullABI, PythDataBridge.address);

        let cached = await initialized.methods.latestAttestation("0x1515151515151515151515151515151515151515151515151515151515151515", 1).call();

        assert.equal(cached.magic, 1345476424);
        assert.equal(cached.version, 1);
        assert.equal(cached.payloadId, 1);
        assert.equal(cached.productId, "0x1515151515151515151515151515151515151515151515151515151515151515");
        assert.equal(cached.priceId, "0xdededededededededededededededededededededededededededededededede");
        assert.equal(cached.priceType, 1);
        assert.equal(cached.price, -2401053088876217666);
        assert.equal(cached.exponent, -3);

        assert.equal(cached.twap.value, -42);
        assert.equal(cached.twap.numerator, 15);
        assert.equal(cached.twap.denominator, 37);

        assert.equal(cached.twac.value, 42);
        assert.equal(cached.twac.numerator, 1111);
        assert.equal(cached.twac.denominator, 2222);

        assert.equal(cached.confidenceInterval, 101);

        assert.equal(cached.status, 1);
        assert.equal(cached.corpAct, 0);

        assert.equal(cached.timestamp, 123456789);
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
