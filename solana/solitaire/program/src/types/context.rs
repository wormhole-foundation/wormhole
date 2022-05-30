use solana_program::{
    account_info::AccountInfo,
    pubkey::Pubkey,
};

/// The context is threaded through each check. Include anything within this structure that you
/// would like to have access to as each layer of dependency is peeled off.
pub struct Context<'a, 'b: 'a, T> {
    /// A reference to the program_id of the current program.
    pub this: &'a Pubkey,

    /// This is a reference to the AccountInfo of the field that is currently being parsed.
    pub info: &'a AccountInfo<'b>,

    /// Reference to the data passed to the current instruction.
    pub data: &'a T,

    /// Whether to enforce immutability.
    pub immutable: bool,
}

impl<'a, 'b: 'a, T> Context<'a, 'b, T> {
    pub fn new(this: &'a Pubkey, info: &'a AccountInfo<'b>, data: &'a T) -> Self {
        Context {
            immutable: true,
            this,
            info,
            data,
        }
    }
}
