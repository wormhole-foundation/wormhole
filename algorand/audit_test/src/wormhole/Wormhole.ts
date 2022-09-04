import { decodeAddress, getApplicationAddress, LogicSigAccount, OnApplicationComplete, Transaction } from 'algosdk'
import { Address, AppId } from '../sdk/AlgorandTypes'
import { concatArrays, encodeUint16 } from "../sdk/Encoding"
import { Deployer, SignCallback, IStateInfo } from "../sdk/Deployer"
import path from 'path'
import assert from 'assert'
import { SignedVAA, WormholeSigner } from './WormholeTypes'
import { signWormholeMessage } from './WormholeEncoders'
import { generateInitVAA } from './WormholeVAA'
import { EMITTER_GUARDIAN, WormholeTmplSig } from './WormholeTmplSig'

export class Wormhole {
	public static BASE_PATH = path.resolve(__dirname, '../../../')

	public static CORE_STATE_MAP: IStateInfo = {
		local: {
			'meta': 'bytes',
			'\x00': 'bytes',
			'\x01': 'bytes',
			'\x02': 'bytes',
			'\x03': 'bytes',
			'\x04': 'bytes',
			'\x05': 'bytes',
			'\x06': 'bytes',
			'\x07': 'bytes',
			'\x08': 'bytes',
			'\x09': 'bytes',
			'\x0A': 'bytes',
			'\x0B': 'bytes',
			'\x0C': 'bytes',
			'\x0D': 'bytes',
			'\x0E': 'bytes',
		},
		global: {
			'MessageFee': 'uint',
			'currentGuardianSetIndex': 'uint',
			'booted': 'bytes',
			'vphash': 'bytes',
			'validUpdateApproveHash': 'bytes',
			'validUpdateClearHash': 'bytes',
		},
	}

	public static BRIDGE_STATE_MAP: IStateInfo = {
		local: {},
		global: {
			'coreid': 'uint',
			'coreAddr': 'bytes',
			'validUpdateApproveHash': 'bytes',
			'validUpdateClearHash': 'bytes',
			
			'chain\x00': 'bytes',
			'chain\x01': 'bytes',
			'chain\x02': 'bytes',
			'chain\x03': 'bytes',
			'chain\x04': 'bytes',
			'chain\x05': 'bytes',
			'chain\x06': 'bytes',
			'chain\x07': 'bytes',
			'chain\x08': 'bytes',
			'chain\x09': 'bytes',

			'chain\x0A': 'bytes',
			'chain\x0B': 'bytes',
			'chain\x0C': 'bytes',
			'chain\x0D': 'bytes',
			'chain\x0E': 'bytes',
			'chain\x0F': 'bytes',
			'chain\x10': 'bytes',
			'chain\x11': 'bytes',
			'chain\x12': 'bytes',
			'chain\x13': 'bytes',

			'chain\x14': 'bytes',
			'chain\x15': 'bytes',
			'chain\x16': 'bytes',
			'chain\x17': 'bytes',
			'chain\x18': 'bytes',
			'chain\x19': 'bytes',
			'chain\x1A': 'bytes',
		},
	}

	public constructor(
		private readonly deployer: Deployer,
		private readonly owner: Address,
		public readonly coreId: AppId,
		public readonly bridgeId: AppId,
		private readonly signCallback: SignCallback,
	) {}

