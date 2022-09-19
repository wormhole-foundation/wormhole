//! Functions for creating instructions for CPI calls.

use {
    crate::{
        accounts::Account,
        Config,
        FeeCollector,
        Sequence,
    },
    borsh::{
        BorshDeserialize,
        BorshSerialize,
    },
    solana_program::{
        instruction::{
            AccountMeta,
            Instruction as SolanaInstruction,
        },
        pubkey::Pubkey,
        sysvar,
    },
    wormhole::WormholeError,
};

/// A Copy of the `Instruction` enum from the Solitaire Solana program.
#[repr(u8)]
#[derive(BorshSerialize, BorshDeserialize)]
pub enum Instruction {
    Initialize,
    PostMessage,
    PostVAA,
    SetFees,
    TransferFees,
    UpgradeContract,
    UpgradeGuardianSet,
    VerifySignatures,
    PostMessageUnreliable,
}

#[derive(Debug, PartialEq, BorshSerialize)]
struct PostMessageData<'a> {
    nonce:             u32,
    payload:           &'a [u8],
    consistency_level: u8,
}

pub fn post_message(
    wormhole: Pubkey,
    payer: Pubkey,
    emitter: Pubkey,
    message: Pubkey,
    nonce: u32,
    payload: &[u8],
    consistency_level: u8,
) -> Result<SolanaInstruction, WormholeError> {
    let bridge = Config::key(&wormhole, ());
    let fee_collector = FeeCollector::key(&wormhole, ());
    let sequence = Sequence::key(&wormhole, emitter);

    Ok(SolanaInstruction {
        program_id: wormhole,
        accounts:   vec![
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
        data:       (
            0,
            PostMessageData {
                nonce,
                payload,
                consistency_level,
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
    payload: &[u8],
    consistency_level: u8,
) -> Result<SolanaInstruction, WormholeError> {
    let bridge = Config::key(&program_id, ());
    let fee_collector = FeeCollector::key(&program_id, ());
    let sequence = Sequence::key(&program_id, emitter);

    Ok(SolanaInstruction {
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
            0,
            PostMessageData {
                nonce,
                payload,
                consistency_level,
            },
        )
            .try_to_vec()?,
    })
}

#[cfg(test)]
mod tests {
    use {
        super::Instruction as A,
        bridge::instruction::Instruction as B,
    };

    // This checks that the `Instruction` enum defined in this package has the same variants as the
    // `Instruction` coming from the Solitaire Solana program, the unnecessary looking match is
    // there to force an exhaustiveness check.
    #[test]
    #[rustfmt::skip]
    fn test_variants() {
        match B::Initialize {
            B::Initialize => assert_eq!(A::Initialize as u8, B::Initialize as u8),
            B::PostMessage => assert_eq!(A::PostMessage as u8, B::PostMessage as u8),
            B::PostVAA => assert_eq!(A::PostVAA as u8, B::PostVAA as u8),
            B::SetFees => assert_eq!(A::SetFees as u8, B::SetFees as u8),
            B::TransferFees => assert_eq!(A::TransferFees as u8, B::TransferFees as u8),
            B::UpgradeContract => assert_eq!(A::UpgradeContract as u8, B::UpgradeContract as u8),
            B::UpgradeGuardianSet => assert_eq!(A::UpgradeGuardianSet as u8, B::UpgradeGuardianSet as u8),
            B::VerifySignatures => assert_eq!(A::VerifySignatures as u8, B::VerifySignatures as u8),
            B::PostMessageUnreliable => assert_eq!(A::PostMessageUnreliable as u8, B::PostMessageUnreliable as u8),
        }
    }
}
