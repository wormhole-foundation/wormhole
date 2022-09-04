import { Address } from '../sdk/AlgorandTypes'
import elliptic from 'elliptic'
import { WormholeTmplSig } from './WormholeTmplSig'

export type WormholeSigner = elliptic.ec.KeyPair
export type VAACommand = 'init' | 'governance'

// Wormhole pipeline
// 1. Raw data -> generate*VAA (WormholeVAA.ts)
//    Whatever data is needed is passed in to the corresponding generator/helper
//    These functions mostly hold the constants and common generation tasks for a given message type
//
//    generateVAA itself is used as the final step before signing, to set up all members to sign the same message(standard use case for testing)
//
// 2. generate<x>VAA -> signWormholeMessage (WormholeEncoders.ts)
//    After generateVAA runs inside the generate*VAA function, the UnsignedVAA is encoded and signed by signWormholeMessage

// Core types
// The VAA header is a common data structure shared across the various stages of the pipeline below
export type VAAHeader = {
	timestamp: number
	nonce: number
	chainId: number
	emitter: Uint8Array
	sequence: number
	consistencyLevel: number
}

export type UngeneratedVAA = {
	command: VAACommand
	version: number
	gsIndex: number
	header: Partial<VAAHeader>
	signers: elliptic.ec.KeyPair[]
	payload: VAAPayload
	extraTmplSigs: WormholeTmplSig[]
}

export type VAAEntry = {
	signer: WormholeSigner
	header: VAAHeader
	payload: VAAPayload
}

// Unsigned VAAs 
export type UnsignedVAA = {
	command: VAACommand
	version: number
	gsIndex: number
	header: VAAHeader
	entries: VAAEntry[]

	payload: VAAPayload
	extraTmplSigs: WormholeTmplSig[]
}

// This encodes all information required to generate the transaction stack. A SignedVAA is one step before being a raw transaction
export type SignedVAA = {
	command: string
	gsIndex: number
	signatures: Uint8Array[]
	keys: Uint8Array[]
	hash: Uint8Array
	data: Uint8Array
	sequence: number
	chainId: number
	emitter: Uint8Array
	extraTmplSigs: WormholeTmplSig[]
}

// Payload types
export enum VAAPayloadType {
	Raw,
	Governance,
	RegisterChain,
}

export type VAAPayload = {
	type: VAAPayloadType.Raw
	payload: Uint8Array
} | {
	type: VAAPayloadType.Governance
	payload: GovernancePayload
} | {
	type: VAAPayloadType.RegisterChain
	payload: RegisterChainPayload
}

// Governance payload type
export enum GovenanceMessageType {
	SetUpdateHash = 1,
	UpdateGuardians = 2,
	SetMessageFee = 3,
	SendAlgo = 4,
}

export type GovernancePayload = {
	// NOTE: I'm not sure if this belongs here, or if it should go in the unsigned VAA, it seems to be common on all messages?
	targetChainId: number
} & ({
	type: GovenanceMessageType.SetUpdateHash
	updateHash: Uint8Array
} | {
	type: GovenanceMessageType.UpdateGuardians
	oldGSIndex: number
	newGSIndex: number
	guardians: Uint8Array[]
} | {
	type: GovenanceMessageType.SetMessageFee
	messageFee: number
} | {
	type: GovenanceMessageType.SendAlgo
	unknown: Uint8Array // 24 bytes
	fee: number
	dest: Address
})

// Register chain payload type
//
// Since anyone can use Wormhole to publish messages that match the payload format of the token bridge, an authorization payload needs to be implemented. 
// This is done using an (emitter_chain, emitter_address) tuple. Every endpoint of the token bridge needs to know the addresses of the respective other 
// endpoints on other chains. This registration of token bridge endpoints is implemented via RegisterChain where a (chain_id, emitter_address) tuple can be registered. 

export type RegisterChainPayload = {
	targetChainId: number
	emitterChainId: number
	emitterAddress: Uint8Array
}
