const jsonfile = require('jsonfile');
const elliptic = require('elliptic');
const path = require('path');
const { assert } = require("chai");

const Wormhole = artifacts.require("Wormhole");
const MockImplementation = artifacts.require("MockImplementation");
const Implementation = artifacts.require("Implementation");
const MockBatchedVAASender = artifacts.require("MockBatchedVAASender");

const testSigner1PK = "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";
const testSigner2PK = "892330666a850761e7370376430bb8c2aa1494072d3bfeaed0c4fa3d5a9135fe";
const testSigner3PK = "87b45997ea577b93073568f06fc4838cffc1d01f90fc4d57f936957f3c4d99fb";
const testBadSigner1PK = "87b45997ea577b93073568f06fc4838cffc1d01f90fc4d57f936957f3c4d99fc";


const core = '0x' + Buffer.from("Core").toString("hex").padStart(64, 0)
const actionContractUpgrade = "01"
const actionGuardianSetUpgrade = "02"
const actionMessageFee = "03"
const actionTransferFee = "04"
const actionRecoverChainId = "05"


const ImplementationFullABI = jsonfile.readFileSync("build/contracts/Implementation.json").abi

const fakeChainId = 1337;
const fakeEvmChainId = 10001;

let lastDeployed;

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
    const testSigner1 = web3.eth.accounts.privateKeyToAccount(testSigner1PK);
    const testSigner2 = web3.eth.accounts.privateKeyToAccount(testSigner2PK);
    const testSigner3 = web3.eth.accounts.privateKeyToAccount(testSigner3PK);
    const testChainId = "2";
    const testEvmChainId = "1";
    const testGovernanceChainId = "1";
    const testGovernanceContract = "0x0000000000000000000000000000000000000000000000000000000000000004";

    it("should be initialized with the correct signers and values", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);

        const index = await initialized.methods.getCurrentGuardianSetIndex().call();
        const set = (await initialized.methods.getGuardianSet(index).call());

        // check set
        assert.lengthOf(set[0], 1);
        assert.equal(set[0][0], testSigner1.address);

        // check expiration
        assert.equal(set.expirationTime, "0");

        // chain id
        const chainId = await initialized.methods.chainId().call();
        assert.equal(chainId, testChainId);

        // evm chain id
        const evmChainId = await initialized.methods.evmChainId().call();
        assert.equal(evmChainId, testEvmChainId);

        // governance
        const governanceChainId = await initialized.methods.governanceChainId().call();
        assert.equal(governanceChainId, testGovernanceChainId);
        const governanceContract = await initialized.methods.governanceContract().call();
        assert.equal(governanceContract, testGovernanceContract);
    })

    it("should log a published message correctly", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const log = await initialized.methods.publishMessage(
            "0x123",
            "0x123321",
            32
        ).send({
            value: 0, // fees are set to 0 initially
            from: accounts[0]
        })

        assert.equal(log.events.LogMessagePublished.returnValues.sender.toString(), accounts[0]);
        assert.equal(log.events.LogMessagePublished.returnValues.sequence.toString(), "0");
        assert.equal(log.events.LogMessagePublished.returnValues.nonce, 291);
        assert.equal(log.events.LogMessagePublished.returnValues.payload.toString(), "0x123321");
        assert.equal(log.events.LogMessagePublished.returnValues.consistencyLevel, 32);
    })

    it("should log sequential sequence numbers for multi-VAA transactions", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        await mockIntegration.methods.sendMultipleMessages(
            "0x1",
            [
                "0x1",
                "0x2",
                "0x3"
            ],
            [
                32,
                32,
                32
            ]
        ).send({
            value: 0, // fees are set to 0 initially
            from: accounts[0]
        });

        const events = (await initialized.getPastEvents('LogMessagePublished', {
            fromBlock: 'latest'
        }))

        let firstSequence = Number(events[0].returnValues.sequence.toString())

        let secondSequence = Number(events[1].returnValues.sequence.toString())
        assert.equal(secondSequence, firstSequence + 1);

        let thirdSequence = Number(events[2].returnValues.sequence.toString())
        assert.equal(thirdSequence, secondSequence + 1);
    })

    it("should increase the sequence for an account", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const log = await initialized.methods.publishMessage(
            "0x1",
            "0x1",
            32
        ).send({
            value: 0, // fees are set to 0 initially
            from: accounts[0]
        })

        assert.equal(log.events.LogMessagePublished.returnValues.sequence.toString(), "1");
    })

    it("should get the same nonce from all VAAs produced by a transaction", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        const nonce = Math.round(Date.now() / 1000);
        const nonceHex = nonce.toString(16)

        await mockIntegration.methods.sendMultipleMessages(
            "0x" + nonceHex,
            [
                "0x1",
                "0x2",
                "0x3"
            ],
            [
                32,
                32,
                32
            ]
        ).send({
            value: 0, // fees are set to 0 initially
            from: accounts[0]
        });

        const events = (await initialized.getPastEvents('LogMessagePublished', {
            fromBlock: 'latest'
        }))

        assert.equal(events[0].returnValues.nonce, nonce);
        assert.equal(events[1].returnValues.nonce, nonce);
        assert.equal(events[2].returnValues.nonce, nonce);
    })

    it("parses VMs correctly", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = 11;
        const emitterAddress = "0x0000000000000000000000000000000000000000000000000000000000000eee"
        const data = "0xaaaaaa";

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            1337,
            data,
            [
                testSigner1PK,
            ],
            0,
            2
        );

        let result
        try {
            result = await initialized.methods.parseAndVerifyVM("0x" + vm).call();
        } catch (err) {
            console.log(err)
            assert.fail("parseAndVerifyVM failed")
        }

        assert.equal(result.vm.version, 1);
        assert.equal(result.vm.timestamp, timestamp);
        assert.equal(result.vm.nonce, nonce);
        assert.equal(result.vm.emitterChainId, emitterChainId);
        assert.equal(result.vm.emitterAddress, emitterAddress);
        assert.equal(result.vm.payload, data);
        assert.equal(result.vm.guardianSetIndex, 0);
        assert.equal(result.vm.sequence, 1337);
        assert.equal(result.vm.consistencyLevel, 2);

        assert.equal(result.valid, true);

        assert.equal(result.reason, "");
    })

    it("should fail quorum on VMs with no signers", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = 11;
        const emitterAddress = "0x0000000000000000000000000000000000000000000000000000000000000eee"
        const data = "0xaaaaaa";

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            1337,
            data,
            [], // no valid signers present
            0,
            2
        );

        let result = await initialized.methods.parseAndVerifyVM("0x" + vm).call();
        assert.equal(result[1], false)
        assert.equal(result[2], "no quorum")
    })


    it("should fail to verify on VMs with bad signer", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = 11;
        const emitterAddress = "0x0000000000000000000000000000000000000000000000000000000000000eee"
        const data = "0xaaaaaa";

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            1337,
            data,
            [
                testBadSigner1PK, // not a valid signer
            ],
            0,
            2
        );

        let result = await initialized.methods.parseAndVerifyVM("0x" + vm).call();
        assert.equal(result[1], false)
        assert.equal(result[2], "VM signature invalid")
    })

    it("should error on VMs with invalid guardian set index", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = 11;
        const emitterAddress = "0x0000000000000000000000000000000000000000000000000000000000000eee"
        const data = "0xaaaaaa";

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            1337,
            data,
            [
                testSigner1PK,
            ],
            200,
            2
        );

        let result = await initialized.methods.parseAndVerifyVM("0x" + vm).call();
        assert.equal(result[1], false)
        assert.equal(result[2], "invalid guardian set")
    })

    it("should revert on VMs with duplicate non-monotonic signature indices", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = 11;
        const emitterAddress = "0x0000000000000000000000000000000000000000000000000000000000000eee"
        const data = "0xaaaaaa";

        const vm = await signAndEncodeVMFixedIndex(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            1337,
            data,
            [
                testSigner1PK,
                testSigner1PK,
                testSigner1PK,
            ],
            0,
            2
        );

        try {
            await initialized.methods.parseAndVerifyVM("0x" + vm).call();
            assert.fail("accepted signature indices being the same in a VM");
        } catch (e) {
            assert.equal(e.data[Object.keys(e.data)[0]].reason, 'signature indices must be ascending')
        }
    })


    it("should set and enforce fees", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = testGovernanceChainId;
        const emitterAddress = testGovernanceContract

        data = [
            //Core
            core,
            // Action 3 (Set Message Fee)
            actionMessageFee,
            // ChainID
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            // Message Fee
            web3.eth.abi.encodeParameter("uint256", 1111).substring(2),
        ].join('')

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            0,
            data,
            [
                testSigner1PK,
            ],
            0,
            2
        );


        let before = await initialized.methods.messageFee().call();

        let set = await initialized.methods.submitSetMessageFee("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 1000000
        });

        let after = await initialized.methods.messageFee().call();

        assert.notEqual(before, after);
        assert.equal(after, 1111);

        // test message publishing
        await initialized.methods.publishMessage(
            "0x123",
            "0x123321",
            32
        ).send({
            from: accounts[0],
            value: 1111
        })

        let failed = false;
        try {
            await initialized.methods.publishMessage(
                "0x123",
                "0x123321",
                32
            ).send({
                value: 1110,
                from: accounts[0]
            })
        } catch (e) {
            failed = true
        }

        assert.equal(failed, true);
    })

    it("should transfer out collected fees", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const receiver = "0x" + zeroPadBytes(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER).toString(16), 20);

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = testGovernanceChainId;
        const emitterAddress = testGovernanceContract

        data = [
            // Core
            core,
            // Action 4 (Transfer Fees)
            actionTransferFee,
            // ChainID
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            // Amount
            web3.eth.abi.encodeParameter("uint256", 11).substring(2),
            // Recipient
            web3.eth.abi.encodeParameter("address", receiver).substring(2),
        ].join('')

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            0,
            data,
            [
                testSigner1PK,
            ],
            0,
            2
        );

        let WHBefore = await web3.eth.getBalance(Wormhole.address);
        let receiverBefore = await web3.eth.getBalance(receiver);

        let set = await initialized.methods.submitTransferFees("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 1000000
        });

        let WHAfter = await web3.eth.getBalance(Wormhole.address);
        let receiverAfter = await web3.eth.getBalance(receiver);

        assert.equal(WHBefore - WHAfter, 11);
        assert.equal(receiverAfter - receiverBefore, 11);
    })

    it("should revert when submitting a new guardian set with the zero address", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = testGovernanceChainId;
        const emitterAddress = testGovernanceContract;
        const zeroAddress = "0x0000000000000000000000000000000000000000";

        let oldIndex = Number(await initialized.methods.getCurrentGuardianSetIndex().call());

        data = [
            // Core
            core,
            // Action 2 (Guardian Set Upgrade)
            actionGuardianSetUpgrade,
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            web3.eth.abi.encodeParameter("uint32", oldIndex + 1).substring(2 + (64 - 8)),
            web3.eth.abi.encodeParameter("uint8", 3).substring(2 + (64 - 2)),
            web3.eth.abi.encodeParameter("address", testSigner1.address).substring(2 + (64 - 40)),
            web3.eth.abi.encodeParameter("address", testSigner2.address).substring(2 + (64 - 40)),
            web3.eth.abi.encodeParameter("address", zeroAddress).substring(2 + (64 - 40)),
        ].join('')

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            0,
            data,
            [
                testSigner1PK
            ],
            0,
            2
        );

        // try to submit a new guardian set including the zero address
        failed = false;
        try {
            await initialized.methods.submitNewGuardianSet("0x" + vm).send({
                value: 0,
                from: accounts[0],
                gasLimit: 1000000
            });
        } catch (e) {
            assert.equal(e.message, "Returned error: VM Exception while processing transaction: revert Invalid key");
            failed = true;
        }

        assert.ok(failed);
    })

    it("should accept a new guardian set", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = testGovernanceChainId;
        const emitterAddress = testGovernanceContract

        let oldIndex = Number(await initialized.methods.getCurrentGuardianSetIndex().call());

        data = [
            // Core
            core,
            // Action 2 (Guardian Set Upgrade)
            actionGuardianSetUpgrade,
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            web3.eth.abi.encodeParameter("uint32", oldIndex + 1).substring(2 + (64 - 8)),
            web3.eth.abi.encodeParameter("uint8", 3).substring(2 + (64 - 2)),
            web3.eth.abi.encodeParameter("address", testSigner1.address).substring(2 + (64 - 40)),
            web3.eth.abi.encodeParameter("address", testSigner2.address).substring(2 + (64 - 40)),
            web3.eth.abi.encodeParameter("address", testSigner3.address).substring(2 + (64 - 40)),
        ].join('')

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            0,
            data,
            [
                testSigner1PK,
            ],
            0,
            2
        );

        let set = await initialized.methods.submitNewGuardianSet("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 1000000
        });

        let index = await initialized.methods.getCurrentGuardianSetIndex().call();

        assert.equal(oldIndex + 1, index);

        assert.equal(index, 1);

        let guardians = await initialized.methods.getGuardianSet(index).call();

        assert.equal(guardians.expirationTime, 0);

        assert.lengthOf(guardians[0], 3);
        assert.equal(guardians[0][0], testSigner1.address);
        assert.equal(guardians[0][1], testSigner2.address);
        assert.equal(guardians[0][2], testSigner3.address);

        let oldGuardians = await initialized.methods.getGuardianSet(oldIndex).call();

        const time = (await web3.eth.getBlock("latest")).timestamp;

        // old guardian set expiry is set
        assert.ok(
            oldGuardians.expirationTime > Number(time) + 86000
            && oldGuardians.expirationTime < Number(time) + 88000
        );
    })

    it("should accept smart contract upgrades", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const mock = await MockImplementation.new();

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = testGovernanceChainId;
        const emitterAddress = testGovernanceContract

        data = [
            // Core
            core,
            // Action 1 (Contract Upgrade)
            actionContractUpgrade,
            // ChainID
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            // New Contract Address
            web3.eth.abi.encodeParameter("address", mock.address).substring(2),
        ].join('')

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            0,
            data,
            [
                testSigner1PK,
                testSigner2PK,
                testSigner3PK
            ],
            1,
            2
        );

        let before = await web3.eth.getStorageAt(Wormhole.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(before.toLowerCase(), Implementation.address.toLowerCase());

        let set = await initialized.methods.submitContractUpgrade("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 1000000
        });

        let after = await web3.eth.getStorageAt(Wormhole.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(after.toLowerCase(), mock.address.toLowerCase());

        const mockImpl = new web3.eth.Contract(MockImplementation.abi, Wormhole.address);

        let isUpgraded = await mockImpl.methods.testNewImplementationActive().call();

        assert.ok(isUpgraded);
        lastDeployed = mock;
    })

    it("should revert recover chain ID governance packets on canonical chains (non-fork)", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = testGovernanceChainId;
        const emitterAddress = testGovernanceContract

        data = [
            // Core
            core,
            // Action 5 (Recover Chain ID)
            actionRecoverChainId,
            // EvmChainID
            web3.eth.abi.encodeParameter("uint256", 1).substring(2),
            // NewChainID
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
        ].join('')

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            0,
            data,
            [
                testSigner1PK,
                testSigner2PK,
                testSigner3PK
            ],
            1,
            2
        );

        try {
            await initialized.methods.submitRecoverChainId("0x" + vm).send({
                value: 0,
                from: accounts[0],
                gasLimit: 1000000
            });
            assert.fail("recover chain ID governance packet on supported chain accepted")
        } catch (e) {
            assert.equal(e.data[Object.keys(e.data)[0]].reason, "not a fork")
        }
    })

    it("should revert governance packets from old guardian set", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        data = [
            // Core
            core,
            // Action 4 (Transfer Fee)
            actionTransferFee,
            // ChainID
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            // Amount
            web3.eth.abi.encodeParameter("uint256", 1).substring(2),
            // Recipient
            web3.eth.abi.encodeParameter("address", "0x0000000000000000000000000000000000000000").substring(2),
        ].join('')

        const vm = await signAndEncodeVM(
            0,
            0,
            testGovernanceChainId,
            testGovernanceContract,
            0,
            data,
            [
                testSigner1PK,
            ],
            0,
            2
        );

        let failed = false;
        try {
            await initialized.methods.submitTransferFees("0x" + vm).send({
                value: 0,
                from: accounts[0],
                gasLimit: 1000000
            });
            assert.fail("governance packet of old guardian set accepted")
        } catch (e) {
            assert.equal(e.data[Object.keys(e.data)[0]].reason, "not signed by current guardian set")
        }
    })

    it("should time out old guardians", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = 11;
        const emitterAddress = "0x0000000000000000000000000000000000000000000000000000000000000eee"
        const data = "0xaaaaaa";

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            0,
            data,
            [
                testSigner1PK,
            ],
            0,
            2
        );

        // this should pass
        const current = await initialized.methods.parseAndVerifyVM("0x" + vm).call();

        assert.equal(current.valid, true)

        await advanceTimeAndBlock(100000);

        const expired = await initialized.methods.parseAndVerifyVM("0x" + vm).call();

        assert.equal(expired.valid, false)
    })

    it("should revert governance packets from wrong governance chain", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        data = [
            // Core
            core,
            // Action 4 (set fees)
            actionTransferFee,
            // ChainID
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            // Amount
            web3.eth.abi.encodeParameter("uint256", 1).substring(2),
            // Recipient
            web3.eth.abi.encodeParameter("address", "0x0000000000000000000000000000000000000000").substring(2),
        ].join('')

        const vm = await signAndEncodeVM(
            0,
            0,
            999,
            testGovernanceContract,
            0,
            data,
            [
                testSigner1PK,
                testSigner2PK,
                testSigner3PK,
            ],
            1,
            2
        );

        try {
            await initialized.methods.submitTransferFees("0x" + vm).send({
                value: 0,
                from: accounts[0],
                gasLimit: 1000000
            });
            assert.fail("governance packet from wrong governance chain accepted")
        } catch (e) {
            assert.equal(e.data[Object.keys(e.data)[0]].reason, "wrong governance chain")
        }
    })

    it("should revert governance packets from wrong governance contract", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        data = [
            // Core
            core,
            // Action 4 (Transfer Fee)
            actionTransferFee,
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            web3.eth.abi.encodeParameter("uint256", 1).substring(2),
            web3.eth.abi.encodeParameter("address", "0x0000000000000000000000000000000000000000").substring(2),
        ].join('')

        const vm = await signAndEncodeVM(
            0,
            0,
            testGovernanceChainId,
            core,
            0,
            data,
            [
                testSigner1PK,
                testSigner2PK,
                testSigner3PK,
            ],
            1,
            2
        );

        try {
            await initialized.methods.submitTransferFees("0x" + vm).send({
                value: 0,
                from: accounts[0],
                gasLimit: 1000000
            });
            assert.fail("governance packet from wrong governance contract accepted")
        } catch (e) {
            assert.equal(e.data[Object.keys(e.data)[0]].reason, "wrong governance contract")
        }
    })

    it("should revert on governance packets that already have been applied", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        data = [
            // Core
            core,
            // Action 4 (Transfer Fee)
            actionTransferFee,
            // ChainID
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            // Amount
            web3.eth.abi.encodeParameter("uint256", 1).substring(2),
            // Recipient
            web3.eth.abi.encodeParameter("address", "0x0000000000000000000000000000000000000000").substring(2),
        ].join('')

        const vm = await signAndEncodeVM(
            0,
            0,
            testGovernanceChainId,
            testGovernanceContract,
            0,
            data,
            [
                testSigner1PK,
                testSigner2PK,
                testSigner3PK,
            ],
            1,
            2
        );

        await initialized.methods.submitTransferFees("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 1000000
        });

        try {
            await initialized.methods.submitTransferFees("0x" + vm).send({
                value: 0,
                from: accounts[0],
                gasLimit: 1000000
            });

            assert.fail("governance packet accepted twice")
        } catch (e) {
            assert.equal(e.data[Object.keys(e.data)[0]].reason, "governance action already consumed")
        }
    })

    it("should reject smart contract upgrades on forks", async function () {
        const mockInitialized = new web3.eth.Contract(MockImplementation.abi, Wormhole.address);
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const mock = await MockImplementation.new();

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = testGovernanceChainId;
        const emitterAddress = testGovernanceContract

        // simulate a fork
        await mockInitialized.methods.testOverwriteEVMChainId(fakeChainId, fakeEvmChainId).send({
            value: 0,
            from: accounts[0],
            gasLimit: 1000000
        });

        const chainId = await initialized.methods.chainId().call();
        assert.equal(chainId, fakeChainId);

        const evmChainId = await initialized.methods.evmChainId().call();
        assert.equal(evmChainId, fakeEvmChainId);

        data = [
            // Core
            core,
            // Action 1 (Contract Upgrade)
            actionContractUpgrade,
            // ChainID
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            // New Contract Address
            web3.eth.abi.encodeParameter("address", mock.address).substring(2),
        ].join('')

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            0,
            data,
            [
                testSigner1PK,
                testSigner2PK,
                testSigner3PK
            ],
            1,
            2
        );

        try {
            await initialized.methods.submitContractUpgrade("0x" + vm).send({
                value: 0,
                from: accounts[0],
                gasLimit: 1000000
            });

            assert.fail("governance packet accepted")
        } catch (e) {
            assert.equal(e.data[Object.keys(e.data)[0]].reason, "invalid fork")
        }
    })

    it("should allow recover chain ID governance packets forks", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = testGovernanceChainId;
        const emitterAddress = testGovernanceContract;

        data = [
            // Core
            core,
            // Action 5 (Recover Chain ID)
            actionRecoverChainId,
            // EvmChainID
            web3.eth.abi.encodeParameter("uint256", testEvmChainId).substring(2),
            // NewChainID
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
        ].join('')

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            0,
            data,
            [
                testSigner1PK,
                testSigner2PK,
                testSigner3PK
            ],
            1,
            2
        );

        await initialized.methods.submitRecoverChainId("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 1000000
        });

        const newChainId = await initialized.methods.chainId().call();
        assert.equal(newChainId, testChainId);

        const newEvmChainId = await initialized.methods.evmChainId().call();
        assert.equal(newEvmChainId, testEvmChainId);
    })

    it("should accept smart contract upgrades after chain ID has been recovered", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        const mock = await MockImplementation.new();

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = testGovernanceChainId;
        const emitterAddress = testGovernanceContract

        data = [
            // Core
            core,
            // Action 1 (Contract Upgrade)
            actionContractUpgrade,
            // ChainID
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            // New Contract Address
            web3.eth.abi.encodeParameter("address", mock.address).substring(2),
        ].join('')

        const vm = await signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            0,
            data,
            [
                testSigner1PK,
                testSigner2PK,
                testSigner3PK
            ],
            1,
            2
        );

        let before = await web3.eth.getStorageAt(Wormhole.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(before.toLowerCase(), lastDeployed.address.toLowerCase());

        let set = await initialized.methods.submitContractUpgrade("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 1000000
        });

        let after = await web3.eth.getStorageAt(Wormhole.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(after.toLowerCase(), mock.address.toLowerCase());

        const mockImpl = new web3.eth.Contract(MockImplementation.abi, Wormhole.address);

        let isUpgraded = await mockImpl.methods.testNewImplementationActive().call();

        assert.ok(isUpgraded);
    })
});

