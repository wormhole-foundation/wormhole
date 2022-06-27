use borsh::BorshSerialize;
use solana_program::{
    instruction::{
        AccountMeta,
        Instruction,
    },
    pubkey::Pubkey,
    sysvar,
};

use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use sha3::Digest;
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};
use std::io::{
    Cursor,
    Write,
};

use crate::{
    accounts::{
        Bridge,
        Claim,
        ClaimDerivationData,
        FeeCollector,
        GuardianSet,
        GuardianSetDerivationData,
        PostedVAA,
        PostedVAADerivationData,
        Sequence,
        SequenceDerivationData,
    },
    types::ConsistencyLevel,
    InitializeData,
    PostMessageData,
    PostVAAData,
    SetFeesData,
    TransferFeesData,
    UpgradeContractData,
    UpgradeGuardianSetData,
    VerifySignaturesData,
    CHAIN_ID_SOLANA,
};

pub fn initialize(
    program_id: Pubkey,
    payer: Pubkey,
    fee: u64,
    guardian_set_expiration_time: u32,
    initial_guardians: &[[u8; 20]],
) -> solitaire::Result<Instruction> {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let guardian_set = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData { index: 0 },
        &program_id,
    );
    let fee_collector = FeeCollector::key(None, &program_id);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(bridge, false),
            AccountMeta::new(guardian_set, false),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(sysvar::clock::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],
        data: (
            crate::instruction::Instruction::Initialize,
            InitializeData {
                initial_guardians: initial_guardians.to_vec(),
                fee,
                guardian_set_expiration_time,
            },
        )
            .try_to_vec()?,
    })
}

pub fn post_message(
    program_id: Pubkey,
    payer: Pubkey,
    emitter: Pubkey,
    message: Pubkey,
    nonce: u32,
    payload: Vec<u8>,
    commitment: ConsistencyLevel,
) -> solitaire::Result<Instruction> {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let fee_collector = FeeCollector::<'_>::key(None, &program_id);
    let sequence = Sequence::<'_>::key(
        &SequenceDerivationData {
            emitter_key: &emitter,
        },
        &program_id,
    );

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(bridge, false),
            AccountMeta::new(message, true),
            AccountMeta::new_readonly(emitter, true),
            AccountMeta::new(sequence, false),
            AccountMeta::new(payer, true),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new_readonly(sysvar::clock::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],
        data: (
            crate::instruction::Instruction::PostMessage,
            PostMessageData {
                nonce,
                payload,
                consistency_level: commitment,
            },
        )
            .try_to_vec()?,
    })
}

pub fn post_message_unreliable(
    program_id: Pubkey,
    payer: Pubkey,
    emitter: Pubkey,
    message: Pubkey,
    nonce: u32,
    payload: Vec<u8>,
    commitment: ConsistencyLevel,
) -> solitaire::Result<Instruction> {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let fee_collector = FeeCollector::<'_>::key(None, &program_id);
    let sequence = Sequence::<'_>::key(
        &SequenceDerivationData {
            emitter_key: &emitter,
        },
        &program_id,
    );

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(bridge, false),
            AccountMeta::new(message, true),
            AccountMeta::new_readonly(emitter, true),
            AccountMeta::new(sequence, false),
            AccountMeta::new(payer, true),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new_readonly(sysvar::clock::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],
        data: (
            crate::instruction::Instruction::PostMessageUnreliable,
            PostMessageData {
                nonce,
                payload,
                consistency_level: commitment,
            },
        )
            .try_to_vec()?,
    })
}

pub fn verify_signatures(
    program_id: Pubkey,
    payer: Pubkey,
    guardian_set_index: u32,
    signature_set: Pubkey,
    data: VerifySignaturesData,
) -> solitaire::Result<Instruction> {
    let guardian_set = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData {
            index: guardian_set_index,
        },
        &program_id,
    );

    Ok(Instruction {
        program_id,

        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(guardian_set, false),
            AccountMeta::new(signature_set, true),
            AccountMeta::new_readonly(sysvar::instructions::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],

        data: (crate::instruction::Instruction::VerifySignatures, data).try_to_vec()?,
    })
}

pub fn post_vaa(
    program_id: Pubkey,
    payer: Pubkey,
    signature_set: Pubkey,
    vaa: PostVAAData,
) -> Instruction {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let guardian_set = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData {
            index: vaa.guardian_set_index,
        },
        &program_id,
    );

    let msg_derivation_data = &PostedVAADerivationData {
        payload_hash: hash_vaa(&vaa).to_vec(),
    };

    let message =
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(msg_derivation_data, &program_id);

    Instruction {
        program_id,

        accounts: vec![
            AccountMeta::new_readonly(guardian_set, false),
            AccountMeta::new_readonly(bridge, false),
            AccountMeta::new_readonly(signature_set, false),
            AccountMeta::new(message, false),
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(sysvar::clock::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],

        data: (crate::instruction::Instruction::PostVAA, vaa)
            .try_to_vec()
            .unwrap(),
    }
}

