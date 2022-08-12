const jsonfile = require('jsonfile');
const elliptic = require('elliptic');
const path = require('path');
const { assert } = require("chai");

const Wormhole = artifacts.require("Wormhole");
const MockImplementation = artifacts.require("MockImplementation");
const Implementation = artifacts.require("Implementation");
const MockBatchedVAASender = artifacts.require("MockBatchedVAASender");
const MockIntegration = artifacts.require("MockBatchMessageIntegration");

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
            "0x1",
            32
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
            "0x1",
            32
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

    it("should revert on VMs with duplicate non-monotonic signature indexes", async function () {
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
            assert.fail("accepted signature indexes being the same in a VM");
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
                testSigner1PK
            ],
            0
        );

        // parse the batch VAA (parseAndVerifyVM2 should only be called from a contract)
        let result;
        try {
            result = await initialized.methods.parseVM2("0x" + vm2).call();
        } catch (err) {
            console.log(err);
            assert.fail("parseVM2 failed");
        }

        // confirm the header was parsed correctly
        assert.equal(result.header.version, 2);
        assert.equal(result.header.guardianSetIndex, 0);

        // confirm each observation was parsed correctly
        let index = 0;
        for (let i = 0; i < TEST_OBSERVATIONS.length; i++) {
            const testObservation = TEST_OBSERVATIONS[i];
            const parsedObservation = result.observations[i].substring(2);

            // version
            assert.equal(parsedObservation.substring(index, 2), web3.eth.abi.encodeParameter("uint8", vaaVersion).substring(2 + 64 - 2));
            index += 2;

            // timestamp
            assert.equal(parseInt(parsedObservation.substring(index, index+8), 16), testObservation.timestamp);
            index += 8;

            // nonce
            assert.equal(parseInt(parsedObservation.substring(index, index+8), 16), testObservation.nonce);
            index += 8;

            // emitterChainId
            assert.equal(parseInt(parsedObservation.substring(index, index+4), 16), testObservation.emitterChainId);
            index += 4;

            // emitterAddress
            assert.equal(parsedObservation.substring(index, index+64), testObservation.emitterAddress.substring(2));
            index += 64;

            // sequence
            assert.equal(parseInt(parsedObservation.substring(index, index+16), 16), testObservation.sequence);
            index += 16;

            // consistencyLevel
            assert.equal(parseInt(parsedObservation.substring(index, index+2), 16), testObservation.consistencyLevel);
            index += 2;

            // payload
            assert.equal(parsedObservation.substring(index), testObservation.payload.substring(2));
            index = 0;
        }
    })

    it("should verify VM2s from a contract correctly", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        // deploy the mock integration contract
        const integrationContract = await deployMockIntegrationContract(accounts[0]);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // parse and store the VM2
        const parsedVM2 = await initialized.methods.parseVM2("0x" + vm2).call();

        // We need to call this from a contract to verify that the batch has been verified
        // properly. parseAndVerifyVM2 modifies state, so the valid status (and parsed VM2) is not returned
        // when calling from JS.
        try {
            await integrationContract.methods.parseAndVerifyVM2("0x" + vm2).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyVM2 failed");
        }

        // clear the batch cache
        await initialized.methods.clearBatchCache(parsedVM2.header).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });
    })

    it("should store the hash of each observation in a batch cache", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        // deploy the mock integration contract
        const integrationContract = await deployMockIntegrationContract(accounts[0]);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // parse and store the VM2
        const parsedVM2 = await initialized.methods.parseVM2("0x" + vm2).call();

        // parse and verify the batch of observations
        await integrationContract.methods.parseAndVerifyVM2("0x" + vm2).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // Compute the hash of each observation and confirm
        // that it is correctly stored in the contract's cache.
        for (const observation of TEST_OBSERVATIONS) {
            const observationHash = doubleKeccak256(encodeObservation(observation));

            // query the contract using the hash
            const hashIsCached = await initialized.methods.verifiedHashCached(observationHash).call();
            assert.ok(hashIsCached);
        }

        // clear the batch cache
        await initialized.methods.clearBatchCache(parsedVM2.header).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });
    })

    it("should remove the hash of each observation from the batch cache", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        // deploy the mock integration contract
        const integrationContract = await deployMockIntegrationContract(accounts[0]);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // parse and store the VM2
        const parsedVM2 = await initialized.methods.parseVM2("0x" + vm2).call();

        // parse and verify the batch of observations
        await integrationContract.methods.parseAndVerifyVM2("0x" + vm2).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

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
                    assert.ok(!hashIsCached);                }
            }

            // clear the batch cache after completing the first loop
            if (i == 0) {
                await initialized.methods.clearBatchCache(parsedVM2.header).send({
                    value: 0,
                    from: accounts[0],
                    gasLimit: 2000000
                });
            }
        }
    })

    it("should not verify a VM2 with parseAndVerifyVAA", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);

        // simulate signing the VM2
        const vm2 = await signAndEncodeVM2(
            TEST_OBSERVATIONS,
            [
                testSigner1PK
            ],
            0
        );

        // attempt to verify a VM2 with parseAndVerifyVAA
        const result = await initialized.methods.parseAndVerifyVAA("0x" + vm2).call();

        assert.equal(result.reason, "Invalid version");
        assert.ok(!result.valid);
    })

    it("should not verify a VM2 with a missing observation", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const signers = [testSigner1PK];

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
        const parsedStartingVM2 = await initialized.methods.parseVM2("0x" + startingVM2).call();
        const parsedEndingVM2 = await initialized.methods.parseVM2("0x" + endingVM2).call();

        // confirm that both VM2s have the same hashes (except for the last one that was removed)
        assert.equal(parsedStartingVM2.header.hashes[0], parsedEndingVM2.header.hashes[0]);
        assert.equal(parsedStartingVM2.header.hashes[1], parsedEndingVM2.header.hashes[1]);

        // confirm that the original VM has more hashes than the modified one
        assert.equal(parsedStartingVM2.header.hashes.length, parsedEndingVM2.header.hashes.length + 1);

        // confirm both VMs have the same signatures
        const startingSig = parsedStartingVM2.header.signatures[0];
        const endingSig = parsedEndingVM2.header.signatures[0];

        assert.equal(startingSig.r, endingSig.r);
        assert.equal(startingSig.s, endingSig.s);
        assert.equal(startingSig.v, endingSig.v);
        assert.equal(startingSig.guardianIndex, endingSig.guardianIndex);

        // try to verify the modified VM2
        failed = false;
        try {
            const integrationContract = await deployMockIntegrationContract(accounts[0]);
            await integrationContract.methods.parseAndVerifyVM2("0x" + endingVM2).send({
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

    it("should not verify a VM2 with an additional observation", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const signers = [testSigner1PK];
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
        const parsedStartingVM2 = await initialized.methods.parseVM2("0x" + startingVM2).call();
        const parsedEndingVM2 = await initialized.methods.parseVM2("0x" + endingVM2).call();

        // confirm that both VMs have the same hashes (except for the last additional observation)
        assert.equal(parsedStartingVM2.header.hashes[0], parsedEndingVM2.header.hashes[0]);
        assert.equal(parsedStartingVM2.header.hashes[1], parsedEndingVM2.header.hashes[1]);
        assert.equal(parsedStartingVM2.header.hashes[2], parsedEndingVM2.header.hashes[2]);

        // confirm that the original VM has one less hash than the modified one
        assert.equal(parsedStartingVM2.header.hashes.length, parsedEndingVM2.header.hashes.length - 1);

        // confirm both VMs have the same signatures
        const startingSig = parsedStartingVM2.header.signatures[0];
        const endingSig = parsedEndingVM2.header.signatures[0];

        assert.equal(startingSig.r, endingSig.r);
        assert.equal(startingSig.s, endingSig.s);
        assert.equal(startingSig.v, endingSig.v);
        assert.equal(startingSig.guardianIndex, endingSig.guardianIndex);

        // try to verify the modified VM2
        failed = false;
        try {
            const integrationContract = await deployMockIntegrationContract(accounts[0]);
            await integrationContract.methods.parseAndVerifyVM2("0x" + endingVM2).send({
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

    it("should not verify a VM2 with reorganized observations", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const signers = [testSigner1PK];
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
        const parsedStartingVM2 = await initialized.methods.parseVM2("0x" + startingVM2).call();
        const parsedEndingVM2 = await initialized.methods.parseVM2("0x" + endingVM2).call();

        // confirm that both VMs have the same hashes (but in different orders)
        assert.equal(parsedStartingVM2.header.hashes[2], parsedEndingVM2.header.hashes[0]);
        assert.equal(parsedStartingVM2.header.hashes[0], parsedEndingVM2.header.hashes[1]);
        assert.equal(parsedStartingVM2.header.hashes[1], parsedEndingVM2.header.hashes[2]);

        // confirm both VMs have the same number of hashes
        assert.equal(parsedStartingVM2.header.hashes.length, parsedEndingVM2.header.hashes.length);

        // confirm both VMs have the same signatures
        const startingSig = parsedStartingVM2.header.signatures[0];
        const endingSig = parsedEndingVM2.header.signatures[0];

        assert.equal(startingSig.r, endingSig.r);
        assert.equal(startingSig.s, endingSig.s);
        assert.equal(startingSig.v, endingSig.v);
        assert.equal(startingSig.guardianIndex, endingSig.guardianIndex);

        // try to verify the modified VM2
        failed = false;
        try {
            const integrationContract = await deployMockIntegrationContract(accounts[0]);
            await integrationContract.methods.parseAndVerifyVM2("0x" + endingVM2).send({
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

        // deploy the mock integration contract
        const integrationContract = await deployMockIntegrationContract(accounts[0]);

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
        const parsedVM2 = await initialized.methods.parseVM2("0x" + vm2).call();
        await integrationContract.methods.parseAndVerifyVM2("0x" + vm2).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // confirm that each observation is parsed and verified correctly by parseAndVerifyVAA
        for (let i = 0; i < parsedVM2.observations.length; i++) {
            let parsedObservation;
            try {
                parsedObservation = await initialized.methods.parseAndVerifyVAA(
                    parsedVM2.observations[i]
                ).call();
            } catch (err) {
                console.log(err);
                assert.fail("parseAndVerifyVAA failed");
            }

            // compare the test observation with the parsed output from parseAndVerifyVAA
            assert.equal(parsedObservation.observation.timestamp, TEST_OBSERVATIONS[i].timestamp);
            assert.equal(parsedObservation.observation.nonce, TEST_OBSERVATIONS[i].nonce);
            assert.equal(parsedObservation.observation.emitterChainId, TEST_OBSERVATIONS[i].emitterChainId);
            assert.equal(parsedObservation.observation.emitterAddress, TEST_OBSERVATIONS[i].emitterAddress);
            assert.equal(parsedObservation.observation.sequence, TEST_OBSERVATIONS[i].sequence);
            assert.equal(parsedObservation.observation.consistencyLevel, TEST_OBSERVATIONS[i].consistencyLevel);
            assert.equal(parsedObservation.observation.payload, TEST_OBSERVATIONS[i].payload);

            // confirm the VM3 was verified
            assert.ok(parsedObservation.valid);
            assert.equal(parsedObservation.reason, "");
        }

        // clear the batch cache
        await initialized.methods.clearBatchCache(parsedVM2.header).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });
    })

    it("should not verify VM3s after the batch cache is cleared", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        // deploy the mock integration contract
        const integrationContract = await deployMockIntegrationContract(accounts[0]);

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
        const parsedVM2 = await initialized.methods.parseVM2("0x" + vm2).call();
        await integrationContract.methods.parseAndVerifyVM2("0x" + vm2).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // verify the first two VM3s
        for (let i = 0; i < parsedVM2.observations.length - 1; i++) {
            try {
                await initialized.methods.parseAndVerifyVAA(
                    parsedVM2.observations[i]
                ).call();
            } catch (err) {
                console.log(err);
                assert.fail("parseAndVerifyVAA failed");
            }
        }

        // clear the batch cache and try to verify the last VM3 (it should fail)
        await initialized.methods.clearBatchCache(parsedVM2.header).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        const result = await initialized.methods.parseAndVerifyVAA(
            parsedVM2.observations[parsedVM2.length-1]
        ).call();

        assert.equal(result.reason, "Could not find hash in cache");
        assert.ok(!result.valid);
    })

    it("should verify VM3s from a contract correctly", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();

        // deploy the mock integration contract
        const integrationContract = await deployMockIntegrationContract(accounts[0]);

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
            await integrationContract.methods.consumeBatchVAA("0x" + vm2).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });
        } catch (err) {
            console.log(err);
            assert.fail("consumeBatchVAA failed");
        }

        // fetch the payloads that are stored in the mock integration contract
        const payloadResults = await integrationContract.methods.getPayloads().call();

        for (let i = 0; i < TEST_OBSERVATIONS.length; i++) {
            // validate payloads
            assert.equal(payloadResults[i], TEST_OBSERVATIONS[i].payload);

            // confirm that the batch cache was cleared
            const observationHash = doubleKeccak256(encodeObservation(TEST_OBSERVATIONS[i]));

            // query the contract using the hash
            const hashIsCached = await initialized.methods.verifiedHashCached(observationHash).call();
            assert.ok(!hashIsCached);
        }
    })

    it("parseAndVerifyVAA should be backwards compatible and correctly parse VM1s", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const testObservation = TEST_OBSERVATIONS[0];

        // simulate signing the VM1
        const vm = await signAndEncodeVM(
            testObservation.timestamp,
            testObservation.nonce,
            testObservation.emitterChainId,
            testObservation.emitterAddress,
            testObservation.sequence,
            testObservation.payload,
            [
                testSigner1PK,
            ],
            0,
            testObservation.consistencyLevel
        );

        let result
        try {
            result = await initialized.methods.parseAndVerifyVAA("0x" + vm).call();
        } catch (err) {
            console.log(err)
            assert.fail("parseAndVerifyVAA failed");
        }

        // verify the observation returned by parseAndVerifyVAA
        assert.equal(result.observation.timestamp, testObservation.timestamp);
        assert.equal(result.observation.nonce, testObservation.nonce);
        assert.equal(result.observation.emitterChainId, testObservation.emitterChainId);
        assert.equal(result.observation.emitterAddress, testObservation.emitterAddress);
        assert.equal(result.observation.payload, testObservation.payload);
        assert.equal(result.observation.sequence, testObservation.sequence);
        assert.equal(result.observation.consistencyLevel, testObservation.consistencyLevel);

        // confirm that the VM1 was verified
        assert.equal(result.valid, true);
        assert.equal(result.reason, "");
    })

    it("parses and verifies VM1s after the batch cache is cleared", async function () {
        const initialized = new web3.eth.Contract(ImplementationFullABI, Wormhole.address);
        const accounts = await web3.eth.getAccounts();
        const signers = [testSigner1PK];

        // deploy the mock integration contract
        const integrationContract = await deployMockIntegrationContract(accounts[0]);

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
        const parsedVM2 = await initialized.methods.parseVM2("0x" + vm2).call();
        await integrationContract.methods.parseAndVerifyVM2("0x" + vm2).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // parse and verify the first VM3
        // use the parseAndVerifyVM3 function to retrieve the hash
        let parsedVM3;
        try {
            parsedVM3 = await initialized.methods.parseAndVerifyVM3(
                parsedVM2.observations[0]
            ).call();
        } catch (err) {
            console.log(err);
            assert.fail("parseAndVerifyVAA failed");
        }

        // clear the batch cache
        await initialized.methods.clearBatchCache(parsedVM2.header).send({
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
            assert.fail("parseAndVerifyVAA failed");
        }

        // confirm that the VM1 and VM3 are the same observation and both were both verified
        assert.ok(parsedVM1.valid);
        assert.ok(parsedVM3.valid);
        assert.equal(parsedVM1.vm.hash, parsedVM3.vm.hash);

        assert.equal(parsedVM1.vm.timestamp, parsedVM3.vm.observation.timestamp);
        assert.equal(parsedVM1.vm.nonce, parsedVM3.vm.observation.nonce);
        assert.equal(parsedVM1.vm.emitterChainId, parsedVM3.vm.observation.emitterChainId);
        assert.equal(parsedVM1.vm.emitterAddress, parsedVM3.vm.observation.emitterAddress);
        assert.equal(parsedVM1.vm.sequence, parsedVM3.vm.observation.sequence);
        assert.equal(parsedVM1.vm.consistencyLevel, parsedVM3.vm.observation.consistencyLevel);
        assert.equal(parsedVM1.vm.payload, parsedVM3.vm.observation.payload);
    })
});

function copySignaturesToVM2(fromVM, toVM) {
    // index of the signature length (number of signers for the VM)
    let index = 10;

    // grab the number of signatures for each VM
    sigCountFrom = parseInt(fromVM.slice(index, index+2), 16);
    sigCountTo = parseInt(toVM.slice(index, index+2), 16);
    index += 2

    // grab the signatures for the startVM (each signature is 66 bytes (132 string representation))
    const fromVMSigs = fromVM.slice(index, index + (132 * sigCountFrom));

    // create a new VAA with the signatures from startVM
    const resultVM = toVM.slice(0, index) + fromVMSigs + toVM.slice(index + (132 * sigCountTo));
    return resultVM;
}

async function deployMockIntegrationContract(account) {
    // deploy and intialize mock integration contract
    const mockIntegrationAddress = (await MockIntegration.new()).address;
    const integrationContract = new web3.eth.Contract(MockIntegration.abi, mockIntegrationAddress);
    await integrationContract.methods.setup(Wormhole.address).send({
        value: 0,
        from: account
    });
    return integrationContract;
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
    for (const observation of observationsArray) {
        // encode the observation
        const observationBytes = encodeObservation(observation);

        // hash the observation
        const hash = doubleKeccak256(observationBytes);
        observationHashes += hash.substring(2);

        // grab the length of the observation and add it to the observation bytestring
        // divide observationBytes by two to convert string representation length to bytes
        const observationLen = web3.eth.abi.encodeParameter("uint32", observationBytes.length / 2).substring(2 + (64 - 8))
        encodedObservationsWithLengthPrefix += observationLen + observationBytes;
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