contract("Wormhole VM2 & VM3s", function () {
    // The following tests rely on the observations in TEST_OBSERVATIONS.
    // Be cautious when making changes to the observations in the TEST_OBSERVATIONS array.
    // Adding or removing observations will impact the results the of tests.

    // observation data
    const vaaNonce = 1;
    const emitterChainId = 11;
    const emitterAddress = "0x0000000000000000000000000000000000000000000000000000000000000eee";
    const consistencyLevel = 15;

    let TEST_OBSERVATIONS = [];

    // create the first observation
    TEST_OBSERVATIONS.push({
        timestamp: 1000,
        nonce: vaaNonce,
        emitterChainId: emitterChainId,
        emitterAddress: emitterAddress,
        sequence: 1337,
        consistencyLevel: consistencyLevel,
        payload: "0xaaaa"
    });

    // create a second observation with the same nonce
    TEST_OBSERVATIONS.push({
        timestamp: 1001,
        nonce: vaaNonce,
        emitterChainId: emitterChainId,
        emitterAddress: emitterAddress,
        sequence: 1338,
        consistencyLevel: consistencyLevel,
        payload: "0xbbbbbb"
    });

    // create a third observation with the same nonce
    TEST_OBSERVATIONS.push({
        timestamp: 1002,
        nonce: vaaNonce,
        emitterChainId: emitterChainId,
        emitterAddress: emitterAddress,
        sequence: 1339,
        consistencyLevel: consistencyLevel,
        payload: "0xcccccccccc"
    });

    it("parses VM2s (Batch VAAs) correctly", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const vaaVersion = 3;

        // simulate signing the batch VAA
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK,
                testSigner2PK
            ],
            0
        );

        // parse the batch VAA (parseAndVerifyBatchVM should only be called from a contract)
        let result;
        try {
            result = await initialized.methods.parseBatchVM("0x" + vm2).call();
        } catch (err) {
            console.log(err);
            assert.fail("parseBatchVM failed");
        }

        // confirm that the batch header was parsed correctly
        assert.equal(result.version, 2);
        assert.equal(result.guardianSetIndex, 0);

        // confirm each observation was parsed correctly
        let index = 0;
        for (let i = 0; i < TEST_OBSERVATIONS.length; i++) {
            const testObservation = TEST_OBSERVATIONS[i];
            const indexedObservation = result.indexedObservations[i];

            // index of the observation
            assert.equal(indexedObservation.index.substring(index, 2), i);

            // remove the 0x prefix
            index += 2

            // version
            assert.equal(indexedObservation.observation.substring(index, index + 2), web3.eth.abi.encodeParameter("uint8", vaaVersion).substring(2 + 64 - 2));
            index += 2;

            // timestamp
            assert.equal(parseInt(indexedObservation.observation.substring(index, index + 8), 16), testObservation.timestamp);
            index += 8;

            // nonce
            assert.equal(parseInt(indexedObservation.observation.substring(index, index + 8), 16), testObservation.nonce);
            index += 8;

            // emitterChainId
            assert.equal(parseInt(indexedObservation.observation.substring(index, index + 4), 16), testObservation.emitterChainId);
            index += 4;

            // emitterAddress
            assert.equal(indexedObservation.observation.substring(index, index + 64), testObservation.emitterAddress.substring(2));
            index += 64;

            // sequence
            assert.equal(parseInt(indexedObservation.observation.substring(index, index + 16), 16), testObservation.sequence);
            index += 16;

            // consistencyLevel
            assert.equal(parseInt(indexedObservation.observation.substring(index, index + 2), 16), testObservation.consistencyLevel);
            index += 2;

            // payload
            assert.equal(indexedObservation.observation.substring(index), testObservation.payload.substring(2));
            index = 0;
        }
    })

    it("parses partial VM2s (Paritial Batch VAAs) correctly", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const vaaVersion = 3;
        const removedIndex = 1;

        // simulate signing the batch VAA
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK,
                testSigner2PK
            ],
            0
        );

        // remove the specified observation and create a partial batch
        const partialVm2 = removeObservationFromBatch(removedIndex, vm2);
        assert(partialVm2.length < vm2.length, "failed to remove observation");

        // parse the batch VAA (parseAndVerifyBatchVM should only be called from a contract)
        let result;
        try {
            result = await initialized.methods.parseBatchVM("0x" + partialVm2).call();
        } catch (err) {
            console.log(err);
            assert.fail("parseBatchVM failed");
        }

        // confirm that the batch header was parsed correctly
        assert.equal(result.version, 2);
        assert.equal(result.guardianSetIndex, 0);

        // confirm that the expected number of observations was parsed
        assert.equal(result.indexedObservations.length, TEST_OBSERVATIONS.length - 1);

        // confirm that the indices for the observations are correct
        assert.equal(result.indexedObservations[0].index, 0);
        assert.equal(result.indexedObservations[1].index, 2); // index 1 was removed

        // confirm each observation was parsed correctly
        index = 0;
        for (let i = 0; i < TEST_OBSERVATIONS.length - 1; i++) {
            // skip the observation that was removed
            if (i == removedIndex) { continue; }

            const testObservation = TEST_OBSERVATIONS[i];
            const indexedObservation = result.indexedObservations[i];

            // remove the 0x prefix
            index += 2

            // version
            assert.equal(indexedObservation.observation.substring(index, index + 2), web3.eth.abi.encodeParameter("uint8", vaaVersion).substring(2 + 64 - 2));
            index += 2;

            // timestamp
            assert.equal(parseInt(indexedObservation.observation.substring(index, index + 8), 16), testObservation.timestamp);
            index += 8;

            // nonce
            assert.equal(parseInt(indexedObservation.observation.substring(index, index + 8), 16), testObservation.nonce);
            index += 8;

            // emitterChainId
            assert.equal(parseInt(indexedObservation.observation.substring(index, index + 4), 16), testObservation.emitterChainId);
            index += 4;

            // emitterAddress
            assert.equal(indexedObservation.observation.substring(index, index + 64), testObservation.emitterAddress.substring(2));
            index += 64;

            // sequence
            assert.equal(parseInt(indexedObservation.observation.substring(index, index + 16), 16), testObservation.sequence);
            index += 16;

            // consistencyLevel
            assert.equal(parseInt(indexedObservation.observation.substring(index, index + 2), 16), testObservation.consistencyLevel);
            index += 2;

            // payload
            assert.equal(indexedObservation.observation.substring(index), testObservation.payload.substring(2));
            index = 0;
        }
    })

    it("should verify VM2s from a contract correctly", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const cacheObservations = false;

        // create the mock integration contract instance
        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // We need to call this from a contract to verify that the batch has been verified
        // properly. parseAndVerifyBatchVM modifies state, so the valid status (and parsed VM2) is not returned
        // when calling from JS.
        try {
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + vm2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyBatchVM failed");
        }

        // Compute the hash of each observation and confirm
        // that it is not stored in the batch cache, since
        // the cacheObservations flag is set to false.
        for (const observation of TEST_OBSERVATIONS) {
            const observationHash = doubleKeccak256(encodeObservation(observation));

            // query the contract using the hash
            const hashIsCached = await initialized.methods.verifiedHashCached(observationHash).call();
            assert.ok(!hashIsCached);
        }
    })

    it("should verify partial VM2s from a contract correctly", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const cacheObservations = false;
        const removedIndex = 1;

        // create the mock integration contract instance
        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // remove the specified observation and create a partial batch
        const partialVm2 = removeObservationFromBatch(removedIndex, vm2);
        assert(partialVm2.length < vm2.length, "failed to remove observation");

        // We need to call this from a contract to verify that the batch has been verified
        // properly. parseAndVerifyBatchVM modifies state, so the valid status (and parsed VM2) is not returned
        // when calling from JS.
        try {
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + partialVm2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyBatchVM failed");
        }

        // Compute the hash of each observation and confirm
        // that it is not stored in the batch cache, since
        // the cacheObservations flag is set to false.
        for (let i = 0; i < TEST_OBSERVATIONS.length - 1; i++) {
            const observation = TEST_OBSERVATIONS[i];
            const observationHash = doubleKeccak256(encodeObservation(observation));

            // query the contract using the hash
            const hashIsCached = await initialized.methods.verifiedHashCached(observationHash).call();
            assert.ok(!hashIsCached);
        }
    })

    it("should store the hash of each observation in a batch cache", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const cacheObservations = true;

        // create the mock integration contract instance
        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // parse and store the VM2
        const parsedVM2 = await initialized.methods.parseBatchVM("0x" + vm2).call();

        // We need to call this from a contract to verify that the batch has been verified
        // properly. parseAndVerifyBatchVM modifies state, so the valid status (and parsed VM2) is not returned
        // when calling from JS.
        try {
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + vm2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyBatchVM failed");
        }

        // Compute the hash of each observation and confirm
        // that it is correctly stored in the contract's cache.
        for (const observation of TEST_OBSERVATIONS) {
            const observationHash = doubleKeccak256(encodeObservation(observation));

            // query the contract using the hash
            const hashIsCached = await initialized.methods.verifiedHashCached(observationHash).call();
            assert.ok(hashIsCached);
        }

        // clear the batch cache
        await initialized.methods.clearBatchCache(parsedVM2.hashes).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });
    })

    it("should only store hashes in a batch cache when the corresponding observation exists", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const cacheObservations = true;
        const removedIndex = 2;

        // deploy the mock integration contract
        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // remove the last observation and create a partial batch
        const partialVm2 = removeObservationFromBatch(removedIndex, vm2);
        assert(partialVm2.length < vm2.length, "failed to remove observation");

        // parse and store the VM2
        const parsedPartialVm2 = await initialized.methods.parseBatchVM("0x" + partialVm2).call();

        // We need to call this from a contract to verify that the batch has been verified
        // properly. parseAndVerifyBatchVM modifies state, so the valid status (and parsed VM2) is not returned
        // when calling from JS.
        try {
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + partialVm2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyBatchVM failed");
        }

        // Compute the hash of each observation and confirm
        // that it is correctly stored in the contract's cache.
        for (let i = 0; i < TEST_OBSERVATIONS.length; i++) {
            const observationHash = doubleKeccak256(encodeObservation(TEST_OBSERVATIONS[i]));

            // Query the contract using the hash. Make sure that
            // the indexed obsevation that was removed from the batch is not cached.
            const hashIsCached = await initialized.methods.verifiedHashCached(observationHash).call();
            if (i == removedIndex) {
                assert.ok(!hashIsCached)
            } else {
                assert.ok(hashIsCached);
            }
        }

        // clear the batch cache (only provide hashes in the parital VM2)
        let hashesToRemove = [];
        for (const indexedObservation of parsedPartialVm2.indexedObservations) {
            hashesToRemove.push(parsedPartialVm2.hashes[indexedObservation.index])
        }
        await initialized.methods.clearBatchCache(hashesToRemove).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });
    })

    it("should remove the hash of each observation from the batch cache", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const cacheObservations = true;

        // deploy the mock integration contract
        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // parse and store the VM2
        const parsedVM2 = await initialized.methods.parseBatchVM("0x" + vm2).call();

        // We need to call this from a contract to verify that the batch has been verified
        // properly. parseAndVerifyBatchVM modifies state, so the valid status (and parsed VM2) is not returned
        // when calling from JS.
        try {
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + vm2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyBatchVM failed");
        }

        // Loop through the list of observations twice. In the first loop
        // check that each observation is cached. Then clear the cache,
        // and confirm that each hash was removed.
        for (let i = 0; i < 2; i++) {
            for (const observation of TEST_OBSERVATIONS) {
                const observationHash = doubleKeccak256(encodeObservation(observation));

                // query the contract using the hash
                const hashIsCached = await initialized.methods.verifiedHashCached(observationHash).call();

                if (i == 0) {
                    assert.ok(hashIsCached);
                } else {
                    assert.ok(!hashIsCached);
                }
            }

            // clear the batch cache after completing the first loop
            if (i == 0) {
                await initialized.methods.clearBatchCache(parsedVM2.hashes).send({
                    value: 0,
                    from: accounts[0],
                    gasLimit: 2000000
                });
            }
        }
    })

    it("should not verify a VM2 with parseAndVerifyVM", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        failed = false;
        try {
            // attempt to verify a VM2 with parseAndVerifyVM
            const result = await initialized.methods.parseAndVerifyVM("0x" + vm2).call();
        } catch (e) {
            assert.equal(
                e.message,
                "Returned error: VM Exception while processing transaction: revert Invalid version"
            );
            failed = true;
        }

        assert.ok(failed);
    })

    it("should not verify a VM2 with a missing observation and hash", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const signers = [testSigner1PK];
        const cacheObservations = false;

        // sign VM2 with all observations
        const startingVM2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            signers,
            0
        );

        // remove the last observation and sign a new VM2
        let endingVM2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS.slice(0, TEST_OBSERVATIONS.length - 1),
            signers,
            0
        );

        // copy the signatures from startingVM2 to endingVM2
        endingVM2 = copySignaturesToVM2(startingVM2, endingVM2);

        // parse the VM2s
        const parsedStartingVM2 = await initialized.methods.parseBatchVM("0x" + startingVM2).call();
        const parsedEndingVM2 = await initialized.methods.parseBatchVM("0x" + endingVM2).call();

        // confirm that both VM2s have the same hashes (except for the last one that was removed)
        assert.equal(parsedStartingVM2.hashes[0], parsedEndingVM2.hashes[0]);
        assert.equal(parsedStartingVM2.hashes[1], parsedEndingVM2.hashes[1]);

        // confirm that the original VM2 has more hashes than the modified one
        assert.equal(parsedStartingVM2.hashes.length, parsedEndingVM2.hashes.length + 1);

        // confirm both VM2s have the same signatures
        const startingSig = parsedStartingVM2.signatures[0];
        const endingSig = parsedEndingVM2.signatures[0];

        assert.equal(startingSig.r, endingSig.r);
        assert.equal(startingSig.s, endingSig.s);
        assert.equal(startingSig.v, endingSig.v);
        assert.equal(startingSig.guardianIndex, endingSig.guardianIndex);

        // try to verify the modified VM2
        failed = false;
        try {
            const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + endingVM2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (e) {
            assert.equal(
                e.message,
                "Returned error: VM Exception while processing transaction: revert VM signature invalid"
            );
            failed = true;
        }

        assert.ok(failed);
    })

    it("should not verify a VM2 with an additional observation and hash", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const signers = [testSigner1PK];
        const cacheObservations = false;
        let testObservations = [...TEST_OBSERVATIONS];

        // sign VM2 with all observations
        const startingVM2 = await signAndEncodeVM2(
            testObservations,
            signers,
            0
        );

        // create new observation
        testObservations.push({
            timestamp: 1005,
            nonce: vaaNonce,
            emitterChainId: emitterChainId,
            emitterAddress: emitterAddress,
            sequence: 1340,
            consistencyLevel: consistencyLevel,
            payload: "0xffffffff"
        });

        // add a new observation and sign a new VM2
        let endingVM2 = await signAndEncodeVM2(
            testObservations,
            signers,
            0
        );

        // copy the signatures from startingVM2 to endingVM2
        endingVM2 = copySignaturesToVM2(startingVM2, endingVM2);

        // parse the VM2s
        const parsedStartingVM2 = await initialized.methods.parseBatchVM("0x" + startingVM2).call();
        const parsedEndingVM2 = await initialized.methods.parseBatchVM("0x" + endingVM2).call();

        // confirm that both VM2s have the same hashes (except for the last additional observation)
        assert.equal(parsedStartingVM2.hashes[0], parsedEndingVM2.hashes[0]);
        assert.equal(parsedStartingVM2.hashes[1], parsedEndingVM2.hashes[1]);
        assert.equal(parsedStartingVM2.hashes[2], parsedEndingVM2.hashes[2]);

        // confirm that the original VM2 has one less hash than the modified one
        assert.equal(parsedStartingVM2.hashes.length, parsedEndingVM2.hashes.length - 1);

        // confirm both VM2s have the same signatures
        const startingSig = parsedStartingVM2.signatures[0];
        const endingSig = parsedEndingVM2.signatures[0];

        assert.equal(startingSig.r, endingSig.r);
        assert.equal(startingSig.s, endingSig.s);
        assert.equal(startingSig.v, endingSig.v);
        assert.equal(startingSig.guardianIndex, endingSig.guardianIndex);

        // try to verify the modified VM2
        failed = false;
        try {
            const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + endingVM2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (e) {
            assert.equal(
                e.message,
                "Returned error: VM Exception while processing transaction: revert VM signature invalid"
            );
            failed = true;
        }

        assert.ok(failed);
    })

    it("should not verify a VM2 with reorganized observations and hashes", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const signers = [testSigner1PK];
        const cacheObservations = false;
        let testObservations = [];

        // sign VM2 with all observations
        const startingVM2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            signers,
            0
        );

        // reorganize the obsevations and sign a new VM2
        testObservations.push(TEST_OBSERVATIONS[2]);
        testObservations.push(TEST_OBSERVATIONS[0]);
        testObservations.push(TEST_OBSERVATIONS[1]);

        let endingVM2 = await signAndEncodeVM2(
            testObservations,
            signers,
            0
        );

        // copy the signatures from startingVM2 to endingVM2
        endingVM2 = copySignaturesToVM2(startingVM2, endingVM2);

        // parse the VM2s
        const parsedStartingVM2 = await initialized.methods.parseBatchVM("0x" + startingVM2).call();
        const parsedEndingVM2 = await initialized.methods.parseBatchVM("0x" + endingVM2).call();

        // confirm that both VM2s have the same hashes (but in different orders)
        assert.equal(parsedStartingVM2.hashes[2], parsedEndingVM2.hashes[0]);
        assert.equal(parsedStartingVM2.hashes[0], parsedEndingVM2.hashes[1]);
        assert.equal(parsedStartingVM2.hashes[1], parsedEndingVM2.hashes[2]);

        // confirm both VM2s have the same number of hashes
        assert.equal(parsedStartingVM2.hashes.length, parsedEndingVM2.hashes.length);

        // confirm both VM2s have the same signatures
        const startingSig = parsedStartingVM2.signatures[0];
        const endingSig = parsedEndingVM2.signatures[0];

        assert.equal(startingSig.r, endingSig.r);
        assert.equal(startingSig.s, endingSig.s);
        assert.equal(startingSig.v, endingSig.v);
        assert.equal(startingSig.guardianIndex, endingSig.guardianIndex);

        // try to verify the modified VM2
        failed = false;
        try {
            const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + endingVM2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (e) {
            assert.equal(
                e.message,
                "Returned error: VM Exception while processing transaction: revert VM signature invalid"
            );
            failed = true;
        }

        assert.ok(failed);
    })

    it("parses and verifies VM3s (Headless VAAs) correctly", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const cacheObservations = true;

        // deploy the mock integration contract
        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // Parse and verify the batch header so the observation hashes are stored
        // and VM3s can be verified.
        const parsedVM2 = await initialized.methods.parseBatchVM("0x" + vm2).call();

        // We need to call this from a contract to verify that the batch has been verified
        // properly. parseAndVerifyBatchVM modifies state, so the valid status (and parsed VM2) is not returned
        // when calling from JS.
        try {
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + vm2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyBatchVM failed");
        }

        // confirm that each observation is parsed and verified correctly by parseAndVerifyVM
        for (let i = 0; i < parsedVM2.indexedObservations.length; i++) {
            let verifiedVm;
            try {
                verifiedVm = await initialized.methods.parseAndVerifyVM(
                    parsedVM2.indexedObservations[i].observation
                ).call();
            } catch (err) {
                console.log(err);
                assert.fail("parseAndVerifyVM failed");
            }

            // confirm signatures array is empty and the guardianSetIndex is zero
            assert.equal(verifiedVm.vm.signatures.length, 0);
            assert.equal(verifiedVm.vm.guardianSetIndex, 0);

            // compare the test observation with the parsed output from parseAndVerifyVM
            assert.equal(verifiedVm.vm.timestamp, TEST_OBSERVATIONS[i].timestamp);
            assert.equal(verifiedVm.vm.nonce, TEST_OBSERVATIONS[i].nonce);
            assert.equal(verifiedVm.vm.emitterChainId, TEST_OBSERVATIONS[i].emitterChainId);
            assert.equal(verifiedVm.vm.emitterAddress, TEST_OBSERVATIONS[i].emitterAddress);
            assert.equal(verifiedVm.vm.sequence, TEST_OBSERVATIONS[i].sequence);
            assert.equal(verifiedVm.vm.consistencyLevel, TEST_OBSERVATIONS[i].consistencyLevel);
            assert.equal(verifiedVm.vm.payload, TEST_OBSERVATIONS[i].payload);

            // compare hash
            assert.equal(verifiedVm.vm.hash, parsedVM2.hashes[i]);

            // confirm the VM3 was verified
            assert.ok(verifiedVm.valid);
            assert.equal(verifiedVm.reason, "");
        }

        // clear the batch cache
        await initialized.methods.clearBatchCache(parsedVM2.hashes).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });
    })

    it("should not verify VM3s after the batch cache is cleared", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const cacheObservations = true;

        // deploy the mock integration contract
        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // Parse and verify the batch header so the observation hashes are stored
        // and VM3s can be verified.
        const parsedVM2 = await initialized.methods.parseBatchVM("0x" + vm2).call();

        // We need to call this from a contract to verify that the batch has been verified
        // properly. parseAndVerifyBatchVM modifies state, so the valid status (and parsed VM2) is not returned
        // when calling from JS.
        try {
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + vm2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyBatchVM failed");
        }

        // verify the first two VM3s
        for (let i = 0; i < parsedVM2.indexedObservations.length - 1; i++) {
            try {
                await initialized.methods.parseAndVerifyVM(
                    parsedVM2.indexedObservations[i].observation
                ).call();
            } catch (err) {
                console.log(err);
                assert.fail("parseAndVerifyVM failed");
            }
        }

        // clear the batch cache and try to verify the last VM3 (it should fail)
        await initialized.methods.clearBatchCache(parsedVM2.hashes).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        const result = await initialized.methods.parseAndVerifyVM(
            parsedVM2.indexedObservations[parsedVM2.indexedObservations.length - 1].observation
        ).call();

        assert.equal(result.reason, "Could not find hash in cache");
        assert.ok(!result.valid);
    })

    it("should verify VM3s from a contract correctly", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        // deploy the mock integration contract
        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // Calls mock integration contract, which will parse and verify the VM2,
        // and then parse and verify each VM3 separately. It stores each VM3 payload
        // in an array to verify that the VM3 was parsed correctly.
        try {
            await mockIntegration.methods.consumeBatchVAA("0x" + vm2).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("consumeBatchVAA failed");
        }

        for (let i = 0; i < TEST_OBSERVATIONS.length; i++) {
            // confirm that the batch cache was cleared
            const observationHash = doubleKeccak256(encodeObservation(TEST_OBSERVATIONS[i]));

            // query the contract using the hash
            const hashIsCached = await initialized.methods.verifiedHashCached(observationHash).call();
            assert.ok(!hashIsCached);

            // fetch the payload that is stored in the mock integration contract
            const queriedPayload = await mockIntegration.methods.getPayload(observationHash).call();

            // validate payloads
            assert.equal(queriedPayload, TEST_OBSERVATIONS[i].payload);
        }
    })

    it("parses and verifies VM1s after the batch cache is cleared", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const signers = [testSigner1PK];
        const cacheObservations = true;

        // deploy the mock integration contract
        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            signers,
            0
        );

        // simulate signing a VM1 that is included in the VM2 batch
        const vm = await signAndEncodeVM(
            TEST_OBSERVATIONS[0].timestamp,
            TEST_OBSERVATIONS[0].nonce,
            TEST_OBSERVATIONS[0].emitterChainId,
            TEST_OBSERVATIONS[0].emitterAddress,
            TEST_OBSERVATIONS[0].sequence,
            TEST_OBSERVATIONS[0].payload,
            signers,
            0,
            TEST_OBSERVATIONS[0].consistencyLevel
        );

        // Parse and verify the batch header so the observation hashes are stored
        // and VM3s can be verified.
        const parsedVM2 = await initialized.methods.parseBatchVM("0x" + vm2).call();

        // We need to call this from a contract to verify that the batch has been verified
        // properly. parseAndVerifyBatchVM modifies state, so the valid status (and parsed VM2) is not returned
        // when calling from JS.
        try {
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + vm2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyBatchVM failed");
        }

        // parse and verify the first VM3
        let parsedVM3;
        try {
            parsedVM3 = await initialized.methods.parseAndVerifyVM(
                parsedVM2.indexedObservations[0].observation
            ).call();
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyVM failed");
        }

        // clear the batch cache
        await initialized.methods.clearBatchCache(parsedVM2.hashes).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // confirm that the VM1 can still be parsed and verified
        let parsedVM1;
        try {
            parsedVM1 = await initialized.methods.parseAndVerifyVM(
                "0x" + vm
            ).call();
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyVM failed");
        }

        // confirm that the VM1 and VM3 are the same observation and both were verified
        assert.ok(parsedVM1.valid);
        assert.ok(parsedVM3.valid);
        assert.equal(parsedVM1.vm.hash, parsedVM3.vm.hash);

        assert.equal(parsedVM1.vm.timestamp, parsedVM3.vm.timestamp);
        assert.equal(parsedVM1.vm.nonce, parsedVM3.vm.nonce);
        assert.equal(parsedVM1.vm.emitterChainId, parsedVM3.vm.emitterChainId);
        assert.equal(parsedVM1.vm.emitterAddress, parsedVM3.vm.emitterAddress);
        assert.equal(parsedVM1.vm.sequence, parsedVM3.vm.sequence);
        assert.equal(parsedVM1.vm.consistencyLevel, parsedVM3.vm.consistencyLevel);
        assert.equal(parsedVM1.vm.payload, parsedVM3.vm.payload);
    })

    it("should not verify a VM3 with spoofed version", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const signers = [testSigner1PK];
        const cacheObservations = false;

        // deploy the mock integration contract
        const mockIntegration = new web3.eth.Contract(MockBatchedVAASender.abi, MockBatchedVAASender.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            signers,
            0
        );

        // Parse and verify the batch header so the observation hashes are stored
        // and VM3s can be verified.
        const parsedVM2 = await initialized.methods.parseBatchVM("0x" + vm2).call();

        // We need to call this from a contract to verify that the batch has been verified
        // properly. parseAndVerifyBatchVM modifies state, so the valid status (and parsed VM2) is not returned
        // when calling from JS.
        try {
            await mockIntegration.methods.parseAndVerifyBatchVM("0x" + vm2, cacheObservations).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyBatchVM failed");
        }

        // parse and verify the first VM3
        let parsedVM3 = await initialized.methods.parseVM(
            parsedVM2.indexedObservations[0].observation
        ).call();

        // create a spoofed version of the VM1
        const spoofedVM3 = {
            version: "1",
            timestamp: parsedVM3.timestamp,
            nonce: parsedVM3.timestamp,
            emitterChainId: parsedVM3.emitterChainId,
            emitterAddress: parsedVM3.emitterAddress,
            sequence: parsedVM3.sequence,
            consistencyLevel: parsedVM3.consistencyLevel,
            payload: parsedVM3.payload,
            guardianSetIndex: parsedVM3.guardianSetIndex,
            signatures: parsedVM3.signatures,
            hash: parsedVM3.hash
        }

        // try to verify the spoofed VM3
        let result;
        try {
            result = await initialized.methods.verifyVM(spoofedVM3).call();
        } catch (err) {
            console.log(err);
            assert.fail("verifyVM failed");
        }

        assert.ok(!result.valid);
        assert.equal(result.reason, "no quorum");
    })
});

