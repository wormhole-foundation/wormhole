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

    pub iter: &'c mut Iter<'a, AccountInfo<'b>>,

    /// This is a reference to the instruction data we are processing this
    /// account for.
    pub data: &'a T,

    pub info: Option<&'a AccountInfo<'b>>,
}

impl<'a, 'b: 'a, 'c, T> Context<'a, 'b, 'c, T> {
    pub fn new(program: &'a Pubkey, iter: &'c mut Iter<'a, AccountInfo<'b>>, data: &'a T) -> Self {
        Context {
            this: program,
            iter,
            data,
            info: None,
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