	public static async deployAndFund(
		deployer: Deployer,
		owner: Address,
		signCallback: SignCallback,
		signers: WormholeSigner[],
	): Promise<Wormhole> {
		// Generate paths
		const corePath = path.join(Wormhole.BASE_PATH, 'wormhole_core.py')
		const bridgePath = path.join(Wormhole.BASE_PATH, 'token_bridge.py')

		// Deploy core contract
		const coreApp = await deployer.makeSourceApp(corePath, Wormhole.CORE_STATE_MAP)
		const coreCompiled = await deployer.makeApp(coreApp)
		const coreDeployId = await deployer.deployApplication(owner, coreCompiled, signCallback)
		const coreId = (await deployer.waitForTransactionResponse(coreDeployId))["application-index"]
		console.log(`Wormhole core ID: ${coreId}`)
		console.log(`Core address: ${getApplicationAddress(coreId)}, base64: ${Buffer.from(decodeAddress(getApplicationAddress(coreId)).publicKey).toString('base64')}`)

		// Deploy token bridge
		const bridgeApp = await deployer.makeSourceApp(bridgePath, Wormhole.BRIDGE_STATE_MAP)
		const bridgeCompiled = await deployer.makeApp(bridgeApp)
		// FIXME: The application address should be generated contract side for security
		const bridgeArgs = [coreId, decodeAddress(getApplicationAddress(coreId)).publicKey]
		const bridgeDeployId = await deployer.deployApplication(owner, bridgeCompiled, signCallback, undefined, bridgeArgs)
		const bridgeId = (await deployer.waitForTransactionResponse(bridgeDeployId))["application-index"]
		console.log(`Wormhole bridge ID: ${bridgeId}`)
		console.log(`Bridge address: ${getApplicationAddress(bridgeId)}, base64: ${Buffer.from(decodeAddress(getApplicationAddress(bridgeId)).publicKey).toString('base64')}`)

		// Create object
		const result = new Wormhole(deployer, owner, coreId, bridgeId, signCallback)

		// Initialize applications
		const initUnsigned = generateInitVAA(signers, coreId)
		const initialVaa = await signWormholeMessage(initUnsigned)
		const initTxId = await result.sendSignedVAA(initialVaa)
		await deployer.waitForTransactionResponse(initTxId)
		console.log(`Wormhole initialization complete`)

		return result
	}

	private static splitKeysAndSignatures(vaa: SignedVAA): {sigData: Uint8Array[], keyData: Uint8Array[]} {
		const subSetSize = 7
		const sigData = []
		const keyData = []
		for (let start = 0; start < vaa.signatures.length; start += subSetSize) {
			const end = start + subSetSize
			sigData.push(concatArrays(vaa.signatures.slice(start, end)))
			keyData.push(concatArrays(vaa.keys.slice(start, end)))
		}
		return {sigData, keyData}
	}

	public async compileVaaVerify(deployer: Deployer): Promise<LogicSigAccount> {
		const vaaVerifyPath = path.join(Wormhole.BASE_PATH, 'vaa_verify.py')

		const vaaVerifyApp = await deployer.compileStateless(vaaVerifyPath)

		return vaaVerifyApp
	}

