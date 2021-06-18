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
use std::env;

use solana_sdk::{
    signature::{
        read_keypair_file,
        Keypair,
        Signer,
    },
    transaction::Transaction,
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
    PostMessageData,
    PostVAAData,
};

pub use helpers::*;
pub use instructions::*;

mod helpers {
    use super::*;

    pub fn setup() -> (Keypair, RpcClient, Pubkey) {
        let payer =
            read_keypair_file(env::var("BRIDGE_PAYER").unwrap_or("./payer.json".to_string()))
                .unwrap();
        let rpc =
            RpcClient::new(env::var("BRIDGE_RPC").unwrap_or("http://127.0.0.1:8899".to_string()));
        let program = env::var("BRIDGE_PROGRAM")
            .unwrap_or("6mFKdAtUBVbsQ5dgvBrUkn1Pixb7BMTUtVKj4dpwrmQs".to_string())
            .parse::<Pubkey>()
            .unwrap();

        (payer, rpc, program)
    }

    pub fn initialize(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        guardian_set: GuardianSetDerivationData,
    ) {
        let index = guardian_set.index.to_be_bytes();
        let index = index.as_ref();
        let (bridge, _) = Pubkey::find_program_address(&["Bridge".as_ref()], program);
        let (guardian_set, _) =
            Pubkey::find_program_address(&["GuardianSet".as_ref(), index], program);

        let signers = vec![payer];
        let instructions = [instructions::create_initialize(
            *program,
            payer.pubkey(),
            bridge,
            guardian_set,
        )];

        let mut transaction = Transaction::new_with_payer(&instructions, Some(&payer.pubkey()));
        let recent_blockhash = client.get_recent_blockhash().unwrap().0;

        transaction.sign(&signers, recent_blockhash);
        client.send_and_confirm_transaction(&transaction).unwrap();
    }
}

mod instructions {
    use super::*;

    pub fn create_initialize(
        program_id: Pubkey,
        payer: Pubkey,
        bridge: Pubkey,
        guardian_set: Pubkey,
    ) -> Instruction {
        Instruction {
            program_id,

            accounts: vec![
                AccountMeta::new(bridge, false),
                AccountMeta::new(guardian_set, false),
                AccountMeta::new(payer, true),
                AccountMeta::new_readonly(sysvar::rent::id(), false),
                AccountMeta::new_readonly(solana_program::system_program::id(), false),
            ],

            data: instruction::Instruction::Initialize(BridgeConfig {
                fee: 0,
                guardian_set_expiration_time: 0,
            })
            .try_to_vec()
            .unwrap(),
        }
    }
}
