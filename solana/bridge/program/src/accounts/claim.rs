//! ClaimData accounts are one off markers that can be combined with other accounts to represent
//! data that can only be used once.

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
    processors::seeded::Seeded,
    AccountOwner,
    AccountState::{
        self,
        *,
    },
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

pub type Claim<'a, const S: AccountState> = Data<'a, ClaimData, { S }>;

/// Consume a claim by initializing the account. Initialized claims act as an indicator proving
/// that a message has been consumed.
pub fn consume<T>(
    ctx: &ExecutionContext,
    payer: &Pubkey,
    claim: &mut Claim<{ Uninitialized }>,
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

#[derive(Default, Clone, Copy, BorshDeserialize, BorshSerialize, Serialize, Deserialize)]
pub struct ClaimData {
    pub claimed: bool,
}

impl Owned for ClaimData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

pub struct ClaimDerivationData {
    pub emitter_address: [u8; 32],
    pub emitter_chain: u16,
    pub sequence: u64,
}

impl<'b, const State: AccountState> Seeded<&ClaimDerivationData> for Claim<'b, { State }> {
    fn seeds(data: &ClaimDerivationData) -> Vec<Vec<u8>> {
        return vec![
            data.emitter_address.to_vec(),
            data.emitter_chain.to_be_bytes().to_vec(),
            data.sequence.to_be_bytes().to_vec(),
        ];
    }
}
