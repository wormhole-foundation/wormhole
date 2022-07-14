use borsh::{
    BorshDeserialize,
};
use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use libsecp256k1::{
    PublicKey,
    SecretKey,
};
use sha3::Digest;
use solana_program::{
    instruction::{
        Instruction,
    },
    program_pack::Pack,
    pubkey::Pubkey,
    system_instruction,
};
use solana_program_test::{
    BanksClient,
    ProgramTest,
};
use solana_sdk::{
    commitment_config::CommitmentLevel,
    rent::Rent,
    secp256k1_instruction::new_secp256k1_instruction,
    signature::{
        Keypair,
        Signer,
    },
    signers::Signers,
    transaction::Transaction,
    transport::TransportError,
};
use std::{
    env,
    io::{
        Cursor,
        Write,
    },
    time::SystemTime,
};

use token_bridge::{
    instructions,
    types::*,
};

use solitaire::{
    processors::seeded::Seeded,
};

pub use helpers::*;

/// Simple API wrapper for quickly preparing and sending transactions.
pub async fn execute<T: Signers>(
    client: &mut BanksClient,
    payer: &Keypair,
    signers: &T,
    instructions: &[Instruction],
    commitment_level: CommitmentLevel,
) -> Result<(), TransportError> {
    let mut transaction = Transaction::new_with_payer(instructions, Some(&payer.pubkey()));
    let recent_blockhash = client.get_latest_blockhash().await?;
    transaction.sign(signers, recent_blockhash);
    client
        .process_transaction_with_commitment(transaction, commitment_level)
        .await
}

mod helpers {
    use super::*;
    use bridge::{
        accounts::{
            FeeCollector,
        },
        types::ConsistencyLevel,
        PostVAAData,
    };
    use solana_program_test::processor;
    use token_bridge::{
        CompleteNativeData,
        CompleteWrappedData,
        CompleteNativeWithPayloadData,
        CreateWrappedData,
        RegisterChainData,
        TransferNativeData,
        TransferWrappedData,
    };

    use token_bridge::messages::{
        PayloadAssetMeta,
        PayloadGovernanceRegisterChain,
        PayloadTransfer,
        PayloadTransferWithPayload
    };

    /// Generate `count` secp256k1 private keys, along with their ethereum-styled public key
    /// encoding: 0x0123456789ABCDEF01234
    pub fn generate_keys(count: u8) -> (Vec<[u8; 20]>, Vec<SecretKey>) {

        let mut rng = rand::thread_rng();

        // Generate Guardian Keys
        let secret_keys: Vec<SecretKey> = std::iter::repeat_with(|| SecretKey::random(&mut rng))
            .take(count as usize)
            .collect();

        (
            secret_keys
                .iter()
                .map(|key| {
                    let public_key = PublicKey::from_secret_key(key);
                    let mut h = sha3::Keccak256::default();
                    h.write_all(&public_key.serialize()[1..]).unwrap();
                    let key: [u8; 32] = h.finalize().into();
                    let mut address = [0u8; 20];
                    address.copy_from_slice(&key[12..]);
                    address
                })
                .collect(),
            secret_keys,
        )
    }

    /// Initialize the test environment, spins up a solana-test-validator in the background so that
    /// each test has a fresh environment to work within.
    pub async fn setup() -> (BanksClient, Keypair, Pubkey, Pubkey) {
        let (program, token_program) = (
            env::var("BRIDGE_PROGRAM")
                .unwrap_or_else(|_| "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o".to_string())
                .parse::<Pubkey>()
                .unwrap(),
            env::var("TOKEN_BRIDGE_PROGRAM")
                .unwrap_or_else(|_| "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE".to_string())
                .parse::<Pubkey>()
                .unwrap(),
        );

        let mut builder = ProgramTest::new("bridge", program, processor!(bridge::solitaire));
        builder.add_program(
            "spl_token_metadata",
            spl_token_metadata::id(),
            None,
        );
        builder.add_program(
            "token_bridge",
            token_program,
            processor!(token_bridge::solitaire),
        );

        // Some instructions go over the limit when tracing is enabled but we need that for better
        // logging.  We don't really care about the limit during these tests anyway.
        builder.set_compute_max_units(u64::MAX);

        let (client, payer, _) = builder.start().await;
        (client, payer, program, token_program)
    }