function removeObservationFromBatch(indexToRemove, encodedVM) {
    // index of the signature length (number of signers for the VM)
    let index = 10;

    // grab the signature count
    const sigCount = parseInt(encodedVM.slice(index, index + 2), 16);
    index += 2;

    // skip the signatures
    index += 132 * sigCount;

    // hash count
    const hashCount = parseInt(encodedVM.slice(index, index + 2), 16);
    index += 2;

    // skip the hashes
    index += 64 * hashCount;

    // observation count
    const observationCount = parseInt(encodedVM.slice(index, index + 2), 16);
    const observationCountIndex = index; // save the index
    index += 2

    // find the index of the observation that will be removed
    let bytesRangeToRemove = [0, 0];
    for (let i = 0; i < observationCount; i++) {
        const observationStartIndex = index;

        // parse the observation index and the observation length
        const observationIndex = parseInt(encodedVM.slice(index, index + 2), 16);
        index += 2;

        const observationLen = parseInt(encodedVM.slice(index, index + 8), 16);
        index += 8;

        // save the index of the observation we want to remove
        if (observationIndex == indexToRemove) {
            bytesRangeToRemove[0] = observationStartIndex;
            bytesRangeToRemove[1] = observationStartIndex + 10 + observationLen * 2;
        }
        index += observationLen * 2
    }

    // remove the observation from the batch VAA
    let newVAAElements = [
        // slice to the observation count
        encodedVM.slice(0, observationCountIndex),
        web3.eth.abi.encodeParameter("uint8", observationCount - 1).substring(2 + (64 - 2)),
        encodedVM.slice(observationCountIndex + 2, bytesRangeToRemove[0]),
        encodedVM.slice(bytesRangeToRemove[1])
    ];

    return newVAAElements.join("");
}

