const jsonfile = require("jsonfile");
const elliptic = require("elliptic");
const path = require("path");

const WormholeReceiver = artifacts.require("WormholeReceiver");
const MockImplementation = artifacts.require("MockImplementation");
const Implementation = artifacts.require("Implementation");

const testSigner1PK =
  "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";
const testSigner2PK =
  "892330666a850761e7370376430bb8c2aa1494072d3bfeaed0c4fa3d5a9135fe";
const testSigner3PK =
  "87b45997ea577b93073568f06fc4838cffc1d01f90fc4d57f936957f3c4d99fb";
const testBadSigner1PK =
  "87b45997ea577b93073568f06fc4838cffc1d01f90fc4d57f936957f3c4d99fc";

const core = "0x" + Buffer.from("Core").toString("hex").padStart(64, 0);
const actionGuardianSetUpgrade = "02";

const ImplementationFullABI = jsonfile.readFileSync(
  "build/contracts/Implementation.json"
).abi;

// Taken from https://medium.com/fluidity/standing-the-time-of-test-b906fcc374a9
advanceTimeAndBlock = async (time) => {
  await advanceTime(time);
  await advanceBlock();

  return Promise.resolve(web3.eth.getBlock("latest"));
};

advanceTime = (time) => {
  return new Promise((resolve, reject) => {
    web3.currentProvider.send(
      {
        jsonrpc: "2.0",
        method: "evm_increaseTime",
        params: [time],
        id: new Date().getTime(),
      },
      (err, result) => {
        if (err) {
          return reject(err);
        }
        return resolve(result);
      }
    );
  });
};

advanceBlock = () => {
  return new Promise((resolve, reject) => {
    web3.currentProvider.send(
      {
        jsonrpc: "2.0",
        method: "evm_mine",
        id: new Date().getTime(),
      },
      (err, result) => {
        if (err) {
          return reject(err);
        }
        const newBlockHash = web3.eth.getBlock("latest").hash;

        return resolve(newBlockHash);
      }
    );
  });
};

