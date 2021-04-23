use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct VerifySig<'info> {
    bridge: AccountInfo<'info>,
    system: AccountInfo<'info>,
    instruction_sysvar: AccountInfo<'info>,
    bridge_info: AccountInfo<'info>,
    sig_info: AccountInfo<'info>,
    guardian_set_info: AccountInfo<'info>,
    payer_info: AccountInfo<'info>,
}

pub const MAX_LEN_GUARDIAN_KEYS: usize = 20;

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, Debug)]
pub struct VerifySigsData {
    /// hash of the VAA
    pub hash: [u8; 32],
    /// instruction indices of signers (-1 for missing)
    pub signers: [i8; MAX_LEN_GUARDIAN_KEYS],
    /// indicates whether this verification should only succeed if the sig account does not exist
    pub initial_creation: bool,
}

#[program]
pub mod anchor_bridge {
    use super::*;

    pub fn verify_signatures(_ctx: Context<VerifySig>, _data: VerifySigsData) -> ProgramResult {
        Ok(())
    }
}