    /// Wait for a single transaction to fully finalize, guaranteeing chain state has been
    /// confirmed. Useful for consistently fetching data during state checks.
    #[allow(dead_code)]
    pub async fn sync(client: &mut BanksClient, payer: &Keypair) {
        let payer_key = payer.pubkey();
        execute(
            client,
            payer,
            &[payer],
            &[system_instruction::transfer(&payer_key, &payer_key, 1)],
            CommitmentLevel::Finalized,
        )
        .await
        .unwrap();
    }

    /// Fetch account data, the loop is there to re-attempt until data is available.
    pub async fn get_account_data<T: BorshDeserialize>(
        client: &mut BanksClient,
        account: Pubkey,
    ) -> Option<T> {
        let account = client
            .get_account_with_commitment(account, CommitmentLevel::Processed)
            .await
            .unwrap()
            .unwrap();
        T::try_from_slice(&account.data).ok()
    }

    /// Fetch account balance
    #[allow(dead_code)]
    pub async fn get_account_balance(client: &mut BanksClient, account: Pubkey) -> u64 {
        client.get_account(account).await.unwrap().unwrap().lamports
    }

    pub async fn initialize_bridge(
        client: &mut BanksClient,
        program: Pubkey,
        payer: &Keypair,
        initial_guardians: &[[u8; 20]],
    ) -> Result<(), TransportError> {
        execute(
            client,
            payer,
            &[payer],
            &[bridge::instructions::initialize(
                program,
                payer.pubkey(),
                50,
                2_000_000_000,
                initial_guardians,
            )
            .unwrap()],
            CommitmentLevel::Processed,
        )
        .await
    }

    #[allow(dead_code)]
    pub async fn transfer(
        client: &mut BanksClient,
        from: &Keypair,
        to: &Pubkey,
        lamports: u64,
    ) -> Result<(), TransportError> {
        execute(
            client,
            from,
            &[from],
            &[system_instruction::transfer(&from.pubkey(), to, lamports)],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn initialize(
        client: &mut BanksClient,
        program: Pubkey,
        payer: &Keypair,
        bridge: Pubkey,
    ) -> Result<(), TransportError> {
        let instruction = instructions::initialize(program, payer.pubkey(), bridge)
            .expect("Could not create Initialize instruction");

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer],
            &[instruction],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn attest(
        client: &mut BanksClient,
        program: Pubkey,
        bridge: Pubkey,
        payer: &Keypair,
        message: &Keypair,
        mint: Pubkey,
        nonce: u32,
    ) -> Result<(), TransportError> {
        let instruction = instructions::attest(
            program,
            bridge,
            payer.pubkey(),
            message.pubkey(),
            mint,
            nonce,
        )
        .expect("Could not create Attest instruction");

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer, message],
            &[instruction],
            CommitmentLevel::Processed,
        )
        .await
    }

