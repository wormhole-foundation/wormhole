

use borsh::{BorshDeserialize, BorshSerialize};

use solana_program::{msg, program_error::ProgramError, pubkey::Pubkey};
use solitaire::{
    processors::seeded::AccountOwner, AccountState, Context, Data, ExecutionContext, FromAccounts,
    Info, InstructionContext, Keyed, Owned, Peel, Result as SoliResult, Signer,
    ToInstruction,
};

use crate::{config::P2WConfigAccount, types::PriceAttestation};
use solana_program::{
    clock::Clock, instruction::Instruction, msg, program::invoke_signed,
    program_error::ProgramError, pubkey::Pubkey,
};
use solitaire::{
    processors::seeded::AccountOwner, trace, AccountState, CPICall, Context, Data,
    ExecutionContext, FromAccounts, Info, InstructionContext, Keyed, Owned, Peel,
    Result as SoliResult, Signer, SolitaireError, Sysvar, ToInstruction,
};

#[derive(FromAccounts, ToInstruction)]
pub struct Forward<'b> {
    pub payer: Signer<Info<'b>>,
    pub system_program: Signer<Info<'b>>,
    pub config: P2WConfigAccount<'b, {AccountState::Initialized}>,
    pub wormhole_program: Info<'b>,
    pub pyth_product: Info<'b>,
    pub pyth_price: Info<'b>,
    pub post_message_call: CPICall<PostMessage<'b>>,
}

#[derive(BorshDeserialize, BorshSerialize)]
pub struct ForwardData {
    pub target_chain: u32,
}

impl<'b> InstructionContext<'b> for Forward<'b> {
    fn verify(&self, _program_id: &Pubkey) -> SoliResult<()> {
        if self.config.wormhole_program_addr != *self.wormhole_program.key {
            trace!(&format!(
                "wormhole_program pubkey mismatch (expected {:?}",
                self.config.wormhole_program_addr
            ));
            return Err(ProgramError::InvalidAccountData.into());
        }
        if self.config.pyth_owner != *self.pyth_price.owner
            || self.config.pyth_owner != *self.pyth_product.owner
        {
            trace!(&format!(
                "pyth_owner pubkey mismatch (expected {:?}",
                self.config.pyth_owner
            ));
            return Err(SolitaireError::InvalidOwner(self.pyth_price.owner.clone()).into());
        }
        Ok(())
    }

    fn deps(&self) -> Vec<Pubkey> {
        vec![solana_program::system_program::id()]
    }
}

pub fn forward_price(
    _ctx: &ExecutionContext,
    accs: &mut Forward,
    _data: ForwardData,
) -> SoliResult<()> {
    let _price_attestation = PriceAttestation::from_bytes(&*accs.pyth_price.0.try_borrow_data()?)?;

    Ok(())
}
