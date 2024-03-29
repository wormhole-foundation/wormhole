const jsonfile = require('jsonfile');
const elliptic = require('elliptic');
const path = require('path');

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
    const testEvmChainId = "1337";
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
            assert.include(e.message, 'revert signature indices must be ascending')
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
            assert.include(e.message, "revert Invalid key");
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

        assert.equal(BigInt(before), BigInt(Implementation.address));

        let set = await initialized.methods.submitContractUpgrade("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 1000000
        });

        let after = await web3.eth.getStorageAt(Wormhole.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(BigInt(after), BigInt(mock.address));

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
            assert.include(e.message, "revert not a fork")
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
            assert.include(e.message, "revert not signed by current guardian set")
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
            assert.include(e.message, "revert wrong governance chain")
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
            assert.include(e.message, "revert wrong governance contract")
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
            assert.include(e.message, "revert governance action already consumed")
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
            assert.include(e.message, "revert invalid fork")
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

        assert.equal(BigInt(before), BigInt(lastDeployed.address));

        let set = await initialized.methods.submitContractUpgrade("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 1000000
        });

        let after = await web3.eth.getStorageAt(Wormhole.address, "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

        assert.equal(BigInt(after), BigInt(mock.address));

        const mockImpl = new web3.eth.Contract(MockImplementation.abi, Wormhole.address);

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
