/**
 * Generate a PULL_AND_APPEND_MESSAGE for the 19-guardian test setup.
 * 
 * Usage: npx tsx generate-test-message.ts
 * 
 * This script generates the update message needed for localAnvilWithVerifier.sh
 */

import { ethers } from 'ethers';

// Constants from WormholeVerifier.sol
const CHAIN_ID_SOLANA = 1;
const GOVERNANCE_ADDRESS = '0x0000000000000000000000000000000000000000000000000000000000000004';
const MODULE_VERIFICATION_V2 = '0x0000000000000000000000000000000000000000000000000000000000545353';
const ACTION_APPEND_SCHNORR_KEY = 0x01;

const UPDATE_PULL_MULTISIG_KEY_DATA = 0x02;
const UPDATE_APPEND_SCHNORR_KEY = 0x01;

// 19 Guardian Private Keys (from client.sh)
const GUARDIAN_PRIVATE_KEYS: string[] = [
  '0xc67c82a42364074e4a3ffec944a96d758cec08da09275ada471fe82c95159af9',
  '0x22b517009ccd7ede90cebf16ab630868989172c72ebd70ffc6d04cb783eeba61',
  '0x82272a56366878aa05e5cbcd5cd1d195d14ca26064be20cdb7aacebcf2c86a20',
  '0x6e24f50d0c1bae02d47e0ef5ae26cc5e3ee7d98660b508736fa4a7d64fe3c625',
  '0xc81a9b27aeef5a963959d5e424a0686a981918a009cacef18ded6e6496c9a808',
  '0x439bd6108f1590b56ebdea0706c2dc308240c4a47d94f42d16f92db81f6222b9',
  '0xa2218874818e7da07f632a9c9bbfcd96ed478b6a4a6a9829bff2eb5f8ed719a8',
  '0xd73ba7c666d41f554e8beaf96d5e2ae6590aefc7b5ef7008dc0b49b6328966a0',
  '0x7b015af9ed3b47ff5a1a71410bbda0aaf40111858925784658f19acf4c085f2e',
  '0xfe156e000de4ae76fd51fb41331bbab47a10a6f0d8ea96859cd5f66e2ec9b6a9',
  '0xebbe5853f748ac603c984ed162ee44d2e898ca4552f3b153f59a665681ab2c00',
  '0x155f68801d5a51f216b26a3e0f5fb41a1f0525a61cddb6b0ab2b4c19213ebc55',
  '0x2c45b839f24d1cb4ea25c0079e62f930f91499c77d13b43074b3c3810718cd8d',
  '0x61bad7e65e328ae170db4326021c7585baadae785350e12f1f2bdb7e73c10b28',
  '0x9f8efce656540a4ddd16a323d7a3e2e9355dc13c7afa1f6197ea72cfc1375517',
  '0xef60c2cd0f8ae627298a1dd237670198aa1dabb3cbcf2a532fd1cadc698788e3',
  '0x83b81eb33a7fcb0694e212de9e4fb0db2579e8c4941a43adae1ebd8f43559aa9',
  '0x24b2e1ac892f26e3741cd99b29ceb4c9f6acadb1af74942e83f23f8d37dcb6d6',
  '0x759540959b56070ee6effb4793af059d49897d9b54edaf78f0f06f5b03ec6c2c',
];

// Test Schnorr public key (from test file)
const SCHNORR_PUBLIC_KEY = '0xc11b6c8b8e4ecc62ebf10437678eb70f17f1e53abdb3fa8df1912e3b3d11b5b9';

function generateDeterministicShardData(guardianCount: number): { raw: Uint8Array, hash: string } {
  const parts: Uint8Array[] = [];

  for (let i = 0; i < guardianCount; i++) {
    // Generate deterministic shard data based on guardian index
    // In production this would come from DKG, but for testing we use deterministic values
    const shard = ethers.keccak256(ethers.solidityPacked(['string', 'uint256'], ['shard', i]));
    const pubKeyX = ethers.keccak256(ethers.solidityPacked(['string', 'uint256'], ['pubKeyX', i]));
    const pubKeyY = ethers.keccak256(ethers.solidityPacked(['string', 'uint256'], ['pubKeyY', i]));

    parts.push(ethers.getBytes(shard));
    parts.push(ethers.getBytes(pubKeyX));
    parts.push(ethers.getBytes(pubKeyY));
  }

  // Concatenate all parts
  const raw = ethers.concat(parts);
  const hash = ethers.keccak256(raw);

  return { raw: ethers.getBytes(raw), hash };
}

function createAppendSchnorrKeyPayload(
  schnorrKeyIndex: number,
  expectedMultisigKeyIndex: number,
  schnorrPubkey: string,
  expirationDelaySeconds: number,
  shardDataHash: string
): string {
  return ethers.solidityPacked(
    ['bytes32', 'uint8', 'uint32', 'uint32', 'bytes32', 'uint32', 'bytes32'],
    [
      MODULE_VERIFICATION_V2,
      ACTION_APPEND_SCHNORR_KEY,
      schnorrKeyIndex,
      expectedMultisigKeyIndex,
      schnorrPubkey,
      expirationDelaySeconds,
      shardDataHash,
    ]
  );
}

function createVaaEnvelope(
  timestamp: number,
  nonce: number,
  emitterChainId: number,
  emitterAddress: string,
  sequence: bigint,
  consistencyLevel: number,
  payload: string
): string {
  return ethers.solidityPacked(
    ['uint32', 'uint32', 'uint16', 'bytes32', 'uint64', 'uint8', 'bytes'],
    [timestamp, nonce, emitterChainId, emitterAddress, sequence, consistencyLevel, payload]
  );
}