function copySignaturesToVM2(fromVM, toVM) {
    // index of the signature length (number of signers for the VM)
    let index = 10;

    // grab the number of signatures for each VM
    sigCountFrom = parseInt(fromVM.slice(index, index + 2), 16);
    sigCountTo = parseInt(toVM.slice(index, index + 2), 16);
    index += 2

    // grab the signatures for the startVM (each signature is 66 bytes (132 for string representation))
    const fromVMSigs = fromVM.slice(index, index + (132 * sigCountFrom));

    // create a new VAA with the signatures from startVM
    const resultVM = toVM.slice(0, index) + fromVMSigs + toVM.slice(index + (132 * sigCountTo));
    return resultVM;
}

function doubleKeccak256(bytes) {
    return web3.utils.soliditySha3(web3.utils.soliditySha3("0x" + bytes));
}

function encodeObservation(observation) {
    let encodedObservation = [
        web3.eth.abi.encodeParameter("uint32", observation.timestamp).substring(2 + (64 - 8)),
        web3.eth.abi.encodeParameter("uint32", observation.nonce).substring(2 + (64 - 8)),
        web3.eth.abi.encodeParameter("uint16", observation.emitterChainId).substring(2 + (64 - 4)),
        web3.eth.abi.encodeParameter("bytes32", observation.emitterAddress).substring(2),
        web3.eth.abi.encodeParameter("uint64", observation.sequence).substring(2 + (64 - 16)),
        web3.eth.abi.encodeParameter("uint8", observation.consistencyLevel).substring(2 + (64 - 2)),
        observation.payload.substring(2)
    ];
    // create the observation bytestring
    return encodedObservation.join("");
}

