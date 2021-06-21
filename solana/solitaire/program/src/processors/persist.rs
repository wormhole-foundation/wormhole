use solana_program::pubkey::Pubkey;

pub trait Persist {
    fn persist(&self, program_id: &Pubkey) -> crate::Result<()>;
}
