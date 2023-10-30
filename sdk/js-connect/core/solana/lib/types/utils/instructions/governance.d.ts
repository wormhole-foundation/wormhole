import { Connection, PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
import { VAA } from '@wormhole-foundation/connect-sdk';
export declare function createSetFeesInstruction(connection: Connection, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: VAA<'WormholeCore:SetMessageFee'>): TransactionInstruction;
export interface SetFeesAccounts {
    payer: PublicKey;
    bridge: PublicKey;
    vaa: PublicKey;
    claim: PublicKey;
    systemProgram: PublicKey;
}
export declare function getSetFeesAccounts(wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: VAA<'WormholeCore:SetMessageFee'>): SetFeesAccounts;
export declare function createTransferFeesInstruction(connection: Connection, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, recipient: PublicKeyInitData, vaa: VAA<'WormholeCore:TransferFees'>): TransactionInstruction;
export interface TransferFeesAccounts {
    payer: PublicKey;
    bridge: PublicKey;
    vaa: PublicKey;
    claim: PublicKey;
    feeCollector: PublicKey;
    recipient: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
}
export declare function getTransferFeesAccounts(wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, recipient: PublicKeyInitData, vaa: VAA<'WormholeCore:TransferFees'>): TransferFeesAccounts;
export declare function createUpgradeGuardianSetInstruction(connection: Connection, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: VAA<'WormholeCore:GuardianSetUpgrade'>): TransactionInstruction;
export interface UpgradeGuardianSetAccounts {
    payer: PublicKey;
    bridge: PublicKey;
    vaa: PublicKey;
    claim: PublicKey;
    guardianSetOld: PublicKey;
    guardianSetNew: PublicKey;
    systemProgram: PublicKey;
}
export declare function getUpgradeGuardianSetAccounts(wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: VAA<'WormholeCore:GuardianSetUpgrade'>): UpgradeGuardianSetAccounts;
export declare function createUpgradeContractInstruction(connection: Connection, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: VAA<'WormholeCore:UpgradeContract'>): TransactionInstruction;
export interface UpgradeContractAccounts {
    payer: PublicKey;
    bridge: PublicKey;
    vaa: PublicKey;
    claim: PublicKey;
    upgradeAuthority: PublicKey;
    spill: PublicKey;
    implementation: PublicKey;
    programData: PublicKey;
    wormholeProgram: PublicKey;
    rent: PublicKey;
    clock: PublicKey;
    bpfLoaderUpgradeable: PublicKey;
    systemProgram: PublicKey;
}
export declare function getUpgradeContractAccounts(wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: VAA<'WormholeCore:UpgradeContract'>, spill?: PublicKeyInitData): UpgradeContractAccounts;
