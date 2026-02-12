//! Claim accounts add replay protection to Messages.
//!
//! Claim accounts work by having the constraint that they must be uninitialized. This means once a
//! Claim account has been created, they can no longer be passed to an instruction. This gives us
//! the behaviour of replay protection.
//!
//! Example usage:
//!
//! ```rust,noplayground,no_run
//! struct ExampleAccounts {
//!     message: PayloadMessage<'info, Example>,
//!     claim:   Claim<'info>,
//!     payer:   Mut<Signer<'info>>,
//! }
//!
//! // Note that as a Claim must be uninitialized, only the first time this instruction is called
//! // will succeed, subsequent calls will fail the `Uninitialized` check.
//! fn read_message(
//!    ctx:  &ExecutionContext,
//!    accs: &mut ExampleAccounts,
//!    data: (),
//! ) {
//!    claim::consume(ctx, &accs.payer.key, &mut accs.claim, &accs.vaa)?;
//! }
//! ```

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use serde::{
    Deserialize,
    Serialize,
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::{
        Seeded,
        SingleOwned,
    },
    AccountOwner,
    AccountState::*,
    CreationLamports::Exempt,
    Data,
    Owned,
    Result,
    *,
};

use crate::{
    DeserializePayload,
    PayloadMessage,
};

pub type Claim<'a> = Data<'a, ClaimData, { Uninitialized }>;

/// Consume a claim by initializing the account. Initialized claims act as an indicator proving
/// that a message has been consumed.
pub fn consume<T>(
    ctx: &ExecutionContext,
    payer: &Pubkey,
    claim: &mut Claim,
    message: &PayloadMessage<T>,
) -> Result<()>
where
    T: DeserializePayload,
{
    // Verify that the claim account is derived correctly before claiming.
    claim.verify_derivation(
        ctx.program_id,
        &ClaimDerivationData {
            emitter_address: message.meta().emitter_address,
            emitter_chain: message.meta().emitter_chain,
            sequence: message.meta().sequence,
        },
    )?;

    // Claim the account by initializing it with a value.
    claim.create(
        &ClaimDerivationData {
            emitter_address: message.meta().emitter_address,
            emitter_chain: message.meta().emitter_chain,
            sequence: message.meta().sequence,
        },
        ctx,
        payer,
        Exempt,
    )?;

    claim.claimed = true;

    Ok(())
}

/// Account discriminator for ClaimData — prevents type confusion attacks.
pub const CLAIM_DISCRIMINATOR: &[u8] = b"clam";

#[derive(Default, Clone, Copy, Serialize, Deserialize)]
pub struct ClaimData {
    pub claimed: bool,
}

impl BorshSerialize for ClaimData {
    fn serialize<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        writer.write_all(CLAIM_DISCRIMINATOR)?;
        BorshSerialize::serialize(&self.claimed, writer)
    }
}

impl BorshDeserialize for ClaimData {
    fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
        use std::io::{Error, ErrorKind::InvalidData};
        if buf.len() < CLAIM_DISCRIMINATOR.len() {
            return Err(Error::new(InvalidData, "Not enough bytes for ClaimData discriminator"));
        }
        let magic = &buf[..CLAIM_DISCRIMINATOR.len()];
        if magic != CLAIM_DISCRIMINATOR {
            return Err(Error::new(InvalidData, format!("ClaimData discriminator mismatch. Expected {:?} but got {:?}", CLAIM_DISCRIMINATOR, magic)));
        }
        *buf = &buf[CLAIM_DISCRIMINATOR.len()..];
        Ok(ClaimData {
            claimed: BorshDeserialize::deserialize(buf)?,
        })
    }
}

impl Owned for ClaimData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

impl SingleOwned for ClaimData {
}

pub struct ClaimDerivationData {
    pub emitter_address: [u8; 32],
    pub emitter_chain: u16,
    pub sequence: u64,
}

impl<'b> Seeded<&ClaimDerivationData> for Claim<'b> {
    fn seeds(data: &ClaimDerivationData) -> Vec<Vec<u8>> {
        return vec![
            data.emitter_address.to_vec(),
            data.emitter_chain.to_be_bytes().to_vec(),
            data.sequence.to_be_bytes().to_vec(),
        ];
    }
}