	public async sendSignedVAA(vaa: SignedVAA, dryrunDebug = false): Promise<string> {
		// Generate deduplication template sig
		const deduplicationEmitter = concatArrays([encodeUint16(vaa.chainId), vaa.emitter])
		const bits_per_sig = 8 * 15 * 127
		const sequence = Math.floor(vaa.sequence / bits_per_sig)
		const dedupTmplSig = new WormholeTmplSig(sequence, deduplicationEmitter, this.coreId)
		console.log(`TmplSig address: ${dedupTmplSig.address}, base64: ${Buffer.from(decodeAddress(dedupTmplSig.address).publicKey).toString('base64')}`)

		// Generate guardian template sig
		const guardianTmplSig = new WormholeTmplSig(vaa.gsIndex, EMITTER_GUARDIAN, this.coreId)

		// Opt-in tmplsig
		const templateSigs = [dedupTmplSig, guardianTmplSig].concat(vaa.extraTmplSigs)
		const logicSigMap: Map<Address, LogicSigAccount> = new Map()
		for (const sig of templateSigs) {
			await sig.optin(this.deployer, this.owner, this.coreId, this.signCallback, dryrunDebug)
			logicSigMap.set(sig.address, sig.logicSig)
		}

		// Generate VAA verifier
		const vaaVerify = await this.compileVaaVerify(this.deployer)
		const vaaVerifyId = decodeAddress(vaaVerify.lsig.address()).publicKey
		console.log(`VAA Verify address: ${vaaVerify.address()}, base64: ${Buffer.from(decodeAddress(vaaVerify.address()).publicKey).toString('base64')}`)

		// Generate init transaction group
		const accounts = [dedupTmplSig.address, guardianTmplSig.address].concat(vaa.extraTmplSigs.map((tmplSig) => tmplSig.address))
		let txns: Transaction[]
		switch (vaa.command) {
			case 'init': {
				txns = await Promise.all([
					this.deployer.makeCallTransaction(this.owner, this.coreId, OnApplicationComplete.NoOpOC, ['nop', Math.floor(Math.random() * 2 ** 32)]),
					this.deployer.makeCallTransaction(this.owner, this.coreId, OnApplicationComplete.NoOpOC, ['nop', Math.floor(Math.random() * 2 ** 32)]),
					this.deployer.makeCallTransaction(
						this.owner,
						this.coreId,
						OnApplicationComplete.NoOpOC,
						['init', vaa.data, vaaVerifyId],
						accounts,
					),
					this.deployer.makePayTransaction(this.owner, vaaVerify.address(), BigInt(100_000)),
				])
				break
			}
			case 'governance': {
				const {sigData, keyData} = Wormhole.splitKeysAndSignatures(vaa)
				assert(sigData.length === 3)
				assert(keyData.length === 3)
				
				// Assertions mirroring underlying contract code
				assert(accounts.length >= 2)
				assert(sigData.length > 0)

				sigData.map((sigs, i) => {
					let offset = 6
					const sigLength = sigs.length
					const vaaData = vaa.data.slice(offset, offset + sigLength)
					if (Buffer.compare(vaaData, sigs) !== 0) {
						console.log(`On entry ${i}, expected VAA sig data: ${Buffer.from(vaaData).toString('hex')}\nGot: ${Buffer.from(sigs).toString('hex')}`)
					} else {
						console.log(`Data matched expected for entry ${i}`)
					}

					let guardianCounter = 0
					const endPointer = offset + sigLength
					const keyBuffers: Uint8Array[] = []
					while (offset < endPointer) {
						const guardian = sigs[offset]
						if (guardian !== guardianCounter) {
							// TODO: This is broken and does not generate the same output as the contract!
							console.log(`Expected ${guardianCounter}, got ${guardian}`)
						}
						const guardianKey = keyData[i].slice(guardian * 20, guardian * 20 + 20)
						keyBuffers.push(guardianKey)

						offset += 66
						guardianCounter++
					}

					const allKeyData = concatArrays(keyBuffers)
					if (Buffer.compare(allKeyData, keyData[i]) !== 0) {
						console.log(`On entry ${i}, expected key data ${Buffer.from(keyData[i]).toString('hex')}, got ${Buffer.from(allKeyData).toString('hex')}`)
					} else {
						console.log(`Key matched expected for entry ${i}`)
					}
				})

				// Generate transactions
				txns = await Promise.all([
					this.deployer.makeCallTransaction(vaaVerify.address(), this.coreId, OnApplicationComplete.NoOpOC, ['verifySigs', sigData[0], keyData[0], vaa.hash], accounts, [], [], '', 0),
					this.deployer.makeCallTransaction(vaaVerify.address(), this.coreId, OnApplicationComplete.NoOpOC, ['verifySigs', sigData[1], keyData[1], vaa.hash], accounts, [], [], '', 0),
					this.deployer.makeCallTransaction(vaaVerify.address(), this.coreId, OnApplicationComplete.NoOpOC, ['verifySigs', sigData[2], keyData[2], vaa.hash], accounts, [], [], '', 0),
					this.deployer.makeCallTransaction(this.owner, this.coreId, OnApplicationComplete.NoOpOC, ['verifyVAA', vaa.data], accounts, [], [], '', 0),
					this.deployer.makeCallTransaction(this.owner, this.coreId, OnApplicationComplete.NoOpOC, ['governance', vaa.data], accounts, [], [], '', 5000),
				])

				logicSigMap.set(vaaVerify.address(), vaaVerify)
				
				break
			}
			default: {
				throw new Error(`Unknown command ${vaa.command}`)
			}
		}

		// Call init group
		const initTxId = await this.deployer.callGroupTransaction(txns, logicSigMap, this.signCallback, dryrunDebug)
		
		return initTxId
	}
}
