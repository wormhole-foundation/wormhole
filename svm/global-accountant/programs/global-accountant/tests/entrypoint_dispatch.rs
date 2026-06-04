//! Host-side dispatch tests for `process_instruction`.
//!
//! No accounts are constructed: every case asserts a rejection that fires
//! before any account access, which is exactly the surface the dispatch layer
//! and the not-yet-implemented handlers expose. Behavioural coverage of
//! `submit_observations` arrives with the SVM test harness in a later commit.

use global_accountant::definitions::{GlobalAccountantError, Instruction};
use global_accountant::entrypoint::process_instruction;
use pinocchio::{error::ProgramError, Address};

fn program_id() -> Address {
    Address::from([7u8; 32])
}

fn custom(e: GlobalAccountantError) -> ProgramError {
    ProgramError::Custom(e as u32)
}

#[test]
fn empty_instruction_data_rejected() {
    let result = process_instruction(&program_id(), &mut [], &[]);
    assert_eq!(
        result,
        Err(custom(GlobalAccountantError::InvalidInstructionData))
    );
}

#[test]
fn unknown_discriminator_rejected() {
    // 0 is reserved (no instruction claims it); 7+ are out of range.
    for discriminator in [0u8, 7, 42, 255] {
        let result = process_instruction(&program_id(), &mut [], &[discriminator]);
        assert_eq!(
            result,
            Err(custom(GlobalAccountantError::InvalidInstruction)),
            "discriminator {discriminator} must be rejected"
        );
    }
}

#[test]
fn unimplemented_instructions_return_not_implemented() {
    for instruction in [
        Instruction::CloseDigest,
        Instruction::ClosePending,
        Instruction::SubmitVaas,
        Instruction::RegisterChain,
        Instruction::ModifyBalance,
    ] {
        let data = [instruction as u8];
        let result = process_instruction(&program_id(), &mut [], &data);
        assert_eq!(
            result,
            Err(custom(GlobalAccountantError::NotImplemented)),
            "{instruction:?} must be a NotImplemented stub"
        );
    }
}

#[test]
fn submit_observations_short_data_rejected() {
    // Dispatches into the real handler; its fixed-prefix length check (102
    // bytes + 2-byte body length) fires before any account access.
    let mut data = vec![Instruction::SubmitObservations as u8];
    data.extend_from_slice(&[0u8; 50]);
    let result = process_instruction(&program_id(), &mut [], &data);
    assert_eq!(
        result,
        Err(custom(GlobalAccountantError::InvalidInstructionData))
    );
}

#[test]
fn submit_observations_short_body_rejected() {
    // Full 102-byte fixed prefix, but the declared body length (51 bytes) is
    // below the 52-byte minimum (VAA header + action byte).
    let mut data = vec![Instruction::SubmitObservations as u8];
    data.extend_from_slice(&[0u8; 102]);
    data.extend_from_slice(&51u16.to_le_bytes());
    data.extend_from_slice(&[0u8; 51]);
    let result = process_instruction(&program_id(), &mut [], &data);
    assert_eq!(
        result,
        Err(custom(GlobalAccountantError::InvalidInstructionData))
    );
}

#[test]
fn submit_observations_missing_body_length_rejected() {
    // Exactly the 102-byte fixed prefix with no 2-byte body-length field —
    // one byte short of the minimum wire size.
    let mut data = vec![Instruction::SubmitObservations as u8];
    data.extend_from_slice(&[0u8; 102]);
    data.push(0);
    let result = process_instruction(&program_id(), &mut [], &data);
    assert_eq!(
        result,
        Err(custom(GlobalAccountantError::InvalidInstructionData))
    );
}

#[test]
fn submit_observations_truncated_body_rejected() {
    // Valid declared body length (60 ≥ the 52-byte minimum), but fewer body
    // bytes actually supplied — the buffer-vs-declared-length check fires.
    let mut data = vec![Instruction::SubmitObservations as u8];
    data.extend_from_slice(&[0u8; 102]);
    data.extend_from_slice(&60u16.to_le_bytes());
    data.extend_from_slice(&[0u8; 40]);
    let result = process_instruction(&program_id(), &mut [], &data);
    assert_eq!(
        result,
        Err(custom(GlobalAccountantError::InvalidInstructionData))
    );
}
