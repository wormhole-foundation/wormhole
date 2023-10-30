import { PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
import { TokenBridge } from '@wormhole-foundation/connect-sdk';
export declare function createRegisterChainInstruction(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: TokenBridge.VAA<'RegisterChain'>): TransactionInstruction;
export interface RegisterChainAccounts {
    payer: PublicKey;
    config: PublicKey;
    endpoint: PublicKey;
    vaa: PublicKey;
    claim: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
    wormholeProgram: PublicKey;
}
export declare function getRegisterChainAccounts(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: TokenBridge.VAA<'RegisterChain'>): RegisterChainAccounts;
export declare function createUpgradeContractInstruction(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: TokenBridge.VAA<'UpgradeContract'>, spill?: PublicKeyInitData): TransactionInstruction;
export interface UpgradeContractAccounts {
    payer: PublicKey;
    vaa: PublicKey;
    claim: PublicKey;
    upgradeAuthority: PublicKey;
    spill: PublicKey;
    implementation: PublicKey;
    programData: PublicKey;
    tokenBridgeProgram: PublicKey;
    rent: PublicKey;
    clock: PublicKey;
    bpfLoaderUpgradeable: PublicKey;
    systemProgram: PublicKey;
}
export declare function getUpgradeContractAccounts(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: TokenBridge.VAA<'UpgradeContract'>, spill?: PublicKeyInitData): UpgradeContractAccounts;
