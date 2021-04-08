#![allow(clippy::too_many_arguments)]
//! Instruction types

use std::mem::size_of;

use solana_program::{
    instruction::{AccountMeta, Instruction},
    program_error::ProgramError,
    pubkey::Pubkey,
};

use crate::{
    instruction::BridgeInstruction::{
        Initialize, PostVAA, VerifySignatures,
    },
    state::{Bridge, BridgeConfig},
    vaa::{VAABody, VAA},
};
use crate::instruction::BridgeInstruction::PublishMessage;
use std::io::{Cursor, Read, Write};
use byteorder::{ReadBytesExt, BigEndian, WriteBytesExt};

/// chain id of this chain
pub const CHAIN_ID_SOLANA: u8 = 1;
/// maximum number of guardians
pub const MAX_LEN_GUARDIAN_KEYS: usize = 20;
/// maximum size of a posted VAA
pub const MAX_VAA_SIZE: usize = 1000;
/// maximum size of a posted VAA
pub const MAX_PAYLOAD_SIZE: usize = 400;
/// size of a foreign address in bytes
const FOREIGN_ADDRESS_SIZE: usize = 32;

/// serialized VAA data
pub type VAAData = Vec<u8>;
/// X and Y point of P for guardians
pub type GuardianKey = [u8; 64];
/// address on a foreign chain
pub type ForeignAddress = [u8; FOREIGN_ADDRESS_SIZE];

#[repr(C)]
#[derive(Clone, Copy)]
pub struct InitializePayload {
    /// number of initial guardians
    pub len_guardians: u8,
    /// guardians that are allowed to sign mints
    pub initial_guardian: [[u8; 20]; MAX_LEN_GUARDIAN_KEYS],
    /// config for the bridge
    pub config: BridgeConfig,
}

pub struct PublishMessagePayload {
    /// unique nonce for this message
    pub nonce: u32,
    /// message payload
    pub payload: Vec<u8>,
}


impl Clone for PublishMessagePayload {
    fn clone(&self) -> PublishMessagePayload {
        let payload = self.payload.clone();
        return PublishMessagePayload {
            payload,
            nonce: self.nonce,
        };
    }
}

#[derive(Clone, Copy, Debug)]
pub struct VerifySigPayload {
    /// hash of the VAA
    pub hash: [u8; 32],
    /// instruction indices of signers (-1 for missing)
    pub signers: [i8; MAX_LEN_GUARDIAN_KEYS],
    /// indicates whether this verification should only succeed if the sig account does not exist
    pub initial_creation: bool,
}

/// Instructions supported by the SwapInfo program.
#[repr(C)]
pub enum BridgeInstruction {
    /// Initializes a new Bridge
    /// Accounts expected by this instruction:
    ///
    ///   0. `[writable, derived]`  The bridge to initialize.
    ///   1. `[]` The System program
    ///   2. `[]` The clock SysVar
    ///   3. `[writable, derived]` The initial guardian set account
    ///   4. `[signer]` The fee payer for new account creation
    Initialize(InitializePayload),

    /// Publishes a message over the Wormhole network.
    /// See docs for accounts
    PublishMessage(PublishMessagePayload),

    /// Submits a VAA signed by `guardian` on a valid `proposal`.
    /// See docs for accounts
    PostVAA(VAAData),

    /// Verifies signature instructions
    VerifySignatures(VerifySigPayload),
}

impl BridgeInstruction {
    /// Deserializes a byte buffer into a BridgeInstruction
    pub fn deserialize(input: &[u8]) -> Result<Self, ProgramError> {
        if input.len() < size_of::<u8>() {
            return Err(ProgramError::InvalidAccountData);
        }
        Ok(match input[0] {
            0 => {
                let payload: &InitializePayload = unpack(input)?;

                Initialize(*payload)
            }
            1 => {
                let mut payload_data = Cursor::new(input);

                let nonce = payload_data.read_u32::<BigEndian>().map_err(|_| ProgramError::InvalidArgument)?;
                let mut message_payload: Vec<u8> = vec![];
                payload_data.read(&mut message_payload).map_err(|_| ProgramError::InvalidArgument)?;

                let payload: PublishMessagePayload = PublishMessagePayload {
                    nonce,
                    payload: message_payload,
                };

                PublishMessage(payload)
            }
            2 => {
                let payload: VAAData = input[1..].to_vec();
                PostVAA(payload)
            }
            3 => {
                let payload: &VerifySigPayload = unpack(input)?;

                VerifySignatures(*payload)
            }
            _ => return Err(ProgramError::InvalidInstructionData),
        })
    }

