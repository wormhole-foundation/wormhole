import { CHAIN_ID_ALGORAND, CHAIN_ID_SOLANA } from '@certusone/wormhole-sdk'
import assert from 'assert'
import { AppId } from '../sdk/AlgorandTypes'
import { EMITTER_GOVERNANCE, generateKeySet } from './WormholeEncoders'
import { EMITTER_GUARDIAN, WormholeTmplSig } from './WormholeTmplSig'
import { UnsignedVAA, GovernancePayload, UngeneratedVAA, GovenanceMessageType, WormholeSigner, VAAHeader, VAAPayloadType } from './WormholeTypes'

// Takes an ungenerated VAA and generates a fully prepared VAA
// The ungenerated VAA represents a signle header, this function spreads it across the signer set
export function generateVAA(vaa: UngeneratedVAA): UnsignedVAA {
	assert(vaa.gsIndex < vaa.signers.length && vaa.gsIndex >= 0 && Number.isSafeInteger(vaa.gsIndex))

	const header: VAAHeader = {
		timestamp: vaa.header.timestamp ?? 0,
		nonce: vaa.header.nonce ?? Math.floor(Math.random() * (2 ** 32)),
		chainId: vaa.header.chainId ?? CHAIN_ID_SOLANA,
		emitter: vaa.header.emitter ?? EMITTER_GOVERNANCE,
		sequence: vaa.header.sequence ?? 0,
		consistencyLevel: vaa.header.consistencyLevel ?? 0,
	}

	return {
		command: vaa.command,
		version: vaa.version,
		gsIndex: vaa.gsIndex,
		header,
		entries: vaa.signers.map((signer) => ({
			signer,
			header,
			payload: vaa.payload,
		})),
		payload: vaa.payload,
		extraTmplSigs: vaa.extraTmplSigs,
	}
}

// Generates the initialization transaction
export function generateInitVAA(signers: WormholeSigner[], coreId: AppId): UnsignedVAA {
	const header: VAAHeader = {
		timestamp: 0,
		nonce: 0,
		chainId: CHAIN_ID_SOLANA,
		emitter: EMITTER_GOVERNANCE,
		sequence: 0,
		consistencyLevel: 0,
	}

	const payload: GovernancePayload = {
		type: GovenanceMessageType.UpdateGuardians,
		targetChainId: CHAIN_ID_ALGORAND,
		oldGSIndex: 0,
		newGSIndex: 1,
		guardians: generateKeySet(signers),
	}

	const result = generateGovernanceVAA(signers, 0, coreId, header, payload)

	result.command = 'init'
	
	return result
}

export function generateGovernanceVAA(signers: WormholeSigner[], gsIndex: number, coreId: AppId, header: Partial<VAAHeader>, payload: GovernancePayload): UnsignedVAA {
	const template: UngeneratedVAA = {
		command: 'governance',
		version: 1,
		gsIndex,
		header,
		signers,
		payload: {
			type: VAAPayloadType.Governance,
			payload,
		},
		extraTmplSigs: [],
	}

	switch (payload.type) {
		case GovenanceMessageType.UpdateGuardians: {
			const newGuardianTmplSig = new WormholeTmplSig(payload.newGSIndex, EMITTER_GUARDIAN, coreId)
			template.extraTmplSigs.push(newGuardianTmplSig)
			break
		}
	}

	return generateVAA(template)
}
