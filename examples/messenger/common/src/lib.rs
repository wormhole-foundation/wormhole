use borsh::{
    BorshDeserialize,
    BorshSerialize,
};

#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq)]
pub struct Message {
    /// Messenger/DM username.
    pub nick: String,

    /// Message text to be output on the target networks node logs.
    pub text: String,
}
