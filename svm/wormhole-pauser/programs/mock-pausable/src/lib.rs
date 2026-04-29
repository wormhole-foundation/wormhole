use anchor_lang::prelude::*;

declare_id!("5QWi9BQVwNv1dWpcSD8V8h65rNFemzkEYukGFj9Jm63Y");

/// Mock target used by WormholePauser tests to verify proposal execution. Mirrors the EVM
/// `MockPausable` contract: `pause` flips a boolean and records the caller (the WormholePauser
/// authority PDA), and `set_should_revert` lets a test force an execution failure.
#[program]
pub mod mock_pausable {
    use super::*;

    pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
        let state = &mut ctx.accounts.state;
        state.paused = false;
        state.should_revert = false;
        state.last_caller = Pubkey::default();
        Ok(())
    }

    pub fn pause(ctx: Context<Pause>) -> Result<()> {
        let state = &mut ctx.accounts.state;
        require!(!state.should_revert, MockPausableError::ForcedRevert);
        state.paused = true;
        state.last_caller = ctx.accounts.authority.key();
        Ok(())
    }

    pub fn set_should_revert(ctx: Context<SetShouldRevert>, value: bool) -> Result<()> {
        ctx.accounts.state.should_revert = value;
        Ok(())
    }
}

#[account]
pub struct State {
    pub paused: bool,
    pub should_revert: bool,
    pub last_caller: Pubkey,
}

impl State {
    pub const SEED_PREFIX: &'static [u8] = b"state";
    pub const SIZE: usize = 8 + 1 + 1 + 32;
}

#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(mut)]
    pub payer: Signer<'info>,

    #[account(
        init,
        payer = payer,
        space = State::SIZE,
        seeds = [State::SEED_PREFIX],
        bump,
    )]
    pub state: Account<'info, State>,

    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct Pause<'info> {
    /// The pauser's authority PDA. Verified only by the runtime via the CPI's `is_signer` flag,
    /// matching the EVM design where `msg.sender` is the WormholePauser contract.
    pub authority: Signer<'info>,

    #[account(
        mut,
        seeds = [State::SEED_PREFIX],
        bump,
    )]
    pub state: Account<'info, State>,
}

#[derive(Accounts)]
pub struct SetShouldRevert<'info> {
    pub payer: Signer<'info>,

    #[account(
        mut,
        seeds = [State::SEED_PREFIX],
        bump,
    )]
    pub state: Account<'info, State>,
}

#[error_code]
pub enum MockPausableError {
    #[msg("forced revert")]
    ForcedRevert,
}
