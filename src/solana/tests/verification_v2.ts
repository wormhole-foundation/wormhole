import assert from "assert"

import * as anchor from "@coral-xyz/anchor"
import { ComputeBudgetProgram, Keypair, PublicKey } from "@solana/web3.js"
import { keccak256, toUniversal, UniversalAddress } from "@wormhole-foundation/sdk-definitions"
import { Chain, encoding, serializeLayout } from "@wormhole-foundation/sdk-base"
import { randomBytes } from "@noble/hashes/utils"

import { VerificationV2 } from "../target/types/verification_v2.js"

import { guardianAddress, TestingWormholeCore } from "./testing-wormhole-core.js"
import { WormholeContracts, TestsHelper, expectFailure } from "./testing_helpers.js"
import { inspect } from "util"
import { appendSchnorrKeyMessageLayout, HeaderV2, headerV2Layout } from "./layouts.js"


interface SchnorrKeyMessage {
  keyIndex: number;
  publicKey: Uint8Array;
  previousSetExpirationTime: number;
}

interface InitSchnorrKey extends SchnorrKeyMessage {
  operation: "InitSchnorrKey";
}

interface AppendSchnorrKey extends SchnorrKeyMessage {
  operation: "AppendSchnorrKey";
  oldKeyIndex: number;
}

interface AddKeyTest {
  name: string;
  test: InitSchnorrKey | AppendSchnorrKey;
  extraMessageData?: string | Uint8Array;
  expectFailureHandler?: (error: Error) => void | Promise<void>;
}

const $ = new TestsHelper()

export const createAppendSchnorrKeyMessage = ({
  keyIndex,
  publicKey,
  previousSetExpirationTime,
}: SchnorrKeyMessage) => serializeLayout(appendSchnorrKeyMessageLayout, {
  schnorrKeyIndex: keyIndex,
  schnorrKey: publicKey,
  expirationDelaySeconds: previousSetExpirationTime,
  shardDataHash: randomBytes(32),
})

const testSchnorrKey = encoding.bignum.toBytes(
  0xc11b6c8b8e4ecc62ebf10437678eb70f17f1e53abdb3fa8df1912e3b3d11b5b9n, 32
);

const signatureTestMessage100Zeroed = {
  r: encoding.hex.decode("0x41CF8d30EBCc800b655eAD15cC96014d36c4246B"),
  s: encoding.hex.decode("0xfb5fa64887c4a05818b02afa7483e5115f19a93739c4b9ce4e92bae191a2ef4b"),
}

const invalidSignature = {
  r: encoding.hex.decode("0xE46Df5BEa4597CEF7D346EfF36356A3F0bA33a56"),
  s: encoding.hex.decode("0x1c2d1ca6fd3830e653d6abfc57956f3700059a661d8cabae684ea1bc62294e4c"),
}

const getDeserializedHeaderTestMessage100Zeroed = (schnorrKeyIndex: number): HeaderV2 => ({
  schnorrKeyIndex: schnorrKeyIndex,
  signature: signatureTestMessage100Zeroed,
})

const getDeserializedHeaderTestMessageInvalidSignature = (schnorrKeyIndex: number): HeaderV2 => ({
  schnorrKeyIndex: schnorrKeyIndex,
  signature: invalidSignature,
})

const getHeaderTestMessage100Zeroed = (schnorrKeyIndex: number): Uint8Array =>
  serializeLayout(headerV2Layout, getDeserializedHeaderTestMessage100Zeroed(schnorrKeyIndex))

const getHeaderTestMessageInvalidSignature = (schnorrKeyIndex: number): Uint8Array =>
  serializeLayout(headerV2Layout, getDeserializedHeaderTestMessageInvalidSignature(schnorrKeyIndex))

const getTestMessage100Zeroed = (schnorrKeyIndex: number) => Uint8Array.from([
  ...getHeaderTestMessage100Zeroed(schnorrKeyIndex),
  ...new Uint8Array(100)
])

const getTestMessageInvalidSignature = (schnorrKeyIndex: number) => Uint8Array.from([
  ...getHeaderTestMessageInvalidSignature(schnorrKeyIndex),
  ...new Uint8Array(100)
])