pub fn upgrade_contract(
    program_id: Pubkey,
    payer: Pubkey,
    payload_message: Pubkey,
    emitter: Pubkey,
    new_contract: Pubkey,
    spill: Pubkey,
    sequence: u64,
) -> Instruction {
    let bridge = Bridge::<'_, { AccountState::Initialized }>::key(None, &program_id);
    let claim = Claim::<'_>::key(
        &ClaimDerivationData {
            emitter_address: emitter.to_bytes(),
            emitter_chain: CHAIN_ID_SOLANA,
            sequence,
        },
        &program_id,
    );

    let (upgrade_authority, _) = Pubkey::find_program_address(&["upgrade".as_bytes()], &program_id);

    let (program_data, _) = Pubkey::find_program_address(
        &[program_id.as_ref()],
        &solana_program::bpf_loader_upgradeable::id(),
    );

    Instruction {
        program_id,

        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(bridge, false),
            AccountMeta::new_readonly(payload_message, false),
            AccountMeta::new(claim, false),
            AccountMeta::new_readonly(upgrade_authority, false),
            AccountMeta::new(spill, false),
            AccountMeta::new(new_contract, false),
            AccountMeta::new(program_data, false),
            AccountMeta::new(program_id, false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(sysvar::clock::id(), false),
            AccountMeta::new_readonly(solana_program::bpf_loader_upgradeable::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],

        data: (
            crate::instruction::Instruction::UpgradeContract,
            UpgradeContractData {},
        )
            .try_to_vec()
            .unwrap(),
    }
}

pub fn upgrade_guardian_set(
    program_id: Pubkey,
    payer: Pubkey,
    payload_message: Pubkey,
    emitter: Pubkey,
    old_index: u32,
    new_index: u32,
    sequence: u64,
) -> Instruction {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let claim = Claim::<'_>::key(
        &ClaimDerivationData {
            emitter_address: emitter.to_bytes(),
            emitter_chain: CHAIN_ID_SOLANA,
            sequence,
        },
        &program_id,
    );

    let guardian_set_old = GuardianSet::<'_, { AccountState::Initialized }>::key(
        &GuardianSetDerivationData { index: old_index },
        &program_id,
    );

    let guardian_set_new = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData { index: new_index },
        &program_id,
    );

    Instruction {
        program_id,

        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(bridge, false),
            AccountMeta::new_readonly(payload_message, false),
            AccountMeta::new(claim, false),
            AccountMeta::new(guardian_set_old, false),
            AccountMeta::new(guardian_set_new, false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],

        data: (
            crate::instruction::Instruction::UpgradeGuardianSet,
            UpgradeGuardianSetData {},
        )
            .try_to_vec()
            .unwrap(),
    }
}

pub fn set_fees(
    program_id: Pubkey,
    payer: Pubkey,
    message: Pubkey,
    emitter: Pubkey,
    sequence: u64,
) -> Instruction {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let claim = Claim::<'_>::key(
        &ClaimDerivationData {
            emitter_address: emitter.to_bytes(),
            emitter_chain: CHAIN_ID_SOLANA,
            sequence,
        },
        &program_id,
    );

    Instruction {
        program_id,

        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(bridge, false),
            AccountMeta::new_readonly(message, false),
            AccountMeta::new(claim, false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],

        data: (crate::instruction::Instruction::SetFees, SetFeesData {})
            .try_to_vec()
            .unwrap(),
    }
}

pub fn transfer_fees(
    program_id: Pubkey,
    payer: Pubkey,
    message: Pubkey,
    emitter: Pubkey,
    sequence: u64,
    recipient: Pubkey,
) -> Instruction {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let claim = Claim::<'_>::key(
        &ClaimDerivationData {
            emitter_address: emitter.to_bytes(),
            emitter_chain: CHAIN_ID_SOLANA,
            sequence,
        },
        &program_id,
    );

    let fee_collector = FeeCollector::key(None, &program_id);

    Instruction {
        program_id,

        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(bridge, false),
            AccountMeta::new_readonly(message, false),
            AccountMeta::new(claim, false),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new(recipient, false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],

        data: (
            crate::instruction::Instruction::TransferFees,
            TransferFeesData {},
        )
            .try_to_vec()
            .unwrap(),
    }
}

// Convert a full VAA structure into the serialization of its unique components, this structure is
// what is hashed and verified by Guardians.
pub fn serialize_vaa(vaa: &PostVAAData) -> Vec<u8> {
    let mut v = Cursor::new(Vec::new());
    v.write_u32::<BigEndian>(vaa.timestamp).unwrap();
    v.write_u32::<BigEndian>(vaa.nonce).unwrap();
    v.write_u16::<BigEndian>(vaa.emitter_chain).unwrap();
    v.write_all(&vaa.emitter_address).unwrap();
    v.write_u64::<BigEndian>(vaa.sequence).unwrap();
    v.write_u8(vaa.consistency_level).unwrap();
    v.write_all(&vaa.payload).unwrap();
    v.into_inner()
}

// Hash a VAA, this combines serialization and hashing.
pub fn hash_vaa(vaa: &PostVAAData) -> [u8; 32] {
    let body = serialize_vaa(vaa);
    let mut h = sha3::Keccak256::default();
    h.write_all(body.as_slice()).unwrap();
    h.finalize().into()
}