function getEnvelopeDigest(envelope: string): string {
  return ethers.keccak256(ethers.keccak256(envelope));
}

async function signMultisig(digest: string, privateKeys: string[]): Promise<string> {
  const signatures: Uint8Array[] = [];

  for (let i = 0; i < privateKeys.length; i++) {
    const wallet = new ethers.Wallet(privateKeys[i]);
    const signature = wallet.signingKey.sign(digest);
    
    // Format: guardianIndex (1 byte) + r (32 bytes) + s (32 bytes) + v (1 byte, normalized to 0/1)
    const sigBytes = new Uint8Array(66);
    sigBytes[0] = i; // guardian index
    
    const rBytes = ethers.getBytes(signature.r);
    const sBytes = ethers.getBytes(signature.s);
    
    sigBytes.set(rBytes, 1);
    sigBytes.set(sBytes, 33);
    sigBytes[65] = signature.v === 28 ? 1 : 0; // v normalized
    
    signatures.push(sigBytes);
  }

  return ethers.hexlify(ethers.concat(signatures));
}

function createMultisigVaa(keyIndex: number, signatures: string, envelope: string): string {
  const signatureBytes = ethers.getBytes(signatures);
  const signatureCount = signatureBytes.length / 66;

  return ethers.solidityPacked(
    ['uint8', 'uint32', 'uint8', 'bytes', 'bytes'],
    [1, keyIndex, signatureCount, signatures, envelope]
  );
}

async function main() {
  console.log('Generating PULL_AND_APPEND_MESSAGE for 19-guardian test setup...\n');

  const guardianCount = GUARDIAN_PRIVATE_KEYS.length;
  console.log(`Guardian count: ${guardianCount}`);

  // Step 1: Generate deterministic shard data
  console.log('\n1. Generating shard data for guardians...');
  const { raw: shardDataRaw, hash: shardDataHash } = generateDeterministicShardData(guardianCount);
  console.log(`   Shard data length: ${shardDataRaw.length} bytes (expected: ${guardianCount * 96})`);
  console.log(`   Shard data hash: ${shardDataHash}`);

  // Step 2: Create the governance payload
  console.log('\n2. Creating governance payload...');
  const payload = createAppendSchnorrKeyPayload(
    0,                    // schnorrKeyIndex (first key)
    0,                    // expectedMultisigKeyIndex
    SCHNORR_PUBLIC_KEY,
    0,                    // expirationDelaySeconds (0 for first key)
    shardDataHash
  );
  console.log(`   Payload length: ${ethers.getBytes(payload).length} bytes`);

  // Step 3: Create the VAA envelope
  console.log('\n3. Creating VAA envelope...');
  const timestamp = 0; // Will be overwritten in practice
  const envelope = createVaaEnvelope(
    timestamp,
    0,                    // nonce
    CHAIN_ID_SOLANA,
    GOVERNANCE_ADDRESS,
    0n,                   // sequence
    0,                    // consistencyLevel
    payload
  );
  console.log(`   Envelope length: ${ethers.getBytes(envelope).length} bytes`);

  // Step 4: Sign with all guardians
  console.log('\n4. Signing with all guardians...');
  const digest = getEnvelopeDigest(envelope);
  console.log(`   Digest: ${digest}`);
  const signatures = await signMultisig(digest, GUARDIAN_PRIVATE_KEYS);
  console.log(`   Signatures length: ${ethers.getBytes(signatures).length} bytes`);

  // Step 5: Create the multisig VAA
  console.log('\n5. Creating multisig VAA...');
  const multisigVaa = createMultisigVaa(0, signatures, envelope);
  const multisigVaaBytes = ethers.getBytes(multisigVaa);
  console.log(`   VAA length: ${multisigVaaBytes.length} bytes`);

  // Step 6: Assemble the full update message
  console.log('\n6. Assembling full update message...');
  
  // PULL_MULTISIG_KEY_DATA with limit 1
  const pullMessage = ethers.solidityPacked(
    ['uint8', 'uint32'],
    [UPDATE_PULL_MULTISIG_KEY_DATA, 1]
  );

  // APPEND_SCHNORR_KEY with VAA + shard data
  const appendDataLength = multisigVaaBytes.length + shardDataRaw.length;
  const appendMessage = ethers.solidityPacked(
    ['uint8', 'uint16', 'bytes', 'bytes'],
    [UPDATE_APPEND_SCHNORR_KEY, appendDataLength, multisigVaa, ethers.hexlify(shardDataRaw)]
  );

  // Combine
  const fullMessage = ethers.concat([pullMessage, appendMessage]);
  
  console.log(`   Pull message length: ${ethers.getBytes(pullMessage).length} bytes`);
  console.log(`   Append message length: ${ethers.getBytes(appendMessage).length} bytes`);
  console.log(`   Total message length: ${ethers.getBytes(fullMessage).length} bytes`);

  console.log('\n========================================');
  console.log('PULL_AND_APPEND_MESSAGE=');
  console.log(ethers.hexlify(fullMessage));
  console.log('========================================\n');

  // Also output the guardian addresses for verification
  console.log('Guardian addresses (for GUARDIAN_ADDRESSES array):');
  for (const pk of GUARDIAN_PRIVATE_KEYS) {
    const wallet = new ethers.Wallet(pk);
    console.log(`  "${wallet.address}"`);
  }
}

main().catch(console.error);