    /// Serializes a BridgeInstruction into a byte buffer.
    pub fn serialize(self: Self) -> Result<Vec<u8>, ProgramError> {
        let mut output = Vec::with_capacity(size_of::<BridgeInstruction>());

        match self {
            Self::Initialize(payload) => {
                output.resize(size_of::<InitializePayload>() + 1, 0);
                output[0] = 0;
                #[allow(clippy::cast_ptr_alignment)]
                    let value = unsafe {
                    &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut InitializePayload)
                };
                *value = payload;
            }
            Self::PublishMessage(payload) => {
                let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());
                v.write_u8(1).map_err(|_| ProgramError::InvalidArgument)?;
                v.write_u32::<BigEndian>(payload.nonce).map_err(|_| ProgramError::InvalidArgument)?;
                v.write(&payload.payload).map_err(|_| ProgramError::InvalidArgument)?;

                output = v.into_inner();
            }
            Self::PostVAA(payload) => {
                output.resize(1, 0);
                output[0] = 2;
                #[allow(clippy::cast_ptr_alignment)]
                    output.extend_from_slice(&payload);
            }
            Self::VerifySignatures(payload) => {
                output.resize(size_of::<VerifySigPayload>() + 1, 0);
                output[0] = 3;
                #[allow(clippy::cast_ptr_alignment)]
                    let value = unsafe {
                    &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut VerifySigPayload)
                };
                *value = payload;
            }
        }
        Ok(output)
    }
}

/// Creates an 'initialize' instruction.
#[cfg(not(target_arch = "bpf"))]
pub fn initialize(
    program_id: &Pubkey,
    sender: &Pubkey,
    initial_guardian: Vec<[u8; 20]>,
    config: &BridgeConfig,
) -> Result<Instruction, ProgramError> {
    if initial_guardian.len() > MAX_LEN_GUARDIAN_KEYS {
        return Err(ProgramError::InvalidArgument);
    }
    let mut initial_g = [[0u8; 20]; MAX_LEN_GUARDIAN_KEYS];
    for (i, key) in initial_guardian.iter().enumerate() {
        initial_g[i] = *key;
    }
    let data = BridgeInstruction::Initialize(InitializePayload {
        config: *config,
        len_guardians: initial_guardian.len() as u8,
        initial_guardian: initial_g,
    })
        .serialize()?;

    let bridge_key = Bridge::derive_bridge_id(program_id)?;
    let guardian_set_key = Bridge::derive_guardian_set_id(program_id, &bridge_key, 0)?;

    let accounts = vec![
        AccountMeta::new_readonly(solana_program::system_program::id(), false),
        AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
        AccountMeta::new(bridge_key, false),
        AccountMeta::new(guardian_set_key, false),
        AccountMeta::new(*sender, true),
    ];

    Ok(Instruction {
        program_id: *program_id,
        accounts,
        data,
    })
}

/// Creates an 'TransferOut' instruction.
#[cfg(not(target_arch = "bpf"))]
pub fn post_message(
    program_id: &Pubkey,
    payer: &Pubkey,
    t: &PublishMessagePayload,
) -> Result<Instruction, ProgramError> {
    let bridge_key = Bridge::derive_bridge_id(program_id)?;

    let message_key = Bridge::derive_message_id(
        program_id,
        &bridge_key,
        CHAIN_ID_SOLANA,
        payer.to_bytes(),
        t.nonce,
        t.payload.clone(),
    )?;

    let mut accounts = vec![
        AccountMeta::new_readonly(*program_id, false),
        AccountMeta::new_readonly(solana_program::system_program::id(), false),
        AccountMeta::new_readonly(spl_token::id(), false),
        AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
        AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
        AccountMeta::new_readonly(solana_program::sysvar::instructions::id(), false),
        AccountMeta::new_readonly(bridge_key, false),
        AccountMeta::new(message_key, false),
        AccountMeta::new(*payer, true),
    ];

    let data = BridgeInstruction::PublishMessage(t.clone()).serialize()?;

    Ok(Instruction {
        program_id: *program_id,
        accounts,
        data,
    })
}

