use anchor_lang::prelude::*;

declare_id!("EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX");

#[program]
pub mod wormhole_post_message_shim {
    use super::*;

    /// This instruction is intended to be a significantly cheaper alternative
    /// to the post message instruction on Wormhole Core Bridge program. It
    /// achieves this by reusing the message account (per emitter) via the post
    /// message unreliable instruction and emitting data via self-CPI (Anchor
    /// event) for the guardian to observe. This instruction data contains
    /// information previously found only in the resulting message account.
    ///
    /// Because this instruction passes through the emitter and calls the post
    /// message unreliable instruction on the Wormhole Core Bridge, it can be
    /// used without disruption.
    ///
    /// NOTE: In the initial message publication for a new emitter, this will
    /// require one additional CPI call depth when compared to using the
    /// Wormhole Core Bridge directly. If this initial call depth is an issue,
    /// emit an empty message on initialization (or migration) in order to
    /// instantiate the message account. Posting a message will result in a VAA
    /// from your emitter, so be careful to avoid any issues that may result
    /// from this first message.
    ///
    /// Call depth of direct case:
    /// 1. post message (Wormhole Post Message Shim)
    /// 2. multiple CPI
    ///     - post message unreliable (Wormhole Core Bridge)
    ///     - Anchor event of `MesssageEvent` (Wormhole Post Message Shim)
    ///
    /// Call depth of integrator case:
    /// 1. integrator instruction
    /// 2. CPI post message (Wormhole Post Message Shim)
    /// 3. multiple CPI
    ///    - post message unreliable (Wormhole Core Bridge)
    ///    - Anchor event of `MesssageEvent` (Wormhole Post Message Shim)
    pub fn post_message(
        _ctx: Context<PostMessage>,
        nonce: u32,
        consistency_level: Finality,
        payload: Vec<u8>,
    ) -> Result<()> {
        let _ = (nonce, consistency_level, payload);
        err!(ErrorCode::InstructionMissing)
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, AnchorDeserialize, AnchorSerialize)]
pub enum Finality {
    Confirmed,
    Finalized,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
#[event]
pub struct MessageEvent {
    pub emitter: Pubkey,
    pub sequence: u64,
    pub submission_time: u32,
}

/// The accounts are ordered and named the same as the Wormhole Core Bridge
/// program's post message unreliable instruction.
#[derive(Accounts)]
#[event_cpi]
pub struct PostMessage<'info> {
    /// Wormhole Core Bridge config. The Wormhole Core Bridge program's post
    /// message instruction requires this account to be mutable.
    #[account(
        mut,
        seeds = [b"Bridge"],
        bump,
        seeds::program = wormhole_program.key()
    )]
    bridge: UncheckedAccount<'info>,

    /// Wormhole Message. The Wormhole Core Bridge program's post message
    /// instruction requires this account to be a mutable signer.
    ///
    /// This program uses a PDA per emitter. Messages are already bottle-necked
    /// by emitter sequence and the Wormhole Core Bridge program enforces that
    /// emitter must be identical for reused accounts. While this could be
    /// managed by the integrator, it seems more effective to have this Shim
    /// program manage these accounts.
    #[account(
        mut,
        seeds = [&emitter.key.to_bytes()],
        bump
    )]
    message: UncheckedAccount<'info>,

    /// Emitter of the Wormhole Core Bridge message. Wormhole Core Bridge
    /// program's post message instruction requires this account to be a signer.
    emitter: Signer<'info>,

    /// Emitter's sequence account. Wormhole Core Bridge program's post message
    /// instruction requires this account to be mutable.
    #[account(
        mut,
        seeds = [b"Sequence", &emitter.key.to_bytes()],
        bump,
        seeds::program = wormhole_program.key()
    )]
    sequence: UncheckedAccount<'info>,

    /// Payer will pay the rent for the Wormhole Core Bridge emitter sequence
    /// and message on the first post message call. Subsequent calls will not
    /// require more lamports for rent.
    #[account(mut)]
    payer: Signer<'info>,

    /// Wormhole Core Bridge fee collector. Wormhole Core Bridge program's post
    /// message instruction requires this account to be mutable.
    #[account(
        mut,
        seeds = [b"fee_collector"],
        bump,
        seeds::program = wormhole_program.key()
    )]
    fee_collector: UncheckedAccount<'info>,

    /// Clock sysvar.
    clock: Sysvar<'info, Clock>,

    /// System program.
    system_program: Program<'info, System>,

    /// Wormhole Core Bridge program.
    wormhole_program: UncheckedAccount<'info>,
}