const vaaDigest = (vaaBody: Uint8Array) => keccak256(keccak256(vaaBody));


// ------------------------------------------------------------------------------------------------


describe("VerificationV2", function() {
  const coreV1Address = new PublicKey('worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth')
  const guardianSetExpirationTime = 86400
  const fee = 100
  const txSigner = $.keypair.generate()
  let coreV1: TestingWormholeCore<"Devnet">
  const connection = $.connection
  let payer: Keypair;
  const coreV2 = anchor.workspace.VerificationV2 as anchor.Program<VerificationV2>
  const testKeyIndex = 2

  let fakeCoreV1: TestingWormholeCore<"Devnet">


  function deriveSchnorrKeyPda(schnorrKeyIndex: number) {
    // Buffer write already checks that the value is within bounds
    if (!Number.isSafeInteger(schnorrKeyIndex)) {
      throw new Error(`invalid non integer Schnorr index ${schnorrKeyIndex}`)
    }

    const schnorrKeyIndexBuf = Buffer.alloc(4)
    schnorrKeyIndexBuf.writeUint32LE(schnorrKeyIndex)

    // See impl SeedPrefix for SchnorrKeyAccount
    const seeds = [Buffer.from("schnorrkey"), schnorrKeyIndexBuf]

    return PublicKey.findProgramAddressSync(
      seeds,
      coreV2.programId
    )
  }

  function deriveLatestKeyPda() {
    // See impl SeedPrefix for LatestKeyAccount
    const seeds = [Buffer.from("latestkey")]

    return PublicKey.findProgramAddressSync(
      seeds,
      coreV2.programId
    )
  }

  function postVaaV1(
    message: Uint8Array,
    core = coreV1,
    emitter = {
      chain: "Solana" as Chain,
      emitterAddress: new UniversalAddress("0000000000000000000000000000000000000000000000000000000000000004", "hex"),
    }
  ) {
    return core.postVaa(
      payer,
      emitter,
      message,
    )
  }

  function addKeyTest({
    name,
    test,
    extraMessageData,
    expectFailureHandler,
  }: AddKeyTest) {
    it(name, async () => {
      let message = createAppendSchnorrKeyMessage(test)
      if (extraMessageData !== undefined) {
        message = Uint8Array.from([...message, ...extraMessageData])
      }

      const postedVaaAddress = await postVaaV1(message)

      let ix
      if (test.operation === "InitSchnorrKey") {
        ix = await coreV2.methods.appendSchnorrKey().accountsPartial({
          vaa: postedVaaAddress,
          newSchnorrKey: deriveSchnorrKeyPda(test.keyIndex)[0],
          oldSchnorrKey: null,
        }).instruction()
      } else {
        ix = await coreV2.methods.appendSchnorrKey().accountsPartial({
          vaa: postedVaaAddress,
          newSchnorrKey: deriveSchnorrKeyPda(test.keyIndex)[0],
          oldSchnorrKey: deriveSchnorrKeyPda(test.oldKeyIndex)[0],
        }).instruction()
      }

      if (expectFailureHandler !== undefined) {
        return expectFailure(
          () => $.sendAndConfirm(ix, payer),
          expectFailureHandler
        )
      } else {
        return $.sendAndConfirm(ix, payer)
      }
    })
  }

  before(async function() {
    payer = anchor.getProvider().wallet?.payer!
    assert(payer, "Payer not found")

    await $.airdrop([
      txSigner.publicKey,
      payer.publicKey,
    ]);

    const wormholeContracts = new WormholeContracts();

    coreV1 = new TestingWormholeCore(
      txSigner,
      connection,
      wormholeContracts.network,
      coreV1Address,
      wormholeContracts.addresses,
    );

    let txid = await coreV1.initialize(undefined, guardianSetExpirationTime, fee)
    let tx = await $.getTransaction(txid)

    const fakeWormholeContracts = new WormholeContracts("fake-wormhole-core-v1");

    fakeCoreV1 = new TestingWormholeCore(
      txSigner,
      connection,
      fakeWormholeContracts.network,
      coreV1Address,
      fakeWormholeContracts.addresses,
    );

    txid = await fakeCoreV1.initialize(undefined, guardianSetExpirationTime, fee)
    tx = await $.getTransaction(txid)
  });

  it("Check correct core v1 setup", async function() {
    const accounts = await connection.getProgramAccounts(coreV1Address)
    assert(accounts.length === 2, "Expected 2 accounts")

    const guardianSetIndex = await coreV1.client.getGuardianSetIndex()
    assert(guardianSetIndex === 0, "Expected guardian set index to be 0")
    const guardianSet = await coreV1.client.getGuardianSet(guardianSetIndex);

    const queriedFee = await coreV1.client.getMessageFee();
    assert(queriedFee === BigInt(fee), "Expected fee to be 100")

    assert(guardianSet.index === 0, "Expected guardian set index to be 0")
    assert(guardianSet.keys.length === 1, "Expected guardian set keys to have length 1")

    const queriedGuardian = new UniversalAddress(guardianSet.keys[0], "hex")
    const expectedGuardian = toUniversal("Ethereum", guardianAddress)
    assert(queriedGuardian.equals(expectedGuardian), "Expected guardian set keys to be the devnet guardian")
  });

  ([
    {
      name: "Posts invalid init append Schnorr key VAA and fails",
      test: {
        operation: "InitSchnorrKey",
        keyIndex: 0,
        publicKey: generateMockPubkey(),
        previousSetExpirationTime: guardianSetExpirationTime,
      },
      extraMessageData: "junkdata",
      expectFailureHandler: expectInvalidPayload,
    },
    {
      name: "Posts init append Schnorr key VAA successfully",
      test: {
        operation: "InitSchnorrKey",
        keyIndex: 0,
        publicKey: generateMockPubkey(),
        previousSetExpirationTime: guardianSetExpirationTime,
      },
    },
    {
      name: "Posts invalid append Schnorr key VAA and fails",
      test: {
        operation: "AppendSchnorrKey",
        keyIndex: 1,
        publicKey: generateMockPubkey(),
        previousSetExpirationTime: guardianSetExpirationTime,
        oldKeyIndex: 0,
      },
      extraMessageData: "junkdata",
      expectFailureHandler: expectInvalidPayload,
    },
    {
      name: "Posts append Schnorr key VAA successfully",
      test: {
        operation: "AppendSchnorrKey",
        keyIndex: 1,
        publicKey: generateMockPubkey(),
        previousSetExpirationTime: guardianSetExpirationTime,
        oldKeyIndex: 0,
      }
    },
    {
      name: "Fails to append Schnorr key when skipping indices",
      test: {
        operation: "AppendSchnorrKey",
        keyIndex: 5,
        publicKey: testSchnorrKey,
        previousSetExpirationTime: guardianSetExpirationTime,
        oldKeyIndex: 1,
      },
      expectFailureHandler: expectNewKeyIndexNotDirectSuccessor,
    },
    {
      name: "Appends a third Schnorr key",
      test: {
        operation: "AppendSchnorrKey",
        keyIndex: testKeyIndex,
        publicKey: testSchnorrKey,
        previousSetExpirationTime: guardianSetExpirationTime,
        oldKeyIndex: 1,
      }
    },
    {
      name: "Fails to append Schnorr key when referencing an old key",
      test: {
        operation: "AppendSchnorrKey",
        keyIndex: testKeyIndex,
        publicKey: testSchnorrKey,
        previousSetExpirationTime: guardianSetExpirationTime,
        oldKeyIndex: testKeyIndex - 1,
      },
      expectFailureHandler: expectAllocateAccountError(deriveSchnorrKeyPda(testKeyIndex)[0].toBase58()),
    },
    {
      name: "Fails to append invalid Schnorr key",
      test: {
        operation: "AppendSchnorrKey",
        keyIndex: testKeyIndex + 1,
        publicKey: generateInvalidMockPubkey(),
        previousSetExpirationTime: guardianSetExpirationTime,
        oldKeyIndex: testKeyIndex,
      },
      expectFailureHandler: expectInvalidSchnorrKey,
    },
  ] satisfies AddKeyTest[]).map((test) => addKeyTest(test));

  [{
    name: "Fails to append Schnorr key when emitter chain is not Solana",
    emitter: {
      chain: "Ethereum",
      emitterAddress: new UniversalAddress("0x0000000000000000000000000000000000000000000000000000000000000004"),
    } as const,
    test: {
      keyIndex: testKeyIndex + 1,
      publicKey: testSchnorrKey,
      previousSetExpirationTime: guardianSetExpirationTime,
      oldKeyIndex: testKeyIndex,
    },
    expectFailureHandler: expectInvalidGovernanceChain,
  },{
    name: "Fails to append Schnorr key when emitter address is not governance contract",
    emitter: {
      chain: "Solana",
      emitterAddress: new UniversalAddress("0x0000000000000000000000000000000000000000000000000000000000000009"),
    } as const,
    test: {
      keyIndex: testKeyIndex + 1,
      publicKey: testSchnorrKey,
      previousSetExpirationTime: guardianSetExpirationTime,
      oldKeyIndex: testKeyIndex,
    },
    expectFailureHandler: expectInvalidGovernanceContract,
  }].map(({name, emitter, test, expectFailureHandler}) => it(name, async () => {
    let message = createAppendSchnorrKeyMessage(test)

    const postedVaaAddress = await postVaaV1(message, undefined, emitter)

    let ix = await coreV2.methods.appendSchnorrKey().accountsPartial({
      vaa: postedVaaAddress,
      newSchnorrKey: deriveSchnorrKeyPda(test.keyIndex)[0],
      oldSchnorrKey: deriveSchnorrKeyPda(test.oldKeyIndex)[0],
    }).instruction()

    return expectFailure(
      () => $.sendAndConfirm(ix, payer),
      expectFailureHandler
    )
  }))


  it("Posting a governance VAA to a fake wormhole contract is not accepted by VerificationV2", async () => {
    const newKeyIndex = 10
    const message = createAppendSchnorrKeyMessage({
      keyIndex: newKeyIndex,
      publicKey: testSchnorrKey,
      previousSetExpirationTime: guardianSetExpirationTime,
    })

    const postedVaaAddress = await postVaaV1(message, fakeCoreV1)

    const ix = await coreV2.methods.appendSchnorrKey().accountsPartial({
      vaa: postedVaaAddress,
      newSchnorrKey: deriveSchnorrKeyPda(newKeyIndex)[0],
      oldSchnorrKey: deriveSchnorrKeyPda(testKeyIndex)[0],
    }).instruction()

    return expectFailure(
      () => $.sendAndConfirm(ix, payer),
      (error) => expectAtLeastOneLog(error, "Error Code: AccountOwnedByWrongProgram."),
    )
  })

  it("Verifies a v2 VAA", async function() {
    const vaa = Buffer.from(getTestMessage100Zeroed(testKeyIndex));
    const verifyIx = await coreV2.methods.verifyVaa(vaa).accounts({
      schnorrKey: deriveSchnorrKeyPda(testKeyIndex)[0],
    }).instruction()

    const txid = await $.sendAndConfirm(verifyIx, payer)
    const tx = await $.getTransaction(txid);
    console.log(`logs: ${tx?.meta?.logMessages?.join("\n")}`)
    console.log(`${this.test?.title}: CUs consumed: ${tx?.meta?.computeUnitsConsumed}`)
  })

  it("v2 VAA verification fails for an invalid signature", async function() {
    const vaa = Buffer.from(getTestMessageInvalidSignature(testKeyIndex));
    const verifyIx = await coreV2.methods.verifyVaa(vaa).accounts({
      schnorrKey: deriveSchnorrKeyPda(testKeyIndex)[0],
    }).instruction()

    expectFailure(
      () => $.sendAndConfirm(verifyIx, payer),
      expectFailedSignatureVerification
    )
  })

  it("Verifies a v2 VAA and decodes", async function() {
    const vaa = Buffer.from(getTestMessage100Zeroed(testKeyIndex));
    const verifyIx = await coreV2.methods.verifyVaaAndDecode(vaa).accounts({
      schnorrKey: deriveSchnorrKeyPda(testKeyIndex)[0],
    }).instruction()

    const txid = await $.sendAndConfirm(verifyIx, payer)
    const tx = await $.getTransaction(txid);
    // console.log(`logs: ${tx?.meta?.logMessages?.join("\n")}`)
    console.log(`${this.test?.title}: CUs consumed: ${tx?.meta?.computeUnitsConsumed}`)
  })

  it("v2 VAA verification and decoding fails for an invalid signature", async function() {
    const vaa = Buffer.from(getTestMessageInvalidSignature(testKeyIndex));
    const verifyIx = await coreV2.methods.verifyVaaAndDecode(vaa).accounts({
      schnorrKey: deriveSchnorrKeyPda(testKeyIndex)[0],
    }).instruction()

    expectFailure(
      () => $.sendAndConfirm(verifyIx, payer),
      expectFailedSignatureVerification
    )
  })

  it("Verifies a v2 VAA header with digest", async function() {
    const vaaHeader = [...getHeaderTestMessage100Zeroed(testKeyIndex)]
    const digest = [...vaaDigest(new Uint8Array(100))]
    const verifyIx = await coreV2.methods.verifyVaaHeaderWithDigest(vaaHeader, digest).accounts({
      schnorrKey: deriveSchnorrKeyPda(testKeyIndex)[0],
    }).instruction()

    const txid = await $.sendAndConfirm(verifyIx, payer)
    const tx = await $.getTransaction(txid);
    // console.log(`logs: ${tx?.meta?.logMessages?.join("\n")}`)
    console.log(`${this.test?.title}: CUs consumed: ${tx?.meta?.computeUnitsConsumed}`)
  })

  it("v2 VAA header and digest verification fails for an invalid signature", async function() {
    const vaaHeader = [...getHeaderTestMessageInvalidSignature(testKeyIndex)]
    const digest = [...vaaDigest(new Uint8Array(100))]
    const verifyIx = await coreV2.methods.verifyVaaHeaderWithDigest(vaaHeader, digest).accounts({
      schnorrKey: deriveSchnorrKeyPda(testKeyIndex)[0],
    }).instruction()

    expectFailure(
      () => $.sendAndConfirm(verifyIx, payer),
      expectFailedSignatureVerification
    )
  })
});


