import assert from "assert"

import * as anchor from "@coral-xyz/anchor"
import { ComputeBudgetProgram, Keypair, PublicKey } from "@solana/web3.js"
import { toUniversal, UniversalAddress } from "@wormhole-foundation/sdk-definitions"
import { encoding, serializeLayout } from "@wormhole-foundation/sdk-base"
import { randomBytes } from "@noble/hashes/utils"

import { VerificationV2 } from "../target/types/verification_v2.js"

import { guardianAddress, TestingWormholeCore } from "./testing-wormhole-core.js"
import { WormholeContracts, TestsHelper, expectFailure } from "./testing_helpers.js"
import { inspect } from "util"
import { appendSchnorrKeyMessageLayout, headerV2Layout } from "./layouts.js"


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

// Signature for 100 zero bytes body VAA with `testSchnorrKey` hashed only once
// ContractSig{
//   pkX                : 0x1cafae803bf91a2e5494162625d34fda2f69db7c1f3589938647bc2abd4a0a0f
//   pkyparity          : 0
//   msghash            : 0x913fb9e1f6f1c6d910fd574a5cad8857aa43bfba24e401ada4f56090d4d997a7
//   s                  : 0x1c2d1ca6fd3830e653d2abfc57956f3700059a661d8cabae684ea1bc62294e4c
//   nonceTimesGAddress : 0xe46df5bea4597cef7d3c6eff36356a3f0ba33a56
// }

// const testSchnorrKey = encoding.bignum.toBytes(
//   0x1cafae803bf91a2e5494162625d34fda2f69db7c1f3589938647bc2abd4a0a0fn << 1n, 32
// );

// const signatureTestMessage100Zeroed = {
//   r: encoding.hex.decode("0xE46Df5BEa4597CEF7D3c6EfF36356A3F0bA33a56"),
//   s: encoding.hex.decode("0x1c2d1ca6fd3830e653d2abfc57956f3700059a661d8cabae684ea1bc62294e4c"),
// }


// Signature for 100 zero bytes body VAA with `testSchnorrKey` hashed twice
// ContractSig{
//   pkX                : 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c
//   pkyparity          : 0
//   msghash            : 0x258752639c534fd7fb6b52e5e3ba32ed9e8de081c966fd895992f63464869309
//   s                  : 0xaa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29
//   nonceTimesGAddress : 0x636a8688ef4b82e5a121f7c74d821a5b07d695f3
// }

const testSchnorrKey = encoding.bignum.toBytes(
  0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770cn << 1n, 32
);

const signatureTestMessage100Zeroed = {
  r: encoding.hex.decode("0x636a8688ef4b82e5a121f7c74d821a5b07d695f3"),
  s: encoding.hex.decode("0xaa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29"),
}

const invalidSignature = {
  r: encoding.hex.decode("0xE46Df5BEa4597CEF7D346EfF36356A3F0bA33a56"),
  s: encoding.hex.decode("0x1c2d1ca6fd3830e653d6abfc57956f3700059a661d8cabae684ea1bc62294e4c"),
}


const getTestMessage100Zeroed = (schnorrKeyIndex: number) => Uint8Array.from([
  ...serializeLayout(headerV2Layout, {
    schnorrKeyIndex: schnorrKeyIndex,
    signature: signatureTestMessage100Zeroed,
  }),
  ...new Uint8Array(100)
])

const getTestMessageInvalidSignature = (schnorrKeyIndex: number) => Uint8Array.from([
  ...serializeLayout(headerV2Layout, {
    schnorrKeyIndex: schnorrKeyIndex,
    signature: invalidSignature,
  }),
  ...new Uint8Array(100)
])


// ------------------------------------------------------------------------------------------------


describe("VerificationV2", function() {
  const coreV1Address = new PublicKey('worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth')
  const guardianSetExpirationTime = 86400
  const fee = 100
  const txSigner = $.keypair.generate()
  let coreV1: TestingWormholeCore<"Devnet">;
  const connection = $.connection
  let payer: Keypair;
  const coreV2 = anchor.workspace.VerificationV2 as anchor.Program<VerificationV2>
  const testKeyIndex = 5


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

  function postVaaV1(message: Uint8Array) {
    const governanceContract = new UniversalAddress("0000000000000000000000000000000000000000000000000000000000000004", "hex");
    return coreV1.postVaa(
      payer,
      {
        chain: "Solana",
        emitterAddress: governanceContract,
      },
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
        ix = await coreV2.methods.initSchnorrKey().accounts({
          vaa: postedVaaAddress,
          newSchnorrKey: deriveSchnorrKeyPda(test.keyIndex)[0]
        }).instruction()
      } else {
        ix = await coreV2.methods.appendSchnorrKey().accounts({
          vaa: postedVaaAddress,
          newSchnorrKey: deriveSchnorrKeyPda(test.keyIndex)[0],
          oldSchnorrKey: deriveSchnorrKeyPda(test.oldKeyIndex)[0],
          latestKey: deriveLatestKeyPda()[0],
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

    coreV1 = new TestingWormholeCore(
      txSigner,
      connection,
      WormholeContracts.Network,
      coreV1Address,
      WormholeContracts.addresses,
    );

    const txid = await coreV1.initialize(undefined, guardianSetExpirationTime, fee)
    const tx = await $.getTransaction(txid)
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
      name: "Appends Schnorr key leaving a gap successfully",
      test: {
        operation: "AppendSchnorrKey",
        keyIndex: testKeyIndex,
        publicKey: testSchnorrKey,
        previousSetExpirationTime: guardianSetExpirationTime,
        oldKeyIndex: 1,
      }
    },
  ] satisfies AddKeyTest[]).map((test) => addKeyTest(test))

  it("Verifies a v2 VAA", async function() {
    const vaa = Buffer.from(getTestMessage100Zeroed(testKeyIndex));
    const verifyIx = await coreV2.methods.verifyVaa(vaa).accounts({
      schnorrKey: deriveSchnorrKeyPda(testKeyIndex)[0],
    }).instruction()

    const txid = await $.sendAndConfirm(verifyIx, payer)
    const tx = await $.getTransaction(txid);
    console.log(`logs: ${tx?.meta?.logMessages?.join("\n")}`)
    console.log(`compute units consumed: ${tx?.meta?.computeUnitsConsumed}`)
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
});


function generateMockPubkey() {
  const halfQ = 0x7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a1n;
  let key = randomBytes(32)

  const parity = BigInt((key[0] & 0x80) >> 7)
  const x = encoding.bignum.decode(key) % halfQ
  key = encoding.bignum.toBytes((x << 1n) | parity, 32)

  return key
}

function expectInvalidPayload(error: Error) {
  expectAtLeastOneLog(error, "Error Code: InvalidPayload")
}

function expectFailedSignatureVerification(error: Error) {
  expectAtLeastOneLog(error, "Error Code: SignatureVerificationFailed.")
}

function expectAtLeastOneLog(error: Error, message: string) {
  assert((error as any).transactionLogs.find(
    (log: string) => log.includes(message)
  ))
}