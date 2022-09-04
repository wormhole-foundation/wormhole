import { decodeAddress, encodeUint64 } from 'algosdk'
import elliptic from 'elliptic'
import assert from 'assert'
import web3Utils from 'web3-utils'
import { zeroPad } from 'ethers/lib/utils'
import { concatArrays, encodeUint32, encodeUint16, encodeUint8, decodeBase16 } from '../sdk/Encoding'

import { GovenanceMessageType, GovernancePayload, RegisterChainPayload, SignedVAA, UnsignedVAA, VAAPayload, VAAPayloadType } from './WormholeTypes'

export const EMITTER_GOVERNANCE = decodeBase16('0000000000000000000000000000000000000000000000000000000000000004')

function encodeGovernancePayload(payload: GovernancePayload): Uint8Array {
	let body: Uint8Array
	switch (payload.type) {
		case GovenanceMessageType.SetUpdateHash: {
			body = payload.updateHash
			break
		}
		case GovenanceMessageType.UpdateGuardians: {
			body = concatArrays([
				encodeUint32(payload.newGSIndex),
				encodeUint8(payload.guardians.length),
				...payload.guardians
			])

			break
		}
		case GovenanceMessageType.SetMessageFee: {
			body = concatArrays([
				zeroPad([], 24),
				encodeUint64(payload.messageFee),
			])
			break
		}
		case GovenanceMessageType.SendAlgo: {
			body = concatArrays([
				zeroPad([], 24),
				encodeUint64(payload.fee),
				decodeAddress(payload.dest).publicKey,
			])
			break
		}
		default: {
			throw new Error(`Unknown governance message type`)
		}
	}

	const result = concatArrays([
		decodeBase16('00000000000000000000000000000000000000000000000000000000436f7265'),
		encodeUint8(payload.type),
		encodeUint16(payload.targetChainId),
		body,
	])

	return result
}

function encodeRegisterChainPayload(payload: RegisterChainPayload): Uint8Array {
	return concatArrays([
		decodeBase16('0000000000000000000000000000000000000000000000000000000000000000'),
		decodeBase16('000000000000000000000000000000000000000000546f6b656e427269646765'), // "TokenBridge"
		encodeUint8(1), // FIXME: Is this a version number? We should make it variable so we can test the contract
		encodeUint16(payload.targetChainId),
		encodeUint16(payload.emitterChainId),
		payload.emitterAddress,
	])
}

function encodePayload(payload: VAAPayload): Uint8Array {
	switch (payload.type) {
		case VAAPayloadType.Raw:
			return payload.payload

		case VAAPayloadType.Governance:
			return encodeGovernancePayload(payload.payload)

		case VAAPayloadType.RegisterChain:
			return encodeRegisterChainPayload(payload.payload)
	}
}

// NOTE: Ideally, this should not be exported so we can contain the complexity here
export function generateKeySet(signers: elliptic.ec.KeyPair[]): Uint8Array[] {
	const slices = signers.map((signer) => {
		const pub = signer.getPublic()
		const x = pub.getX().toBuffer()
		const y = pub.getY().toBuffer()
		const combined = concatArrays([x, y])
		const hashStr = web3Utils.keccak256('0x' + Buffer.from(combined).toString('hex'))
		const hash = new Uint8Array(Buffer.from(hashStr.slice(2), 'hex'))
		const result = hash.slice(12, 32)
		assert(result.length === 20, `Expected 20 bytes for key, got ${result.length}`)
		return result
	})

	return slices
}

export function signWormholeMessage(vaa: UnsignedVAA): SignedVAA {
	assert(vaa.entries.length < 2 ** 8)

	// Generate body of payload
	const body = concatArrays([
		encodeUint32(vaa.header.timestamp),
		encodeUint32(vaa.header.nonce),
		encodeUint16(vaa.header.chainId),
		vaa.header.emitter,
		encodeUint64(vaa.header.sequence),
		encodeUint8(vaa.header.consistencyLevel),
		encodePayload(vaa.payload),
	])

	// Generate hash of body
	const hexHash = web3Utils.keccak256(web3Utils.keccak256('0x' + Buffer.from(body).toString('hex')))
	const hash = new Uint8Array(Buffer.from(hexHash.slice(2), 'hex'))

	// Generate signature set
	const signatures = vaa.entries.map((entry, i) => {
		// Create signature over hash
		const signature: elliptic.ec.Signature = entry.signer.sign(hash, { canonical: true })

		// Create resulting data structure
		const result = concatArrays([
			encodeUint8(i),
			zeroPad(signature.r.toBuffer(), 32),
			zeroPad(signature.s.toBuffer(), 32),
			encodeUint8(signature.recoveryParam ?? 0),
		])

		// Validate result
		assert(result.length === 66)
		
		return result
	})

	// Generate key set data
	const keys = generateKeySet(vaa.entries.map(({ signer }) => signer))

	// Generate data needed to send VAA
	return {
		command: vaa.command,
		gsIndex: vaa.gsIndex,
		signatures,
		keys,
		hash,
		sequence: vaa.header.sequence,
		chainId: vaa.header.chainId,
		emitter: vaa.header.emitter,
		extraTmplSigs: vaa.extraTmplSigs,
		data: concatArrays([
			// Header
			encodeUint8(vaa.version),
			encodeUint32(vaa.gsIndex),
			encodeUint8(vaa.entries.length),

			// Signatures
			...signatures,
			body
		])
	}
}
