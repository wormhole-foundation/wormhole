use crate::{accounts, Initialize, InitializeData};
use anchor_lang::{prelude::*, solana_program};

pub fn initialize(ctx: Context<Initialize>, data: InitializeData) -> ProgramResult {
    Ok(())
}
