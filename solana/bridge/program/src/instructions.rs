use borsh::BorshSerialize;
use solana_program::{
    instruction::{
        AccountMeta,
        Instruction,
    },
    pubkey::Pubkey,
    sysvar,
};

use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};

use crate::{
    accounts::{
        Bridge,
        FeeCollector,
        GuardianSet,
        GuardianSetDerivationData,
        Message,
        MessageDerivationData,
        Sequence,
        SequenceDerivationData,
        SignatureSet,
        SignatureSetDerivationData,
    },
    BridgeConfig,
    PostMessageData,
    PostVAAData,
    VerifySignaturesData,
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
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],
        data: crate::instruction::Instruction::Initialize(InitializeData {
            initial_guardians: initial_guardians.to_vec(),
            fee,
            guardian_set_expiration_time,
        })
        .try_to_vec()?,
    })
}

pub fn post_message(
    program_id: Pubkey,
    payer: Pubkey,
    emitter: Pubkey,
    nonce: u32,
    payload: Vec<u8>,
) -> solitaire::Result<Instruction> {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let fee_collector = FeeCollector::<'_>::key(None, &program_id);
    let sequence = Sequence::<'_>::key(
        &SequenceDerivationData {
            emitter_key: &emitter,
        },
        &program_id,
    );
    let message = Message::<'_, { AccountState::Uninitialized }>::key(
        &MessageDerivationData {
            emitter_key: emitter.to_bytes(),
            emitter_chain: 1,
            nonce,
            payload: payload.clone(),
        },
        &program_id,
    );

    Ok(Instruction {
        program_id,

        accounts: vec![
            AccountMeta::new(bridge, false),
            AccountMeta::new(message, false),
            AccountMeta::new(emitter, true),
            AccountMeta::new(sequence, false),
            AccountMeta::new(payer, true),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new_readonly(sysvar::clock::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],

        data: crate::instruction::Instruction::PostMessage(PostMessageData {
            nonce,
            payload: payload.clone(),
        })
        .try_to_vec()?,
    })
}

pub fn verify_signatures(
    program_id: Pubkey,
    payer: Pubkey,
    guardian_set_index: u32,
    data: VerifySignaturesData,
) -> solitaire::Result<Instruction> {
    let guardian_set = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData {
            index: guardian_set_index,
        },
        &program_id,
    );

    let signature_set = SignatureSet::<'_, { AccountState::Uninitialized }>::key(
        &SignatureSetDerivationData { hash: data.hash },
        &program_id,
    );

    Ok(Instruction {
        program_id,

        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(guardian_set, false),
            AccountMeta::new(signature_set, false),
            AccountMeta::new_readonly(sysvar::instructions::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],

        data: crate::instruction::Instruction::VerifySignatures(data).try_to_vec()?,
    })
}

pub fn post_vaa(program_id: Pubkey, payer: Pubkey, vaa: PostVAAData) -> Instruction {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let guardian_set = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData {
            index: vaa.guardian_set_index,
        },
        &program_id,
    );

    let signature_set = SignatureSet::<'_, { AccountState::Uninitialized }>::key(
        &SignatureSetDerivationData {
            hash: hash_vaa(&vaa),
        },
        &program_id,
    );

    let message = Message::<'_, { AccountState::MaybeInitialized }>::key(
        &MessageDerivationData {
            emitter_key: vaa.emitter_address,
            emitter_chain: vaa.emitter_chain,
            nonce: vaa.nonce,
            payload: vaa.payload.clone(),
        },
        &program_id,
    );

    Instruction {
        program_id,

        accounts: vec![
            AccountMeta::new(guardian_set, false),
            AccountMeta::new(bridge, false),
            AccountMeta::new(signature_set, false),
            AccountMeta::new(message, false),
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(sysvar::clock::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],

        data: crate::instruction::Instruction::PostVAA(vaa)
            .try_to_vec()
            .unwrap(),
    }
}

// Convert a full VAA structure into the serialization of its unique components, this structure is
// what is hashed and verified by Guardians.
pub fn serialize_vaa(vaa: &PostVAAData) -> Vec<u8> {
    use byteorder::{
        BigEndian,
        WriteBytesExt,
    };
    use std::io::{
        Cursor,
        Write,
    };

    let mut v = Cursor::new(Vec::new());
    v.write_u32::<BigEndian>(vaa.timestamp).unwrap();
    v.write_u32::<BigEndian>(vaa.nonce).unwrap();
    v.write_u16::<BigEndian>(vaa.emitter_chain).unwrap();
    v.write(&vaa.emitter_address).unwrap();
    v.write_u64::<BigEndian>(vaa.sequence).unwrap();
    v.write(&vaa.payload).unwrap();
    v.into_inner()
}

// Hash a VAA, this combines serialization and hashing.
pub fn hash_vaa(vaa: &PostVAAData) -> [u8; 32] {
    use sha3::Digest;
    use std::io::Write;

    let body = serialize_vaa(vaa);
    let mut h = sha3::Keccak256::default();
    h.write(body.as_slice()).unwrap();
    h.finalize().into()
}
