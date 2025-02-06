pub mod close_signatures;
pub mod post_signatures;
pub mod verify_hash;

use base64::{prelude::BASE64_STANDARD, Engine};
use solana_program_test::{BanksClient, ProgramTest};
use solana_sdk::{
    hash::Hash, pubkey::Pubkey, signature::Keypair, signer::Signer, transaction::Transaction,
};
use wormhole_svm_definitions::{
    borsh::GuardianSignatures, find_guardian_set_address, CORE_BRIDGE_CONFIG,
    CORE_BRIDGE_PROGRAM_ID, GUARDIAN_SIGNATURE_LENGTH, VERIFY_VAA_SHIM_PROGRAM_ID,
};

pub async fn start_test(vaa: &str) -> (BanksClient, Keypair, Hash, DecodedVaa) {
    let mut program_test =
        ProgramTest::new("wormhole_verify_vaa_shim", VERIFY_VAA_SHIM_PROGRAM_ID, None);
    // program_test.add_program("core_bridge", CORE_BRIDGE_PROGRAM_ID, None);
    program_test.add_account_with_base64_data(
        CORE_BRIDGE_CONFIG,
        1_057_920,
        CORE_BRIDGE_PROGRAM_ID,
        "BAAAAAQYDQ0AAAAAgFEBAGQAAAAAAAAA",
    );
    // Guardian set 4 (active).
    program_test.add_account_with_base64_data(
        find_guardian_set_address(u32::to_be_bytes(4), &CORE_BRIDGE_PROGRAM_ID).0,
        3_647_040,
        CORE_BRIDGE_PROGRAM_ID,
        "BAAAABMAAABYk7WnbD9zlkVkiIW9zMBs1wo80/9suVJYm96GLCXvQ5ITL7nUpCFXEU3oRgGTvfOi/PgfhqCXZfR2L9EQegCGsy16CXeSaiBRMdhzHTnL64yCsv2C+u0nEdWa8PJJnRbnJvayEbOXVsBCRBvm2GULabVOvnFeI0NUzltNNI+3S5WOiWbi7D29SVinzRXnyvB8Tj3I58Rp+SyM2I+4AFogdKO/kTlT1pUmDYi8GqJaTu42PvAACsAHZyezX76i2sKP7lzLD+p2jq9FztE2udniSQNGSuiJ9cinI/wU+TEkt8c4hDy7iehkyGLDjN3Mz5XSzDek3ANqjSMrSPYs3UcxQS9IkNp5j2iWozMfZLSMEtHVf9nL5wgRcaob4dNsr+OGeRD5nAnjR4mcGcOBkrbnOHzNdoJ3wX2rG3pQJ8CzzxeOIa0ud64GcRVJz7sfnHqdgJboXhSH81UV0CqSdTUEqNdUcbn0nttvvryJj0A+R3PpX+sV6Ayamcg0jXiZHmYAAAAA",
    );
    // Guardian set 3 (expired).
    program_test.add_account_with_base64_data(
        find_guardian_set_address(u32::to_be_bytes(3), &CORE_BRIDGE_PROGRAM_ID).0,
        3_647_040,
        CORE_BRIDGE_PROGRAM_ID,
        "AwAAABMAAABYzDrlwJeyE848gZeeG5+VcHRqpf9suVJYm96GLCXvQ5ITL7nUpCFXEU3oRgGTvfOi/PgfhqCXZfR2L9EQegCGsy16CXeSaiBRMdhzHTnL64yCsv2C+u0nEdWa8PJJnRbnJvayEbOXVsBCRBvm2GULabVOvnFeI0NUzltNNI+3S5WOiWbi7D29SVinzRXnyvB8Tj3I58Rp+SyM2I+4AFogdKO/kTlT1pUmDYi8GqJaTu42PvAACsAHZyezX76i2sKP7lzLD+p2jq9FztE2udniSQNGSuiJ9cinI/wU+TEkt8c4hDy7iehkyGLDjN3Mz5XSzDek3ANqjSMrSPYs3UcxQS9IkNp5j2iWozMfZLSMEtHVf9nL5wgRcaob4dNsr+OGeRD5nAnjR4mcGcOBkrbnOHzNdoJ3wX2rG3pQJ8CzzxeOIa0ud64GcRVJz7sfnHqdgJboXhSH81UV0CqSdTUEqNdUcbn0nttvvryJj0A+R3PpX+sV6Ayamcg0jUA8xWP46h9m",
    );
    program_test.prefer_bpf(true);

    let (banks_client, payer, recent_blockhash) = program_test.start().await;
    (banks_client, payer, recent_blockhash, vaa.into())
}

