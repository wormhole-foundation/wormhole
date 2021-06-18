#![allow(warnings)]

use borsh::BorshSerialize;
use solana_client::rpc_client::RpcClient;
use solana_program::{
    borsh::try_from_slice_unchecked,
    hash,
    instruction::{
        AccountMeta,
        Instruction,
    },
    program_pack::Pack,
    pubkey::Pubkey,
    system_instruction::{
        self,
        create_account,
    },
    system_program,
    sysvar,
};
use solana_sdk::{
    signature::{
        read_keypair_file,
        Keypair,
        Signer,
    },
    transaction::Transaction,
};
use std::time::{
    Duration,
    SystemTime,
};

use bridge::{
    accounts::GuardianSetDerivationData,
    instruction,
    types::{
        BridgeConfig,
        PostedMessage,
        SequenceTracker,
    },
    Initialize,
};

mod common;

#[test]
fn test_bridge_messages() {
    let (ref payer, ref client, ref program) = common::setup();

    // Initialize the Bridge
    common::initialize(
        client,
        program,
        payer,
        GuardianSetDerivationData { index: 0 },
    );
}
