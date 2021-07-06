use crate::trace;
use solana_program::{
    account_info::{
        next_account_info,
        AccountInfo,
    },
    pubkey::Pubkey,
};
use std::slice::Iter;

/// The context is threaded through each check. Include anything within this structure that you
/// would like to have access to as each layer of dependency is peeled off.
pub struct Context<'a, 'b: 'a, 'c, T> {
    /// A reference to the program_id of the current program.
    pub this: &'a Pubkey,

    /// A reference to the instructions account list, one or more keys may be extracted during
    /// the peeling process.
    pub iter: &'c mut Iter<'a, AccountInfo<'b>>,

    /// Reference to the data passed to the current instruction.
    pub data: &'a T,

    /// An optional account info for this Peelable item, some fields may be other structures that
    /// do not themselves have an account info associated with the field.
    pub info: Option<&'a AccountInfo<'b>>,

    /// Whether to enforce immutability.
    pub immutable: bool,
}

impl<'a, 'b: 'a, 'c, T> Context<'a, 'b, 'c, T> {
    pub fn new(program: &'a Pubkey, iter: &'c mut Iter<'a, AccountInfo<'b>>, data: &'a T) -> Self {
        Context {
            this: program,
            info: None,
            immutable: true,
            iter,
            data,
        }
    }

    pub fn info<'d>(&'d mut self) -> &'a AccountInfo<'b> {
        match self.info {
            None => {
                let info = next_account_info(self.iter).unwrap();
                trace!("{}", info.key);
                self.info = Some(info);
                info
            }
            Some(v) => v,
        }
    }
}
