const jsonfile = require('jsonfile');
const elliptic = require('elliptic');
const BigNumber = require('bignumber.js');

const Wormhole = artifacts.require("Wormhole");
const TokenBridge = artifacts.require("TokenBridge");
const BridgeImplementation = artifacts.require("BridgeImplementation");
const TokenImplementation = artifacts.require("TokenImplementation");
const FeeToken = artifacts.require("FeeToken");
const MockBridgeImplementation = artifacts.require("MockBridgeImplementation");
const MockWETH9 = artifacts.require("MockWETH9");

const testSigner1PK = "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";
const testSigner2PK = "892330666a850761e7370376430bb8c2aa1494072d3bfeaed0c4fa3d5a9135fe";

const WormholeImplementationFullABI = jsonfile.readFileSync("build/contracts/Implementation.json").abi
const BridgeImplementationFullABI = jsonfile.readFileSync("build/contracts/BridgeImplementation.json").abi
const TokenImplementationFullABI = jsonfile.readFileSync("build/contracts/TokenImplementation.json").abi

contract("ShutdownSwitch", function () {
    const testChainId = "2";
    const testGovernanceChainId = "1";
    const testGovernanceContract = "0x0000000000000000000000000000000000000000000000000000000000000004";

    it("should block non-guardians from casting a shutdown vote", async function () {
        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, TokenBridge.address);

        // Cast a vote using the wrong key, and it should fail.
        let voteFailed = false;
        try {
            await initialized.methods.castShutdownVote(testChainId, false).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });            
        } catch (error) {
            assert.equal(error.message, "Returned error: VM Exception while processing transaction: revert you are not a registered voter")
            voteFailed = true
        }

        assert.ok(voteFailed)
        assert.equal((await initialized.methods.enabledFlag().call()), true)

        // Can't check the number of no votes or required votes until after we have a successful vote.
    })

    it("should upgrade to four guardians", async function () {
        const accounts = await web3.eth.getAccounts();

        // Create a guardian set of four, which will require two disable votes to suspend transfers.
        const wormhole = new web3.eth.Contract(WormholeImplementationFullABI, Wormhole.address);

        const timestamp = 1000;
        const nonce = 1001;
        const emitterChainId = testGovernanceChainId;
        const emitterAddress = testGovernanceContract
        let data = "0x00000000000000000000000000000000000000000000000000000000436f726502";
        let oldIndex = Number(await wormhole.methods.getCurrentGuardianSetIndex().call());

        data += [
            web3.eth.abi.encodeParameter("uint16", testChainId).substring(2 + (64 - 4)),
            web3.eth.abi.encodeParameter("uint32", oldIndex + 1).substring(2 + (64 - 8)),
            web3.eth.abi.encodeParameter("uint8", 4).substring(2 + (64 - 2)),
            web3.eth.abi.encodeParameter("address", accounts[0]).substring(2 + (64 - 40)),
            web3.eth.abi.encodeParameter("address", accounts[1]).substring(2 + (64 - 40)),
            web3.eth.abi.encodeParameter("address", accounts[2]).substring(2 + (64 - 40)),
            web3.eth.abi.encodeParameter("address", accounts[3]).substring(2 + (64 - 40)),            
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

        let set = await wormhole.methods.submitNewGuardianSet("0x" + vm).send({
            value: 0,
            from: accounts[0],
            gasLimit: 1000000
        });
    })

    //////////////// NOTE: Every test after this assumes four guardians! ///////////////////////////////

    it("should take two guardian votes to disable transfers", async function () {
        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, TokenBridge.address);
        clearAllVotes(initialized, accounts);
        assert.equal((await initialized.methods.enabledFlag().call()), true)

        // This first vote should succeed, but we should still be enabled.
        await initialized.methods.castShutdownVote(testChainId, false).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        assert.equal((await initialized.methods.numVotesToDisable().call()), 1)
        assert.equal((await initialized.methods.requiredVotesToDisable().call()), 2)
        assert.equal((await initialized.methods.enabledFlag().call()), true)

        // A duplicate vote should change nothing.
        await initialized.methods.castShutdownVote(testChainId, false).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        assert.equal((await initialized.methods.numVotesToDisable().call()), 1)
        assert.equal((await initialized.methods.requiredVotesToDisable().call()), 2)
        assert.equal((await initialized.methods.enabledFlag().call()), true)

        // But a second vote should suspend transfers.
        await initialized.methods.castShutdownVote(testChainId, false).send({
            value: 0,
            from: accounts[1],
            gasLimit: 2000000
        });

        assert.equal((await initialized.methods.numVotesToDisable().call()), 2)
        assert.equal((await initialized.methods.requiredVotesToDisable().call()), 2)
        assert.equal((await initialized.methods.enabledFlag().call()), false)
    })

    it("should prevent a replay attack using a vote from another chain", async function () {
        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, TokenBridge.address);
        clearAllVotes(initialized, accounts);

         // Cast a vote using the wrong chain id, and it should fail.
         let voteFailed = false;
         try {
             await initialized.methods.castShutdownVote(testChainId + 1, false).send({
                 value: 0,
                 from: accounts[0],
                 gasLimit: 2000000
             });            
         } catch (error) {
             assert.equal(error.message, "Returned error: VM Exception while processing transaction: revert invalid chain id")
             voteFailed = true
         }

         assert.ok(voteFailed)

         // We should still be enabled.
         assert.equal((await initialized.methods.enabledFlag().call()), true)
    })

    let vaa = null;

    it("should reject transfers when they are disabled", async function () {
        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, TokenBridge.address);
        clearAllVotes(initialized, accounts);
        assert.equal((await initialized.methods.enabledFlag().call()), true)

        // Set up, mint and approve tokens.
        const token = new web3.eth.Contract(TokenImplementation.abi, TokenImplementation.address);
        await token.methods.initialize(
            "TestToken",
            "TT",
            18,
            0,

            accounts[0],

            0,
            "0x0"
        ).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        const amount = "1000000000000000000";     

        await token.methods.mint(accounts[0], amount).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        await token.methods.approve(TokenBridge.address, amount).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // This transfer should succeed.
        await initialized.methods.transferTokens(
            TokenImplementation.address,
            "10000",
            "10",
            "0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e",
            "0",
            "0"
        ).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        // Get the VAA from the successful transfer so we can try to redeem it.
        const wormhole = new web3.eth.Contract(WormholeImplementationFullABI, Wormhole.address);
        const log = (await wormhole.getPastEvents('LogMessagePublished', {
            fromBlock: 'latest'
        }))[0].returnValues
        vaa = log.payload.substr(2);          

        // Cast two votes to disable transfers.
        await initialized.methods.castShutdownVote(testChainId, false).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        assert.equal((await initialized.methods.numVotesToDisable().call()), 1)
        assert.equal((await initialized.methods.enabledFlag().call()), true)

        await initialized.methods.castShutdownVote(testChainId, false).send({
            value: 0,
            from: accounts[1],
            gasLimit: 2000000
        });
        
        // Make sure transfers are now disabled.
        assert.equal((await initialized.methods.numVotesToDisable().call()), 2)
        assert.equal((await initialized.methods.enabledFlag().call()), false)

        // This transfer should be blocked.
        let transferShouldFail = false;
        try {
            await initialized.methods.transferTokens(
                TokenImplementation.address,
                "10000",
                "10",
                "0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e",
                "0",
                "0"
            ).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            });         
        } catch (error) {
            assert.equal(error.message, "Returned error: VM Exception while processing transaction: revert transfers are temporarily disabled")
            transferShouldFail = true
        }

        assert.ok(transferShouldFail)
    })

    it("a vote changing back to enabled should allow transfers again", async function () {
        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, TokenBridge.address);

        // This assumes the previous test left us disabled.
        assert.equal((await initialized.methods.numVotesToDisable().call()), 2)
        assert.equal((await initialized.methods.enabledFlag().call()), false)

        // Change one vote to enabled and our status should change back to enabled.
        await initialized.methods.castShutdownVote(testChainId, true).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        assert.equal((await initialized.methods.numVotesToDisable().call()), 1)
        assert.equal((await initialized.methods.enabledFlag().call()), true)

        // And now the transfer should go through.
        await initialized.methods.transferTokens(
            TokenImplementation.address,
            "10000",
            "10",
            "0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e",
            "0",
            "0"
        ).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });      
    })

    it("should reject redeems when they are disabled", async function () {
        // This test assumes the previous test initialized the vaa variable.
        assert.ok(vaa != null)

        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, TokenBridge.address);
        clearAllVotes(initialized, accounts);
        assert.equal((await initialized.methods.enabledFlag().call()), true)

        // Redeem of this transfer should not be blocked by shutdown switch. It will instead fail with
        // "no quorum", but that's okay for this test, since that check happens after the enable check.
        try {
            await initialized.methods.completeTransfer("0x" + vaa).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            })
        } catch (error) {
            assert.equal(error.message, "Returned error: VM Exception while processing transaction: revert no quorum")
        }            

        // Cast two votes to disable transfers.
        await initialized.methods.castShutdownVote(testChainId, false).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        assert.equal((await initialized.methods.numVotesToDisable().call()), 1)
        assert.equal((await initialized.methods.enabledFlag().call()), true)

        await initialized.methods.castShutdownVote(testChainId, false).send({
            value: 0,
            from: accounts[1],
            gasLimit: 2000000
        });
        
        // Make sure transfers are now disabled.
        assert.equal((await initialized.methods.numVotesToDisable().call()), 2)
        assert.equal((await initialized.methods.enabledFlag().call()), false)

        // This redeem should be blocked by shutdown switch rather than "no quorum".
        try {
            await initialized.methods.completeTransfer("0x" + vaa).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            })
        } catch (error) {
            assert.equal(error.message, "Returned error: VM Exception while processing transaction: revert transfers are temporarily disabled")
        }
    })

    it("a vote changing back to enabled should allow redeems again", async function () {
        const accounts = await web3.eth.getAccounts();
        const initialized = new web3.eth.Contract(BridgeImplementationFullABI, TokenBridge.address);

        // This assumes the previous test left us disabled.
        assert.equal((await initialized.methods.numVotesToDisable().call()), 2)
        assert.equal((await initialized.methods.enabledFlag().call()), false)
        
        // Change one vote to enabled and our status should change back to enabled.
        await initialized.methods.castShutdownVote(testChainId, true).send({
            value: 0,
            from: accounts[0],
            gasLimit: 2000000
        });

        assert.equal((await initialized.methods.numVotesToDisable().call()), 1)
        assert.equal((await initialized.methods.enabledFlag().call()), true)

        // The redeem should be back to being blocked by "no quorum" rather than by shutdown switch.
        try {
            await initialized.methods.completeTransfer("0x" + vaa).send({
                value: 0,
                from: accounts[0],
                gasLimit: 2000000
            })
        } catch (error) {
            assert.equal(error.message, "Returned error: VM Exception while processing transaction: revert no quorum")
        }         
    })

    async function clearAllVotes(initialized, accounts) {
        for (let idx = 0; (idx < 4); ++idx) {
            await initialized.methods.castShutdownVote(testChainId, true).send({
                value: 0,
                from: accounts[idx],
                gasLimit: 2000000
            });
        }
    }
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