contract("WormholeReceiver", function () {
  const testSigner1 = web3.eth.accounts.privateKeyToAccount(testSigner1PK);
  const testSigner2 = web3.eth.accounts.privateKeyToAccount(testSigner2PK);
  const testSigner3 = web3.eth.accounts.privateKeyToAccount(testSigner3PK);
  const testChainId = "2";
  const testGovernanceChainId = "1";
  const testGovernanceContract =
    "0x0000000000000000000000000000000000000000000000000000000000000004";

  it("should be initialized with the correct signers and values", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );

    const index = await initialized.methods.getCurrentGuardianSetIndex().call();
    const set = await initialized.methods.getGuardianSet(index).call();

    // check set
    assert.lengthOf(set[0], 1);
    assert.equal(set[0][0], testSigner1.address);

    // check expiration
    assert.equal(set.expirationTime, "0");

    // governance
    const governanceChainId = await initialized.methods
      .governanceChainId()
      .call();
    assert.equal(governanceChainId, testGovernanceChainId);
    const governanceContract = await initialized.methods
      .governanceContract()
      .call();
    assert.equal(governanceContract, testGovernanceContract);
  });

  it("parses VMs correctly", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );

    const timestamp = 1000;
    const nonce = 1001;
    const emitterChainId = 11;
    const emitterAddress =
      "0x0000000000000000000000000000000000000000000000000000000000000eee";
    const data = "0xaaaaaa";

    const vm = await signAndEncodeVM(
      timestamp,
      nonce,
      emitterChainId,
      emitterAddress,
      1337,
      data,
      [testSigner1PK],
      0,
      2
    );

    let result;
    try {
      result = await initialized.methods.parseAndVerifyVM("0x" + vm).call();
    } catch (err) {
      console.log(err);
      assert.fail("parseAndVerifyVM failed");
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
  });

  it("should fail quorum on VMs with no signers", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );

    const timestamp = 1000;
    const nonce = 1001;
    const emitterChainId = 11;
    const emitterAddress =
      "0x0000000000000000000000000000000000000000000000000000000000000eee";
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
    assert.equal(result[1], false);
    assert.equal(result[2], "no quorum");
  });

  it("should fail to verify on VMs with bad signer", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );

    const timestamp = 1000;
    const nonce = 1001;
    const emitterChainId = 11;
    const emitterAddress =
      "0x0000000000000000000000000000000000000000000000000000000000000eee";
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
    assert.equal(result[1], false);
    assert.equal(result[2], "VM signature invalid");
  });

  it("should error on VMs with invalid guardian set index", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );

    const timestamp = 1000;
    const nonce = 1001;
    const emitterChainId = 11;
    const emitterAddress =
      "0x0000000000000000000000000000000000000000000000000000000000000eee";
    const data = "0xaaaaaa";

    const vm = await signAndEncodeVM(
      timestamp,
      nonce,
      emitterChainId,
      emitterAddress,
      1337,
      data,
      [testSigner1PK],
      200,
      2
    );

    let result = await initialized.methods.parseAndVerifyVM("0x" + vm).call();
    assert.equal(result[1], false);
    assert.equal(result[2], "invalid guardian set");
  });

  it("should revert on VMs with duplicate non-monotonic signature indexes", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );

    const timestamp = 1000;
    const nonce = 1001;
    const emitterChainId = 11;
    const emitterAddress =
      "0x0000000000000000000000000000000000000000000000000000000000000eee";
    const data = "0xaaaaaa";

    const vm = await signAndEncodeVMFixedIndex(
      timestamp,
      nonce,
      emitterChainId,
      emitterAddress,
      1337,
      data,
      [testSigner1PK, testSigner1PK, testSigner1PK],
      0,
      2
    );

    try {
      await initialized.methods.parseAndVerifyVM("0x" + vm).call();
      assert.fail("accepted signature indexes being the same in a VM");
    } catch (e) {
      assert.equal(
        e.data[Object.keys(e.data)[0]].reason,
        "signature indices must be ascending"
      );
    }
  });

  it("should accept a new guardian set", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );
    const accounts = await web3.eth.getAccounts();

    const timestamp = 1000;
    const nonce = 1001;
    const emitterChainId = testGovernanceChainId;
    const emitterAddress = testGovernanceContract;

    let oldIndex = Number(
      await initialized.methods.getCurrentGuardianSetIndex().call()
    );

    data = [
      // Core
      core,
      // Action 2 (Guardian Set Upgrade)
      actionGuardianSetUpgrade,
      web3.eth.abi
        .encodeParameter("uint16", testChainId)
        .substring(2 + (64 - 4)),
      web3.eth.abi
        .encodeParameter("uint32", oldIndex + 1)
        .substring(2 + (64 - 8)),
      web3.eth.abi.encodeParameter("uint8", 3).substring(2 + (64 - 2)),
      web3.eth.abi
        .encodeParameter("address", testSigner1.address)
        .substring(2 + (64 - 40)),
      web3.eth.abi
        .encodeParameter("address", testSigner2.address)
        .substring(2 + (64 - 40)),
      web3.eth.abi
        .encodeParameter("address", testSigner3.address)
        .substring(2 + (64 - 40)),
    ].join("");

    const vm = await signAndEncodeVM(
      timestamp,
      nonce,
      emitterChainId,
      emitterAddress,
      0,
      data,
      [testSigner1PK],
      0,
      2
    );

    let set = await initialized.methods.submitNewGuardianSet("0x" + vm).send({
      value: 0,
      from: accounts[0],
      gasLimit: 1000000,
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

    let oldGuardians = await initialized.methods
      .getGuardianSet(oldIndex)
      .call();

    const time = (await web3.eth.getBlock("latest")).timestamp;

    // old guardian set expiry is set
    assert.ok(
      oldGuardians.expirationTime > Number(time) + 86000 &&
        oldGuardians.expirationTime < Number(time) + 88000
    );
  });

  it("should accept smart contract upgrades", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );
    const accounts = await web3.eth.getAccounts();

    const mock = await MockImplementation.new();

    let before = await web3.eth.getStorageAt(
      WormholeReceiver.address,
      "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"
    );

    assert.equal(before.toLowerCase(), Implementation.address.toLowerCase());

    // attempt to upgrade the implementation from non-owner account
    let failed = false;
    try {
      await initialized.methods.upgradeImplementation(mock.address).send({
        value: 0,
        from: accounts[1],
        gasLimit: 1000000,
      });
    } catch (e) {
      assert.equal(
        e.message,
        "Returned error: VM Exception while processing transaction: revert caller is not the owner"
      );
      failed = true;
    }

    assert.ok(failed);

    let set = await initialized.methods
      .upgradeImplementation(mock.address)
      .send({
        value: 0,
        from: accounts[0],
        gasLimit: 1000000,
      });

    let after = await web3.eth.getStorageAt(
      WormholeReceiver.address,
      "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"
    );

    assert.equal(after.toLowerCase(), mock.address.toLowerCase());

    const mockImpl = new web3.eth.Contract(
      MockImplementation.abi,
      WormholeReceiver.address
    );

    let isUpgraded = await mockImpl.methods
      .testNewImplementationActive()
      .call();

    assert.ok(isUpgraded);
  });

  it("should allow the owner to transfer ownership", async function () {
    const accounts = await web3.eth.getAccounts();
    const currentOwner = accounts[0];
    const newOwner = accounts[1];

    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );

    // transfer ownership
    await initialized.methods.transferOwnership(newOwner).send({
      value: "0",
      from: currentOwner,
      gasLimit: 1000000,
    });

    // check getters after the action
    let contractOwner = await initialized.methods.owner().call();

    assert.equal(contractOwner, newOwner);

    // make sure only the owner can transfer ownership
    let failed = false;
    try {
      await initialized.methods.transferOwnership(newOwner).send({
        value: "0",
        from: currentOwner, // no longer the current owner
        gasLimit: 1000000,
      });
    } catch (e) {
      assert.equal(
        e.message,
        "Returned error: VM Exception while processing transaction: revert caller is not the owner"
      );
      failed = true;
    }

    assert.ok(failed);

    // revert ownership back to currentOwner
    await initialized.methods.transferOwnership(currentOwner).send({
      value: "0",
      from: newOwner,
      gasLimit: 1000000,
    });

    // check getters before the action
    contractOwner = await initialized.methods.owner().call();

    assert.equal(contractOwner, currentOwner);
  });

  it("should revert governance packets from old guardian set", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );
    const accounts = await web3.eth.getAccounts();

    let oldIndex = Number(
      await initialized.methods.getCurrentGuardianSetIndex().call()
    );

    data = [
      // Core
      core,
      // Action 2 (Guardian Set Upgrade)
      actionGuardianSetUpgrade,
      web3.eth.abi
        .encodeParameter("uint16", testChainId)
        .substring(2 + (64 - 4)),
      web3.eth.abi
        .encodeParameter("uint32", oldIndex + 1)
        .substring(2 + (64 - 8)),
      web3.eth.abi.encodeParameter("uint8", 3).substring(2 + (64 - 2)),
      web3.eth.abi
        .encodeParameter("address", testSigner1.address)
        .substring(2 + (64 - 40)),
      web3.eth.abi
        .encodeParameter("address", testSigner2.address)
        .substring(2 + (64 - 40)),
      web3.eth.abi
        .encodeParameter("address", testSigner3.address)
        .substring(2 + (64 - 40)),
    ].join("");

    const vm = await signAndEncodeVM(
      0,
      0,
      testGovernanceChainId,
      testGovernanceContract,
      0,
      data,
      [testSigner1PK],
      0,
      2
    );

    let failed = false;
    try {
      await initialized.methods.submitNewGuardianSet("0x" + vm).send({
        value: 0,
        from: accounts[0],
        gasLimit: 1000000,
      });
      assert.fail("governance packet of old guardian set accepted");
    } catch (e) {
      assert.equal(
        e.data[Object.keys(e.data)[0]].reason,
        "not signed by current guardian set"
      );
    }
  });

  it("should time out old gardians", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );

    const timestamp = 1000;
    const nonce = 1001;
    const emitterChainId = 11;
    const emitterAddress =
      "0x0000000000000000000000000000000000000000000000000000000000000eee";
    const data = "0xaaaaaa";

    const vm = await signAndEncodeVM(
      timestamp,
      nonce,
      emitterChainId,
      emitterAddress,
      0,
      data,
      [testSigner1PK],
      0,
      2
    );

    // this should pass
    const current = await initialized.methods
      .parseAndVerifyVM("0x" + vm)
      .call();

    assert.equal(current.valid, true);

    await advanceTimeAndBlock(100000);

    const expired = await initialized.methods
      .parseAndVerifyVM("0x" + vm)
      .call();

    assert.equal(expired.valid, false);
  });

  it("should revert governance packets from wrong governance chain", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );
    const accounts = await web3.eth.getAccounts();

    let oldIndex = Number(
      await initialized.methods.getCurrentGuardianSetIndex().call()
    );

    data = [
      // Core
      core,
      // Action 2 (Guardian Set Upgrade)
      actionGuardianSetUpgrade,
      web3.eth.abi
        .encodeParameter("uint16", testChainId)
        .substring(2 + (64 - 4)),
      web3.eth.abi
        .encodeParameter("uint32", oldIndex + 1)
        .substring(2 + (64 - 8)),
      web3.eth.abi.encodeParameter("uint8", 3).substring(2 + (64 - 2)),
      web3.eth.abi
        .encodeParameter("address", testSigner1.address)
        .substring(2 + (64 - 40)),
      web3.eth.abi
        .encodeParameter("address", testSigner2.address)
        .substring(2 + (64 - 40)),
      web3.eth.abi
        .encodeParameter("address", testSigner3.address)
        .substring(2 + (64 - 40)),
    ].join("");

    const vm = await signAndEncodeVM(
      0,
      0,
      999,
      testGovernanceContract,
      0,
      data,
      [testSigner1PK, testSigner2PK, testSigner3PK],
      1,
      2
    );

    try {
      await initialized.methods.submitNewGuardianSet("0x" + vm).send({
        value: 0,
        from: accounts[0],
        gasLimit: 1000000,
      });
      assert.fail("governance packet from wrong governance chain accepted");
    } catch (e) {
      assert.equal(
        e.data[Object.keys(e.data)[0]].reason,
        "wrong governance chain"
      );
    }
  });

  it("should revert governance packets from wrong governance contract", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );
    const accounts = await web3.eth.getAccounts();

    let oldIndex = Number(
      await initialized.methods.getCurrentGuardianSetIndex().call()
    );

    data = [
      // Core
      core,
      // Action 2 (Guardian Set Upgrade)
      actionGuardianSetUpgrade,
      web3.eth.abi
        .encodeParameter("uint16", testChainId)
        .substring(2 + (64 - 4)),
      web3.eth.abi
        .encodeParameter("uint32", oldIndex + 1)
        .substring(2 + (64 - 8)),
      web3.eth.abi.encodeParameter("uint8", 3).substring(2 + (64 - 2)),
      web3.eth.abi
        .encodeParameter("address", testSigner1.address)
        .substring(2 + (64 - 40)),
      web3.eth.abi
        .encodeParameter("address", testSigner2.address)
        .substring(2 + (64 - 40)),
      web3.eth.abi
        .encodeParameter("address", testSigner3.address)
        .substring(2 + (64 - 40)),
    ].join("");

    const vm = await signAndEncodeVM(
      0,
      0,
      testGovernanceChainId,
      core,
      0,
      data,
      [testSigner1PK, testSigner2PK, testSigner3PK],
      1,
      2
    );

    try {
      await initialized.methods.submitNewGuardianSet("0x" + vm).send({
        value: 0,
        from: accounts[0],
        gasLimit: 1000000,
      });
      assert.fail("governance packet from wrong governance contract accepted");
    } catch (e) {
      assert.equal(
        e.data[Object.keys(e.data)[0]].reason,
        "wrong governance contract"
      );
    }
  });

  it("should revert on governance packets that already have been applied", async function () {
    const initialized = new web3.eth.Contract(
      ImplementationFullABI,
      WormholeReceiver.address
    );
    const accounts = await web3.eth.getAccounts();

    let oldIndex = Number(
      await initialized.methods.getCurrentGuardianSetIndex().call()
    );

    data = [
      // Core
      core,
      // Action 2 (Guardian Set Upgrade)
      actionGuardianSetUpgrade,
      web3.eth.abi
        .encodeParameter("uint16", testChainId)
        .substring(2 + (64 - 4)),
      web3.eth.abi
        .encodeParameter("uint32", oldIndex + 1)
        .substring(2 + (64 - 8)),
      web3.eth.abi.encodeParameter("uint8", 3).substring(2 + (64 - 2)),
      web3.eth.abi
        .encodeParameter("address", testSigner1.address)
        .substring(2 + (64 - 40)),
      web3.eth.abi
        .encodeParameter("address", testSigner2.address)
        .substring(2 + (64 - 40)),
      web3.eth.abi
        .encodeParameter("address", testSigner3.address)
        .substring(2 + (64 - 40)),
    ].join("");

    const vm = await signAndEncodeVM(
      0,
      0,
      testGovernanceChainId,
      testGovernanceContract,
      0,
      data,
      [testSigner1PK, testSigner2PK, testSigner3PK],
      1,
      2
    );

    await initialized.methods.submitNewGuardianSet("0x" + vm).send({
      value: 0,
      from: accounts[0],
      gasLimit: 1000000,
    });

    // create another vaa because we had to update the guardian set
    const vm2 = await signAndEncodeVM(
      0,
      0,
      testGovernanceChainId,
      testGovernanceContract,
      0,
      data,
      [testSigner1PK, testSigner2PK, testSigner3PK],
      2,
      2
    );

    try {
      await initialized.methods.submitNewGuardianSet("0x" + vm2).send({
        value: 0,
        from: accounts[0],
        gasLimit: 1000000,
      });

      assert.fail("governance packet accepted twice");
    } catch (e) {
      assert.equal(
        e.data[Object.keys(e.data)[0]].reason,
        "governance action already consumed"
      );
    }
  });
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
    web3.eth.abi
      .encodeParameter("uint16", emitterChainId)
      .substring(2 + (64 - 4)),
    web3.eth.abi.encodeParameter("bytes32", emitterAddress).substring(2),
    web3.eth.abi.encodeParameter("uint64", sequence).substring(2 + (64 - 16)),
    web3.eth.abi
      .encodeParameter("uint8", consistencyLevel)
      .substring(2 + (64 - 2)),
    data.substr(2),
  ];

  const hash = web3.utils.soliditySha3(
    web3.utils.soliditySha3("0x" + body.join(""))
  );

  let signatures = "";

  for (let i in signers) {
    const ec = new elliptic.ec("secp256k1");
    const key = ec.keyFromPrivate(signers[i]);
    const signature = key.sign(hash.substr(2), { canonical: true });

    const packSig = [
      web3.eth.abi.encodeParameter("uint8", i).substring(2 + (64 - 2)),
      zeroPadBytes(signature.r.toString(16), 32),
      zeroPadBytes(signature.s.toString(16), 32),
      web3.eth.abi
        .encodeParameter("uint8", signature.recoveryParam)
        .substr(2 + (64 - 2)),
    ];

    signatures += packSig.join("");
  }

  const vm = [
    web3.eth.abi.encodeParameter("uint8", 1).substring(2 + (64 - 2)),
    web3.eth.abi
      .encodeParameter("uint32", guardianSetIndex)
      .substring(2 + (64 - 8)),
    web3.eth.abi
      .encodeParameter("uint8", signers.length)
      .substring(2 + (64 - 2)),

    signatures,
    body.join(""),
  ].join("");

  return vm;
};

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
    web3.eth.abi
      .encodeParameter("uint16", emitterChainId)
      .substring(2 + (64 - 4)),
    web3.eth.abi.encodeParameter("bytes32", emitterAddress).substring(2),
    web3.eth.abi.encodeParameter("uint64", sequence).substring(2 + (64 - 16)),
    web3.eth.abi
      .encodeParameter("uint8", consistencyLevel)
      .substring(2 + (64 - 2)),
    data.substr(2),
  ];

  const hash = web3.utils.soliditySha3(
    web3.utils.soliditySha3("0x" + body.join(""))
  );

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
      web3.eth.abi
        .encodeParameter("uint8", signature.recoveryParam)
        .substr(2 + (64 - 2)),
    ];

    signatures += packSig.join("");
  }

  const vm = [
    web3.eth.abi.encodeParameter("uint8", 1).substring(2 + (64 - 2)),
    web3.eth.abi
      .encodeParameter("uint32", guardianSetIndex)
      .substring(2 + (64 - 8)),
    web3.eth.abi
      .encodeParameter("uint8", signers.length)
      .substring(2 + (64 - 2)),

    signatures,
    body.join(""),
  ].join("");

  return vm;
};

function zeroPadBytes(value, length) {
  while (value.length < 2 * length) {
    value = "0" + value;
  }
  return value;
}