const signAndEncodeVM2 = async function (
    observationsArray,
    signers,
    guardianSetIndex
) {
    observationHashes = "";
    encodedObservationsWithLengthPrefix = "";
    for (let i = 0; i < observationsArray.length; i++) {
        observation = observationsArray[i];

        // encode the observation
        const observationBytes = encodeObservation(observation);

        // hash the observation
        const hash = doubleKeccak256(observationBytes);
        observationHashes += hash.substring(2);

        // grab the index, and length of the observation and add them to the observation bytestring
        // divide observationBytes by two to convert string representation length to bytes
        const observationElements = [
            web3.eth.abi.encodeParameter("uint8", i).substring(2 + (64 - 2)),
            web3.eth.abi.encodeParameter("uint32", observationBytes.length / 2).substring(2 + (64 - 8)),
            observationBytes
        ]
        encodedObservationsWithLengthPrefix += observationElements.join("");
    }

    // compute the hash of batch hashes - hash(hash(VAA1), hash(VAA2), ...)
    const batchHash = doubleKeccak256(observationHashes);

    let signatures = "";

    for (let i in signers) {
        const ec = new elliptic.ec("secp256k1");
        const key = ec.keyFromPrivate(signers[i]);
        const signature = key.sign(batchHash.substring(2), {canonical: true});

        const packSig = [
            web3.eth.abi.encodeParameter("uint8", i).substring(2 + (64 - 2)),
            zeroPadBytes(signature.r.toString(16), 32),
            zeroPadBytes(signature.s.toString(16), 32),
            web3.eth.abi.encodeParameter("uint8", signature.recoveryParam).substring(2 + (64 - 2)),
        ]

        signatures += packSig.join("")
    }

    const vm = [
        // this is a type 2 VAA since it's a batch
        web3.eth.abi.encodeParameter("uint8", 2).substring(2 + (64 - 2)),
        web3.eth.abi.encodeParameter("uint32", guardianSetIndex).substring(2 + (64 - 8)),
        web3.eth.abi.encodeParameter("uint8", signers.length).substring(2 + (64 - 2)),
        signatures,
        web3.eth.abi.encodeParameter("uint8", observationsArray.length).substring(2 + (64 - 2)),
        observationHashes,
        web3.eth.abi.encodeParameter("uint8", observationsArray.length).substring(2 + (64 - 2)),
        encodedObservationsWithLengthPrefix
    ].join("");

    return vm
}

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
        const signature = key.sign(hash.substr(2), { canonical: true });

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

const signAndEncodeVMFixedIndex = async function (
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
        const signature = key.sign(hash.substr(2), { canonical: true });

        const packSig = [
            // Fixing the index to be zero to product a non-monotonic VM
            web3.eth.abi.encodeParameter("uint8", 0).substring(2 + (64 - 2)),
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