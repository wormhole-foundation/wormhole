use borsh::BorshSerialize;
use solitaire::processors::seeded::Seeded;
use solitaire::AccountState;
use solana_program::{
    borsh::try_from_slice_unchecked,
    hash,
    instruction::{
        AccountMeta,
        Instruction,
    },
    program_pack::Pack,
    pubkey::Pubkey,
    system_instruction::{
        self,
        create_account,
    },
    system_program,
    sysvar,
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
        SignaturesSetDerivationData,
    },
    types::PostedMessage,
    BridgeConfig,
    PostMessageData,
    PostVAAData,
    VerifySignaturesData,
};

pub fn initialize(
    program_id: Pubkey,
    payer: Pubkey,
    bridge: Pubkey,
    guardian_set_index: u32,
    guardian_set: Pubkey,
    fee: u64,
    guardian_set_expiration_time: u32,
) -> solitaire::Result<Instruction> {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let guardian_set = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData {
            index: guardian_set_index,
        },
        &program_id,
    );

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(bridge, false),
            AccountMeta::new(guardian_set, false),
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],
        data: crate::instruction::Instruction::Initialize(BridgeConfig {
            fee,
            guardian_set_expiration_time,
        })
        .try_to_vec()?,
    })
}

pub fn post_message(
    program_id: Pubkey,
    payer: Pubkey,
    bridge: Pubkey,
    emitter: Pubkey,
    message: PostedMessage,
    sequence: u64,
) -> solitaire::Result<Instruction> {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let fee_collector = FeeCollector::<'_>::key(None, &program_id);
    let sequence = Sequence::<'_>::key(&SequenceDerivationData {
        emitter_key: &emitter,
    }, &program_id);
    let message = Message::<'_, { AccountState::Uninitialized }>::key(&MessageDerivationData {
        emitter_key: emitter.to_bytes(),
        emitter_chain: message.emitter_chain,
        nonce: message.nonce,
        payload: message.payload.clone(),
    }, &program_id);

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
            nonce: 0,
            payload: vec![],
        })
        .try_to_vec()?
    })
}

pub fn verify_signatures(
    program_id: Pubkey,
    payer: Pubkey,
    guardian_set_index: u32,
    hash: [u8; 32],
) -> solitaire::Result<Instruction> {
    let guardian_set = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData {
            index: guardian_set_index,
        },
        &program_id,
    );

    let signature_set = SignatureSet::<'_, { AccountState::Uninitialized }>::key(
        &SignaturesSetDerivationData { hash },
        &program_id,
    );

    // Bridge with a single pre-existing signer.
    // TODO: Get rid of this, exists to make testing easier for now.
    let mut signers = [-1; 19];
    signers[0] = 0;

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

        data: crate::instruction::Instruction::VerifySignatures(VerifySignaturesData {
            hash,
            signers,
            initial_creation: true,
        })
        .try_to_vec()?
    })
}

pub fn post_vaa(
    program_id: Pubkey,
    payer: Pubkey,
    hash: [u8; 32],
    message: Vec<u8>,
    emitter: Pubkey,
    guardian_set_index: u32,
    vaa: PostVAAData,
) -> Instruction {
    let bridge = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let guardian_set = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData {
            index: guardian_set_index,
        },
        &program_id,
    );

    let signature_set = SignatureSet::<'_, { AccountState::Uninitialized }>::key(
        &SignaturesSetDerivationData { hash },
        &program_id,
    );

    let message = Message::<'_, { AccountState::MaybeInitialized }>::key(
        &MessageDerivationData {
            emitter_key: emitter.to_bytes(),
            emitter_chain: 1,
            nonce: 0,
            payload: message,
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
