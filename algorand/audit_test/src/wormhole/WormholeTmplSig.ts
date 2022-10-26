import { decodeAddress, getApplicationAddress, LogicSigAccount, OnApplicationComplete } from 'algosdk'
import { Address, AppId } from '../sdk/AlgorandTypes'
import { concatArrays, decodeBase16, AlgorandType } from "../sdk/Encoding"
import { Deployer, SignCallback } from "../sdk/Deployer"
import path from 'path'
import varint from 'varint'
import { Wormhole } from './Wormhole'

export const EMITTER_GUARDIAN = new TextEncoder().encode("guardian")

export class WormholeTmplSig {
	private _logicSig: LogicSigAccount

	public constructor(
		sequence: number,
		emitterId: Uint8Array,
		coreId: AppId,
	) {
		// Generate the template sig bytecode directly
		const program = concatArrays([
			decodeBase16('0620010181'),
			varint.encode(sequence),
			decodeBase16('4880'),
			varint.encode(emitterId.length),
			emitterId,
			decodeBase16('483110810612443119221244311881'),
			varint.encode(coreId),
			decodeBase16('124431208020'),
			decodeAddress(getApplicationAddress(coreId)).publicKey,
			decodeBase16('124431018100124431093203124431153203124422'),
		])

		this._logicSig = new LogicSigAccount(program)
	}

	public get logicSig(): LogicSigAccount {
		return this._logicSig
	}

	public get address(): Address {
		return this.logicSig.address()
	}

	public async optin(deployer: Deployer, owner: Address, coreId: AppId, signCallback: SignCallback, dryrunDebug = false): Promise<void> {
		// Test if contract is opted in
		const optedIn = await deployer.readOptedInApps(this.address)
		if (optedIn.find((entry) => entry.id === coreId) === undefined) {
			console.log(`Performing optin for TmplSig ${this.address}`)
			const tmplSigPayment = BigInt(1_002_000)
			const optinTxns = await Promise.all([
				deployer.makePayTransaction(owner, this.address, tmplSigPayment, 2 * deployer.minFee),
				deployer.makeCallTransaction(this.address, coreId, OnApplicationComplete.OptInOC, [], [], [], [], '', 0, getApplicationAddress(coreId))
			])

			const initTxId = await deployer.callGroupTransaction(optinTxns, new Map([[this.address, this.logicSig]]), signCallback, dryrunDebug)
			await deployer.waitForTransactionResponse(initTxId)
		}
	}

	public static async compileTmplSig(deployer: Deployer, sequence: number | bigint, emitterId: Uint8Array, coreId: AppId): Promise<LogicSigAccount> {
		const tmplSigPath = path.join(Wormhole.BASE_PATH, 'TmplSig.py')
	
		const tmplSigArgs = new Map<string, AlgorandType>([
			['TMPL_ADDR_IDX', sequence],
			['TMPL_EMITTER_ID', emitterId],
			['TMPL_APP_ID', coreId],
			// FIXME: The application address should be generated contract side for security
			['TMPL_APP_ADDRESS', decodeAddress(getApplicationAddress(coreId)).publicKey],
		])

		const tmplSigApp = await deployer.compileStateless(tmplSigPath, tmplSigArgs)
	
		return tmplSigApp
	}
}