    #[allow(clippy::too_many_arguments)]
    pub async fn transfer_native(
        client: &mut BanksClient,
        program: Pubkey,
        bridge: Pubkey,
        payer: &Keypair,
        message: &Keypair,
        from: &Keypair,
        from_owner: &Keypair,
        mint: Pubkey,
        amount: u64,
    ) -> Result<(), TransportError> {
        let instruction = instructions::transfer_native(
            program,
            bridge,
            payer.pubkey(),
            message.pubkey(),
            from.pubkey(),
            mint,
            TransferNativeData {
                nonce: 0,
                amount,
                fee: 0,
                target_address: [0u8; 32],
                target_chain: 2,
            },
        )
        .expect("Could not create Transfer Native");

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer, from_owner, message],
            &[
                spl_token::instruction::approve(
                    &spl_token::id(),
                    &from.pubkey(),
                    &token_bridge::accounts::AuthoritySigner::key(None, &program),
                    &from_owner.pubkey(),
                    &[],
                    amount,
                )
                .unwrap(),
                instruction,
            ],
            CommitmentLevel::Processed,
        )
        .await
    }

    #[allow(clippy::too_many_arguments)]
    pub async fn transfer_wrapped(
        client: &mut BanksClient,
        program: Pubkey,
        bridge: Pubkey,
        payer: &Keypair,
        message: &Keypair,
        from: Pubkey,
        from_owner: &Keypair,
        token_chain: u16,
        token_address: Address,
        amount: u64,
    ) -> Result<(), TransportError> {
        let instruction = instructions::transfer_wrapped(
            program,
            bridge,
            payer.pubkey(),
            message.pubkey(),
            from,
            from_owner.pubkey(),
            token_chain,
            token_address,
            TransferWrappedData {
                nonce: 0,
                amount,
                fee: 0,
                target_address: [5u8; 32],
                target_chain: 2,
            },
        )
        .expect("Could not create Transfer Native");

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer, from_owner, message],
            &[
                spl_token::instruction::approve(
                    &spl_token::id(),
                    &from,
                    &token_bridge::accounts::AuthoritySigner::key(None, &program),
                    &from_owner.pubkey(),
                    &[],
                    amount,
                )
                .unwrap(),
                instruction,
            ],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn register_chain(
        client: &mut BanksClient,
        program: Pubkey,
        bridge: Pubkey,
        message_acc: Pubkey,
        vaa: PostVAAData,
        payload: PayloadGovernanceRegisterChain,
        payer: &Keypair,
    ) -> Result<(), TransportError> {
        let instruction = instructions::register_chain(
            program,
            bridge,
            payer.pubkey(),
            message_acc,
            vaa,
            payload,
            RegisterChainData {},
        )
        .expect("Could not create Transfer Native");

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer],
            &[instruction],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn complete_native(
        client: &mut BanksClient,
        program: Pubkey,
        bridge: Pubkey,
        message_acc: Pubkey,
        vaa: PostVAAData,
        payload: PayloadTransfer,
        payer: &Keypair,
    ) -> Result<(), TransportError> {
        let instruction = instructions::complete_native(
            program,
            bridge,
            payer.pubkey(),
            message_acc,
            vaa,
            Pubkey::new(&payload.to[..]),
            None,
            Pubkey::new(&payload.token_address[..]),
            CompleteNativeData {},
        )
        .expect("Could not create Complete Native instruction");

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer],
            &[instruction],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn complete_transfer_wrapped(
        client: &mut BanksClient,
        program: Pubkey,
        bridge: Pubkey,
        message_acc: Pubkey,
        vaa: PostVAAData,
        payload: PayloadTransfer,
        payer: &Keypair,
    ) -> Result<(), TransportError> {
        let to = Pubkey::new(&payload.to[..]);

        let instruction = instructions::complete_wrapped(
            program,
            bridge,
            payer.pubkey(),
            message_acc,
            vaa,
            payload,
            to,
            None,
            CompleteWrappedData {},
        )
        .expect("Could not create Complete Wrapped instruction");

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer],
            &[instruction],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn complete_native_with_payload(
        client: &mut BanksClient,
        program: Pubkey,
        bridge: Pubkey,
        message_acc: Pubkey,
        vaa: PostVAAData,
        payload: PayloadTransferWithPayload,
        to: Pubkey,
        redeemer: &Keypair,
        payer: &Keypair,
    ) -> Result<(), TransportError> {
        let instruction = instructions::complete_native_with_payload(
            program,
            bridge,
            payer.pubkey(),
            message_acc,
            vaa,
            to,
            redeemer.pubkey(),
            None,
            Pubkey::new(&payload.token_address[..]),
            CompleteNativeWithPayloadData {},
        )
        .expect("Could not create Complete Native With Payload instruction");
        
        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }
        
        execute(
            client,
            payer,
            &[payer, redeemer],
            &[instruction],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn create_wrapped(
        client: &mut BanksClient,
        program: Pubkey,
        bridge: Pubkey,
        message_acc: Pubkey,
        vaa: PostVAAData,
        payload: PayloadAssetMeta,
        payer: &Keypair,
    ) -> Result<(), TransportError> {
        let instruction = instructions::create_wrapped(
            program,
            bridge,
            payer.pubkey(),
            message_acc,
            vaa,
            payload,
            CreateWrappedData {},
        )
        .expect("Could not create Create Wrapped instruction");

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer],
            &[instruction],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn create_mint(
        client: &mut BanksClient,
        payer: &Keypair,
        mint_authority: &Pubkey,
        mint: &Keypair,
    ) -> Result<(), TransportError> {
        let mint_key = mint.pubkey();
        execute(
            client,
            payer,
            &[payer, mint],
            &[
                solana_sdk::system_instruction::create_account(
                    &payer.pubkey(),
                    &mint_key,
                    Rent::default().minimum_balance(spl_token::state::Mint::LEN),
                    spl_token::state::Mint::LEN as u64,
                    &spl_token::id(),
                ),
                spl_token::instruction::initialize_mint(
                    &spl_token::id(),
                    &mint_key,
                    mint_authority,
                    None,
                    0,
                )
                .unwrap(),
            ],
            CommitmentLevel::Processed,
        )
        .await
    }

    #[allow(clippy::too_many_arguments)]
    pub async fn create_spl_metadata(
        client: &mut BanksClient,
        payer: &Keypair,
        metadata_account: Pubkey,
        mint_authority: &Keypair,
        mint: &Keypair,
        update_authority: Pubkey,
        name: String,
        symbol: String,
    ) -> Result<(), TransportError> {
        execute(
            client,
            payer,
            &[payer, mint_authority],
            &[spl_token_metadata::instruction::create_metadata_accounts(
                spl_token_metadata::id(),
                metadata_account,
                mint.pubkey(),
                mint_authority.pubkey(),
                payer.pubkey(),
                update_authority,
                name,
                symbol,
                "https://token.org".to_string(),
                None,
                0,
                false,
                false,
            )],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn create_token_account(
        client: &mut BanksClient,
        payer: &Keypair,
        token_acc: &Keypair,
        token_authority: &Pubkey,
        mint: &Pubkey,
    ) -> Result<(), TransportError> {
        let token_key = token_acc.pubkey();
        execute(
            client,
            payer,
            &[payer, token_acc],
            &[
                solana_sdk::system_instruction::create_account(
                    &payer.pubkey(),
                    &token_key,
                    Rent::default().minimum_balance(spl_token::state::Account::LEN),
                    spl_token::state::Account::LEN as u64,
                    &spl_token::id(),
                ),
                spl_token::instruction::initialize_account(
                    &spl_token::id(),
                    &token_key,
                    mint,
                    token_authority,
                )
                .unwrap(),
            ],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn mint_tokens(
        client: &mut BanksClient,
        payer: &Keypair,
        mint_authority: &Keypair,
        mint: &Keypair,
        token_account: &Pubkey,
        amount: u64,
    ) -> Result<(), TransportError> {
        execute(
            client,
            payer,
            &[payer, mint_authority],
            &[spl_token::instruction::mint_to(
                &spl_token::id(),
                &mint.pubkey(),
                token_account,
                &mint_authority.pubkey(),
                &[],
                amount,
            )
            .unwrap()],
            CommitmentLevel::Processed,
        )
        .await
    }

    /// Utility function for generating VAA's from message data.
    pub fn generate_vaa<T: Into<Vec<u8>>>(
        emitter: Address,
        emitter_chain: u16,
        data: T,
        nonce: u32,
        sequence: u64,
    ) -> (PostVAAData, [u8; 32], [u8; 32]) {
        let vaa = PostVAAData {
            version: 0,
            guardian_set_index: 0,

            // Body part
            emitter_chain,
            emitter_address: emitter,
            sequence,
            payload: data.into(),
            timestamp: SystemTime::now()
                .duration_since(SystemTime::UNIX_EPOCH)
                .unwrap()
                .as_secs() as u32,
            nonce,
            consistency_level: ConsistencyLevel::Confirmed as u8,
        };

        // Hash data, the thing we wish to actually sign.
        let body = {
            let mut v = Cursor::new(Vec::new());
            v.write_u32::<BigEndian>(vaa.timestamp).unwrap();
            v.write_u32::<BigEndian>(vaa.nonce).unwrap();
            v.write_u16::<BigEndian>(vaa.emitter_chain).unwrap();
            v.write_all(&vaa.emitter_address).unwrap();
            v.write_u64::<BigEndian>(vaa.sequence).unwrap();
            v.write_u8(vaa.consistency_level).unwrap();
            v.write_all(&vaa.payload).unwrap();
            v.into_inner()
        };

        // Hash this body, which is expected to be the same as the hash currently stored in the
        // signature account, binding that set of signatures to this VAA.
        let body: [u8; 32] = {
            let mut h = sha3::Keccak256::default();
            h.write_all(body.as_slice()).unwrap();
            h.finalize().into()
        };

        let body_hash: [u8; 32] = {
            let mut h = sha3::Keccak256::default();
            h.write_all(&body).unwrap();
            h.finalize().into()
        };

        (vaa, body, body_hash)
    }

    pub async fn verify_signatures(
        client: &mut BanksClient,
        program: &Pubkey,
        payer: &Keypair,
        body: [u8; 32],
        secret_keys: &[SecretKey],
        guardian_set_version: u32,
    ) -> Result<Pubkey, TransportError> {
        let signature_set = Keypair::new();
        let tx_signers = [payer, &signature_set];
        // Push Secp256k1 instructions for each signature we want to verify.
        for (i, key) in secret_keys.iter().enumerate() {
            // Set this signers signature position as present at 0.
            let mut signers = [-1; 19];
            signers[i] = 0;

            execute(
                client,
                payer,
                &tx_signers,
                &[
                    new_secp256k1_instruction(key, &body),
                    bridge::instructions::verify_signatures(
                        *program,
                        payer.pubkey(),
                        guardian_set_version,
                        signature_set.pubkey(),
                        bridge::VerifySignaturesData { signers },
                    )
                    .unwrap(),
                ],
                CommitmentLevel::Processed,
            )
            .await?;
        }

        Ok(signature_set.pubkey())
    }

    pub async fn post_vaa(
        client: &mut BanksClient,
        program: Pubkey,
        payer: &Keypair,
        signature_set: Pubkey,
        vaa: PostVAAData,
    ) -> Result<(), TransportError> {
        let instruction =
            bridge::instructions::post_vaa(program, payer.pubkey(), signature_set, vaa);

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer],
            &[instruction],
            CommitmentLevel::Processed,
        )
        .await
    }

    #[allow(clippy::too_many_arguments)]
    #[allow(dead_code)]
    pub async fn post_message(
        client: &mut BanksClient,
        program: Pubkey,
        payer: &Keypair,
        emitter: &Keypair,
        message: &Keypair,
        nonce: u32,
        data: Vec<u8>,
        fee: u64,
    ) -> Result<(), TransportError> {
        // Transfer money into the fee collector as it needs a balance/must exist.
        let fee_collector = FeeCollector::<'_>::key(None, &program);

        // Capture the resulting message, later functions will need this.
        let instruction = bridge::instructions::post_message(
            program,
            payer.pubkey(),
            emitter.pubkey(),
            message.pubkey(),
            nonce,
            data,
            ConsistencyLevel::Confirmed,
        )
        .unwrap();

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer, emitter, message],
            &[
                system_instruction::transfer(&payer.pubkey(), &fee_collector, fee),
                instruction,
            ],
            CommitmentLevel::Processed,
        )
        .await
    }
}