/// Creates a 'VerifySignatures' instruction.
#[cfg(not(target_arch = "bpf"))]
pub fn verify_signatures(
    program_id: &Pubkey,
    signature_acc: &Pubkey,
    payer: &Pubkey,
    guardian_set_id: u32,
    p: &VerifySigPayload,
) -> Result<Instruction, ProgramError> {
    let data = BridgeInstruction::VerifySignatures(*p).serialize()?;

    let bridge_key = Bridge::derive_bridge_id(program_id)?;
    let guardian_set_key =
        Bridge::derive_guardian_set_id(program_id, &bridge_key, guardian_set_id)?;

    let accounts = vec![
        AccountMeta::new_readonly(*program_id, false),
        AccountMeta::new_readonly(solana_program::system_program::id(), false),
        AccountMeta::new_readonly(solana_program::sysvar::instructions::id(), false),
        AccountMeta::new(bridge_key, false),
        AccountMeta::new(*signature_acc, false),
        AccountMeta::new_readonly(guardian_set_key, false),
        AccountMeta::new(*payer, true),
    ];

    Ok(Instruction {
        program_id: *program_id,
        accounts,
        data,
    })
}

/// Creates a 'PostVAA' instruction.
#[cfg(not(target_arch = "bpf"))]
pub fn post_vaa(
    program_id: &Pubkey,
    payer: &Pubkey,
    v: VAAData,
) -> Result<Instruction, ProgramError> {
    let mut data = v.clone();
    data.insert(0, 2);

    // Parse VAA
    let vaa = VAA::deserialize(&v[..])?;

    let bridge_key = Bridge::derive_bridge_id(program_id)?;
    let guardian_set_key =
        Bridge::derive_guardian_set_id(program_id, &bridge_key, vaa.guardian_set_index)?;
    let claim_key = Bridge::derive_claim_id(program_id, &bridge_key, vaa.signature_body()?)?;

    let signature_acc = Bridge::derive_signature_id(
        program_id,
        &bridge_key,
        &vaa.body_hash()?,
        vaa.guardian_set_index,
    )?;

    let mut accounts = vec![
        AccountMeta::new_readonly(*program_id, false),
        AccountMeta::new_readonly(solana_program::system_program::id(), false),
        AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
        AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
        AccountMeta::new(bridge_key, false),
        AccountMeta::new(guardian_set_key, false),
        AccountMeta::new(claim_key, false),
        AccountMeta::new(signature_acc, false),
        AccountMeta::new(*payer, true),
    ];

    match vaa.payload.unwrap() {
        VAABody::UpdateGuardianSet(u) => {
            let guardian_set_key =
                Bridge::derive_guardian_set_id(program_id, &bridge_key, u.new_index)?;
            accounts.push(AccountMeta::new(guardian_set_key, false));
        }
        VAABody::UpgradeContract(u) => {
            // Make program writeable
            accounts[0] = AccountMeta::new(*program_id, false);
            accounts.push(AccountMeta::new(u.buffer, false));
            let (programdata_address, _) = Pubkey::find_program_address(&[program_id.as_ref()], &solana_program::bpf_loader_upgradeable::id());
            accounts.push(AccountMeta::new(programdata_address, false));
            accounts.push(AccountMeta::new_readonly(solana_program::bpf_loader_upgradeable::id(), false));
        }
        VAABody::Message(t) => {
            let message_key = Bridge::derive_message_id(
                program_id,
                &bridge_key,
                t.emitter_chain,
                t.emitter_address,
                t.nonce,
                t.data,
            )?;
            accounts.push(AccountMeta::new(message_key, false))
        }
    }

    Ok(Instruction {
        program_id: *program_id,
        accounts,
        data,
    })
}

/// Unpacks a reference from a bytes buffer.
pub fn unpack<T>(input: &[u8]) -> Result<&T, ProgramError> {
    if input.len() < size_of::<u8>() + size_of::<T>() {
        return Err(ProgramError::InvalidInstructionData);
    }
    #[allow(clippy::cast_ptr_alignment)]
        let val: &T = unsafe { &*(&input[1] as *const u8 as *const T) };
    Ok(val)
}