function generateMockPubkey() {
  const halfQ = 0x7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a1n;
  let key = randomBytes(32)

  const parity = BigInt((key[0] & 0x80) >> 7)
  const x = encoding.bignum.decode(key) % halfQ
  key = encoding.bignum.toBytes((x << 1n) | parity, 32)

  return key
}

function generateInvalidMockPubkey() {
  const key = new Uint8Array(32)
  key.fill(0xff)

  return key
}

function expectInvalidPayload(error: Error) {
  expectAtLeastOneLog(error, "Error Message: IO Error: Invalid payload.")
}

function expectFailedSignatureVerification(error: Error) {
  expectAtLeastOneLog(error, "Error Code: SignatureVerificationFailed.")
}

function expectInvalidOldSchnorrKey(error: Error) {
  expectAtLeastOneLog(error, "Error Code: InvalidOldSchnorrKey.")
}

function expectInvalidGovernanceChain(error: Error) {
  expectAtLeastOneLog(error, "Error Code: InvalidGovernanceChainId.")
}

function expectInvalidGovernanceContract(error: Error) {
  expectAtLeastOneLog(error, "Error Code: InvalidGovernanceAddress.")
}

function expectInvalidSchnorrKey(error: Error) {
  expectAtLeastOneLog(error, "Error Code: AccountDidNotSerialize.")
}

function expectNewKeyIndexNotDirectSuccessor(error: Error) {
  expectAtLeastOneLog(error, "Error Code: NewKeyIndexNotDirectSuccessor.")
}

function expectAllocateAccountError(account: string) {
  return (error: Error) => expectAtLeastOneLog(error, `Allocate: account Address { address: ${account}, base: None } already in use`)
}


function expectAtLeastOneLog(error: Error, message: string) {
  assert((error as any).transactionLogs.find(
    (log: string) => log.includes(message)
  ))
}