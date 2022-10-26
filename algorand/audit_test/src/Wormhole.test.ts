import { TestHelper } from './test_helper'
import elliptic from 'elliptic'
import { concatArrays, decodeUint64, encodeUint16, sha256Hash } from './sdk/Encoding'
import { decodeAddress } from 'algosdk'
import path from 'path'
import { GovenanceMessageType, GovernancePayload, UngeneratedVAA, VAAHeader, VAAPayloadType, WormholeSigner } from './wormhole/WormholeTypes'
import { Wormhole } from './wormhole/Wormhole'
import { CHAIN_ID_ALGORAND } from '@certusone/wormhole-sdk'
import { EMITTER_GOVERNANCE, signWormholeMessage } from './wormhole/WormholeEncoders'
import { WormholeTmplSig } from './wormhole/WormholeTmplSig'
import { generateGovernanceVAA, generateVAA } from './wormhole/WormholeVAA'
import assert from 'assert'

describe('Wormhole', () => {
	const helper: TestHelper = TestHelper.fromConfig()
	let wormhole: Wormhole
	let signers: WormholeSigner[]

	it('Can create signer set', () => {
		const ec = new elliptic.ec("secp256k1")
		signers = [...Array(19)].map(() => ec.genKeyPair())
	})

	it('Can compile tmplSig', async () => {
		const sequence = 0x11223344
		const emitterId = decodeAddress(helper.master).publicKey
		const coreId = 1234

		const tmplSig = await WormholeTmplSig.compileTmplSig(helper.deployer, sequence, emitterId, coreId)
		const tmplSigDirect = new WormholeTmplSig(sequence, emitterId, coreId)
		const bytecode = tmplSigDirect.logicSig.lsig.logic
		const returnOpcode = 0x43
		const bytecodeWithReturn = new Uint8Array([...bytecode, returnOpcode])
		expect(tmplSig.lsig.logic).toEqual(bytecodeWithReturn)
	})

	it('Can sign a VAA', () => {
		const vaa: UngeneratedVAA = {
			command: 'governance',
			version: 1,
			gsIndex: 0,
			header: {
				timestamp: 0,
				nonce: 0,
				chainId: CHAIN_ID_ALGORAND,
				emitter: EMITTER_GOVERNANCE,
				sequence: 0,
				consistencyLevel: 0,
			},
			signers,
			payload: {
				type: VAAPayloadType.Raw,
				payload: new Uint8Array([1, 2, 3, 4])
			},
			extraTmplSigs: [],
		}

		const unsignedVAA = generateVAA(vaa)
		const signedVAA = signWormholeMessage(unsignedVAA)
		
		// Validate VAA following wormhole_core
		let ptr = signedVAA.data.slice(5, 6)[0]
		expect(ptr).toBe(unsignedVAA.entries.length)

		ptr = ptr * 66 + 14
		const emitter = signedVAA.data.slice(ptr, ptr + 34)
		assert(vaa.header.chainId)
		assert(vaa.header.emitter)
		expect(emitter).toEqual(concatArrays([encodeUint16(vaa.header.chainId), vaa.header.emitter]))

		ptr += 34
		// = 1254 + 48 = 1302
		// NOTE: This assumes 19 guardians
		expect(ptr).toBe(1302)

		const sequence = decodeUint64(signedVAA.data.slice(ptr, ptr + 8))
		expect(sequence).toBe(BigInt(unsignedVAA.header.sequence))
	})

	it('Can compile vaa_verify', async () => {
		const dummyWormhole = new Wormhole(helper.deployer, helper.master, 0, 0, helper.signCallback)
		const vaaVerify = await dummyWormhole.compileVaaVerify(helper.deployer)
		void(vaaVerify)
	})

	it('Can be deployed', async () => {
		console.log(`Owner base 64: ${Buffer.from(decodeAddress(helper.master).publicKey).toString('base64')}`)
		wormhole = await helper.deployAndFund(Wormhole.deployAndFund, signers)
		void(wormhole)
	})

	it('Can set message fee', async () => {
		const header: Partial<VAAHeader> = {
			sequence: 1,
			consistencyLevel: 0,
		}

		const payload: GovernancePayload = {
			type: GovenanceMessageType.SetMessageFee,
			targetChainId: CHAIN_ID_ALGORAND,
			messageFee: 1337
		}

		const msg = generateGovernanceVAA(signers, 1, wormhole.coreId, header, payload)
		const signedVaa = signWormholeMessage(msg)
		const txId = await wormhole.sendSignedVAA(signedVaa)
		await helper.waitForTransactionResponse(txId)
	})

	it('Can perform an update', async () => {
		// Compile new core app
		const corePath = path.join(Wormhole.BASE_PATH, 'wormhole_core.py')
		const coreApp = await helper.deployer.makeSourceApp(corePath, Wormhole.CORE_STATE_MAP)
		const coreCompiled = await helper.deployer.makeApp(coreApp)

		// Set the update hash
		const updateHash = sha256Hash(coreCompiled.approval)

		const payload: GovernancePayload = {
			type: GovenanceMessageType.SetUpdateHash,
			targetChainId: CHAIN_ID_ALGORAND,
			updateHash,
		}

		const unsignedVaa = generateGovernanceVAA(signers, 1, wormhole.coreId, { sequence: 2 }, payload)
		const signedVaa = signWormholeMessage(unsignedVaa)
		const txId = await wormhole.sendSignedVAA(signedVaa)
		await helper.waitForTransactionResponse(txId)

		// Perform the update
		// TODO: Perform the update here
	})

	it('Can upgrade guardian set', async () => {
        // const ec = new elliptic.ec("secp256k1")
		// signers = [...Array(19)].map(() => ec.genKeyPair())

		// const payload: GovernancePayload = {
		//     type: GovenanceMessageType.UpdateGuardians,
		//     targetChainId: CHAIN_ID_ALGORAND,
		//     oldGSIndex: 1,
		//     newGSIndex: 2,
		//     guardians: generateKeySet(signers)
		// }

		// const unsignedVaa = generateGovernanceVAA(signers, 1, wormhole.coreId, { sequence: 3 }, payload)
		// const signedVaa = signWormholeMessage(unsignedVaa)
		// const txId = await wormhole.sendSignedVAA(signedVaa, true)
		// await helper.waitForTransactionResponse(txId)
    })

	it('Can perform register chain', async () => {
		// const CHAIN_ID_FOOBAR = 65000
		// const EMITTER_FOOBAR = decodeBase16('ccddeeffccddeeffccddeeffccddeeffccddeeffccddeeffccddeeffccddeeff')
		// const regVaa = wormhole.generateRegisterChainVAA(0, signers, 0, 0, 0, 0, CHAIN_ID_ALGORAND, CHAIN_ID_FOOBAR, EMITTER_FOOBAR)
		// const signedVaa = wormhole.signVAA(regVaa)
		// const txId = await wormhole.sendSignedVAA(signedVaa)
		// await helper.waitForTransactionResponse(txId)
	})

	it('Can perform token transfer', () => {})
	
	it('Can redeem token', () => {})

	it('Rejects invalid VAAs', () => {
		// This should be a randomized test over the data structure
	})

	it('Correctly processes many transactions', () => {
		// Process at least 40_000 VAAs to ensure the sequence number system works correctly
	})
})