pub struct DecodedVaa {
    pub guardian_set_index: u32,
    pub total_signatures: u8,
    pub guardian_signatures: Vec<[u8; GUARDIAN_SIGNATURE_LENGTH]>,
    pub body: Vec<u8>,
}

impl From<&str> for DecodedVaa {
    fn from(vaa: &str) -> Self {
        let mut buf = BASE64_STANDARD.decode(vaa).unwrap();
        let guardian_set_index = u32::from_be_bytes(buf[1..5].try_into().unwrap());
        let total_signatures = buf[5];

        let body = buf
            .drain((6 + total_signatures as usize * GUARDIAN_SIGNATURE_LENGTH)..)
            .collect();

        let mut guardian_signatures = Vec::with_capacity(total_signatures as usize);

        for i in 0..usize::from(total_signatures) {
            let offset = 6 + i * 66;
            let mut signature = [0; GUARDIAN_SIGNATURE_LENGTH];
            signature.copy_from_slice(&buf[offset..offset + GUARDIAN_SIGNATURE_LENGTH]);
            guardian_signatures.push(signature);
        }

        Self {
            guardian_set_index,
            total_signatures,
            guardian_signatures,
            body,
        }
    }
}

pub async fn transfer_lamports(
    banks_client: &BanksClient,
    recent_blockhash: Hash,
    payer: &Keypair,
    recipient: &Pubkey,
    lamports: u64,
) -> Hash {
    let transfer_ix =
        solana_sdk::system_instruction::transfer(&payer.pubkey(), recipient, lamports);

    banks_client
        .process_transaction(Transaction::new_signed_with_payer(
            &[transfer_ix],
            Some(&payer.pubkey()),
            &[payer],
            recent_blockhash,
        ))
        .await
        .unwrap();

    banks_client.get_latest_blockhash().await.unwrap()
}

pub fn generate_expected_guardian_signatures_info(
    payer: &Pubkey,
    total_signatures: u8,
    guardian_set_index: u32,
    guardian_signatures: Vec<[u8; GUARDIAN_SIGNATURE_LENGTH]>,
) -> (
    usize, // expected length
    GuardianSignatures,
) {
    let expected_length = {
        8 // discriminator
        + 32 // refund recipient
        + 4 // guardian set index
        + 4 // guardian signatures length
        + (total_signatures as usize) * GUARDIAN_SIGNATURE_LENGTH
    };

    let guardian_signatures = GuardianSignatures {
        refund_recipient: *payer,
        guardian_set_index_be: guardian_set_index.to_be_bytes(),
        guardian_signatures,
    };

    (expected_length, guardian_signatures)
}

pub fn bump_cu_cost(bump: u8) -> u64 {
    1_500 * (255 - u64::from(bump))
}

pub async fn send_post_signatures_transaction(
    banks_client: &BanksClient,
    tx_payer_signer: &Keypair,
    guardian_set_index: u32,
    total_signatures: u8,
    guardian_signatures: &[[u8; GUARDIAN_SIGNATURE_LENGTH]],
    recent_blockhash: Hash,
    guardian_signatures_signer: Option<&Keypair>,
) -> (
    Pubkey,  // guardian signatures
    Keypair, // refund recipient signer
    Hash,
) {
    let refund_recipient_signer = Keypair::new();

    let recent_blockhash = transfer_lamports(
        banks_client,
        recent_blockhash,
        tx_payer_signer,
        &refund_recipient_signer.pubkey(),
        2_000_000_000,
    )
    .await;

    let arbitrary_signer = Keypair::new();
    let guardian_signatures_signer = guardian_signatures_signer.unwrap_or(&arbitrary_signer);

    let transaction = post_signatures::set_up_transaction(
        tx_payer_signer,
        guardian_set_index,
        total_signatures,
        guardian_signatures,
        &refund_recipient_signer,
        guardian_signatures_signer,
        recent_blockhash,
        None,
    );

    banks_client.process_transaction(transaction).await.unwrap();

    (
        guardian_signatures_signer.pubkey(),
        refund_recipient_signer,
        banks_client.get_latest_blockhash().await.unwrap(),
    )
}
