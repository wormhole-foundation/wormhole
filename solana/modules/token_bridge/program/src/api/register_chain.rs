use crate::{
    accounts::{
        ConfigAccount,
        Endpoint,
        EndpointDerivationData,
    },
    messages::PayloadGovernanceRegisterChain,
    types::*,
};
use bridge::vaa::{
    ClaimableVAA,
    DeserializePayload,
    PayloadMessage,
};
use solana_program::{
    account_info::AccountInfo,
    program_error::ProgramError,
    pubkey::Pubkey,
};
use solitaire::{
    processors::seeded::Seeded,
    CreationLamports::Exempt,
    *,
};
use std::ops::{
    Deref,
    DerefMut,
};

#[derive(FromAccounts)]
pub struct RegisterChain<'b> {
    pub payer: Signer<AccountInfo<'b>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub endpoint: Endpoint<'b, { AccountState::Uninitialized }>,

    pub vaa: ClaimableVAA<'b, PayloadGovernanceRegisterChain>,
}

impl<'a> From<&RegisterChain<'a>> for EndpointDerivationData {
    fn from(accs: &RegisterChain<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'b> InstructionContext<'b> for RegisterChain<'b> {
    fn verify(&self, program_id: &Pubkey) -> Result<()> {
        self.endpoint.verify_derivation(program_id, &self.into())?;
        Ok(())
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct RegisterChainData {}

pub fn register_chain(
    ctx: &ExecutionContext,
    accs: &mut RegisterChain,
    data: RegisterChainData,
) -> Result<()> {
    // Claim VAA
    accs.vaa.claim(ctx, accs.payer.key)?;

    // Create endpoint
    accs.endpoint
        .create(&((&*accs).into()), ctx, accs.payer.key, Exempt);

    accs.endpoint.chain = accs.vaa.chain;
    accs.endpoint.contract = accs.vaa.endpoint_address;

    Ok(())
}
