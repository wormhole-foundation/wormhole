#![allow(warnings)]

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use hex_literal::hex;
use secp256k1::{
    Message as Secp256k1Message,
    PublicKey,
    SecretKey,
};
use sha3::Digest;
use solana_client::{
    client_error::ClientError,
    rpc_client::RpcClient,
    rpc_config::RpcSendTransactionConfig,
};
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
    commitment_config::CommitmentConfig,
    rent::Rent,
    secp256k1_instruction::new_secp256k1_instruction,
    signature::{
        read_keypair_file,
        Keypair,
        Signature,
        Signer,
    },
    transaction::Transaction,
};
use spl_token::state::Mint;
use std::{
    convert::TryInto,
    env,
    io::{
        Cursor,
        Write,
    },
    time::{
        Duration,
        SystemTime,
    },
};

use token_bridge::{
    accounts::*,
    instruction,
    instructions,
    types::*,
    Initialize,
};

use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};

pub use helpers::*;

/// Simple API wrapper for quickly preparing and sending transactions.
pub fn execute(
    client: &RpcClient,
    payer: &Keypair,
    signers: &[&Keypair],
    instructions: &[Instruction],
    commitment_level: CommitmentConfig,
) -> Result<Signature, ClientError> {
    let mut transaction = Transaction::new_with_payer(instructions, Some(&payer.pubkey()));
    let recent_blockhash = client.get_recent_blockhash().unwrap().0;
    transaction.sign(&signers.to_vec(), recent_blockhash);
    client.send_and_confirm_transaction_with_spinner_and_config(
        &transaction,
        commitment_level,
        RpcSendTransactionConfig {
            skip_preflight: true,
            preflight_commitment: None,
            encoding: None,
        },
    )
}

mod helpers {
    use bridge::types::{
        ConsistencyLevel,
        PostedVAAData,
    };
    use token_bridge::{
        CompleteNativeData,
        CompleteWrappedData,
        CreateWrappedData,
        RegisterChainData,
        TransferNativeData,
        TransferWrappedData,
    };

    use super::*;
    use bridge::{
        accounts::{
            FeeCollector,
            PostedVAADerivationData,
        },
        PostVAAData,
    };
    use std::ops::Add;
    use token_bridge::messages::{
        PayloadAssetMeta,
        PayloadGovernanceRegisterChain,
        PayloadTransfer,
    };

    /// Initialize the test environment, spins up a solana-test-validator in the background so that
    /// each test has a fresh environment to work within.
    pub fn setup() -> (Keypair, RpcClient, Pubkey, Pubkey) {
        let payer = env::var("BRIDGE_PAYER").unwrap_or("./payer.json".to_string());
        let rpc_address = env::var("BRIDGE_RPC").unwrap_or("http://127.0.0.1:8899".to_string());
        let payer = read_keypair_file(payer).unwrap();
        let rpc = RpcClient::new(rpc_address);

        let (program, token_program) = (
            env::var("BRIDGE_PROGRAM")
                .unwrap_or("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o".to_string())
                .parse::<Pubkey>()
                .unwrap(),
            env::var("TOKEN_BRIDGE_PROGRAM")
                .unwrap_or("B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE".to_string())
                .parse::<Pubkey>()
                .unwrap(),
        );

        (payer, rpc, program, token_program)
    }

    /// Wait for a single transaction to fully finalize, guaranteeing chain state has been
    /// confirmed. Useful for consistently fetching data during state checks.
    pub fn sync(client: &RpcClient, payer: &Keypair) {
        execute(
            client,
            payer,
            &[payer],
            &[system_instruction::transfer(
                &payer.pubkey(),
                &payer.pubkey(),
                1,
            )],
            CommitmentConfig::finalized(),
        )
        .unwrap();
    }

    /// Fetch account data, the loop is there to re-attempt until data is available.
    pub fn get_account_data<T: BorshDeserialize>(
        client: &RpcClient,
        account: &Pubkey,
    ) -> Option<T> {
        let account = client
            .get_account_with_commitment(account, CommitmentConfig::processed())
            .unwrap();
        T::try_from_slice(&account.value.unwrap().data).ok()
    }

    pub fn initialize_bridge(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
    ) -> Result<Signature, ClientError> {
        let initial_guardians = &[[1u8; 20]];
        execute(
            client,
            payer,
            &[payer],
            &[bridge::instructions::initialize(
                *program,
                payer.pubkey(),
                50,
                2_000_000_000,
                initial_guardians,
            )
            .unwrap()],
            CommitmentConfig::processed(),
        )
    }

    pub fn transfer(
        client: &RpcClient,
        from: &Keypair,
        to: &Pubkey,
        lamports: u64,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            from,
            &[from],
            &[system_instruction::transfer(&from.pubkey(), to, lamports)],
            CommitmentConfig::processed(),
        )
    }

    pub fn initialize(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        bridge: &Pubkey,
    ) -> Result<Signature, ClientError> {
        let instruction = instructions::initialize(*program, payer.pubkey(), *bridge)
            .expect("Could not create Initialize instruction");

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer],
            &[instruction],
            CommitmentConfig::processed(),
        )
    }

    pub fn attest(
        client: &RpcClient,
        program: &Pubkey,
        bridge: &Pubkey,
        payer: &Keypair,
        message: &Keypair,
        mint: Pubkey,
        nonce: u32,
    ) -> Result<Signature, ClientError> {
        let mint_data = Mint::unpack(
            &client
                .get_account_with_commitment(&mint, CommitmentConfig::processed())?
                .value
                .unwrap()
                .data,
        )
        .expect("Could not unpack Mint");

        let instruction = instructions::attest(
            *program,
            *bridge,
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
            CommitmentConfig::processed(),
        )
    }

    pub fn transfer_native(
        client: &RpcClient,
        program: &Pubkey,
        bridge: &Pubkey,
        payer: &Keypair,
        message: &Keypair,
        from: &Keypair,
        from_owner: &Keypair,
        mint: Pubkey,
        amount: u64,
    ) -> Result<Signature, ClientError> {
        let instruction = instructions::transfer_native(
            *program,
            *bridge,
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
                    &token_bridge::accounts::AuthoritySigner::key(None, program),
                    &from_owner.pubkey(),
                    &[],
                    amount,
                )
                .unwrap(),
                instruction,
            ],
            CommitmentConfig::processed(),
        )
    }

    pub fn transfer_wrapped(
        client: &RpcClient,
        program: &Pubkey,
        bridge: &Pubkey,
        payer: &Keypair,
        message: &Keypair,
        from: Pubkey,
        from_owner: &Keypair,
        token_chain: u16,
        token_address: Address,
        amount: u64,
    ) -> Result<Signature, ClientError> {
        let instruction = instructions::transfer_wrapped(
            *program,
            *bridge,
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
                    &token_bridge::accounts::AuthoritySigner::key(None, program),
                    &from_owner.pubkey(),
                    &[],
                    amount,
                )
                .unwrap(),
                instruction,
            ],
            CommitmentConfig::processed(),
        )
    }

    pub fn register_chain(
        client: &RpcClient,
        program: &Pubkey,
        bridge: &Pubkey,
        message_acc: &Pubkey,
        vaa: PostVAAData,
        payload: PayloadGovernanceRegisterChain,
        payer: &Keypair,
    ) -> Result<Signature, ClientError> {
        let instruction = instructions::register_chain(
            *program,
            *bridge,
            payer.pubkey(),
            *message_acc,
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
            CommitmentConfig::processed(),
        )
    }

    pub fn complete_native(
        client: &RpcClient,
        program: &Pubkey,
        bridge: &Pubkey,
        message_acc: &Pubkey,
        vaa: PostVAAData,
        payload: PayloadTransfer,
        payer: &Keypair,
    ) -> Result<Signature, ClientError> {
        let instruction = instructions::complete_native(
            *program,
            *bridge,
            payer.pubkey(),
            *message_acc,
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
            CommitmentConfig::processed(),
        )
    }

    pub fn complete_transfer_wrapped(
        client: &RpcClient,
        program: &Pubkey,
        bridge: &Pubkey,
        message_acc: &Pubkey,
        vaa: PostVAAData,
        payload: PayloadTransfer,
        payer: &Keypair,
    ) -> Result<Signature, ClientError> {
        let to = Pubkey::new(&payload.to[..]);

        let instruction = instructions::complete_wrapped(
            *program,
            *bridge,
            payer.pubkey(),
            *message_acc,
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
            CommitmentConfig::processed(),
        )
    }

    pub fn create_wrapped(
        client: &RpcClient,
        program: &Pubkey,
        bridge: &Pubkey,
        message_acc: &Pubkey,
        vaa: PostVAAData,
        payload: PayloadAssetMeta,
        payer: &Keypair,
    ) -> Result<Signature, ClientError> {
        let instruction = instructions::create_wrapped(
            *program,
            *bridge,
            payer.pubkey(),
            *message_acc,
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
            CommitmentConfig::processed(),
        )
    }

    pub fn create_mint(
        client: &RpcClient,
        payer: &Keypair,
        mint_authority: &Pubkey,
        mint: &Keypair,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            payer,
            &[payer, mint],
            &[
                solana_sdk::system_instruction::create_account(
                    &payer.pubkey(),
                    &mint.pubkey(),
                    Rent::default().minimum_balance(spl_token::state::Mint::LEN),
                    spl_token::state::Mint::LEN as u64,
                    &spl_token::id(),
                ),
                spl_token::instruction::initialize_mint(
                    &spl_token::id(),
                    &mint.pubkey(),
                    mint_authority,
                    None,
                    0,
                )
                .unwrap(),
            ],
            CommitmentConfig::processed(),
        )
    }

    pub fn create_spl_metadata(
        client: &RpcClient,
        payer: &Keypair,
        metadata_account: &Pubkey,
        mint_authority: &Keypair,
        mint: &Keypair,
        update_authority: &Pubkey,
        name: String,
        symbol: String,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            payer,
            &[payer, mint_authority],
            &[spl_token_metadata::instruction::create_metadata_accounts(
                spl_token_metadata::id(),
                *metadata_account,
                mint.pubkey(),
                mint_authority.pubkey(),
                payer.pubkey(),
                *update_authority,
                name,
                symbol,
                "https://token.org".to_string(),
                None,
                0,
                false,
                false,
            )],
            CommitmentConfig::processed(),
        )
    }

    pub fn create_token_account(
        client: &RpcClient,
        payer: &Keypair,
        token_acc: &Keypair,
        token_authority: Pubkey,
        mint: Pubkey,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            payer,
            &[payer, token_acc],
            &[
                solana_sdk::system_instruction::create_account(
                    &payer.pubkey(),
                    &token_acc.pubkey(),
                    Rent::default().minimum_balance(spl_token::state::Account::LEN),
                    spl_token::state::Account::LEN as u64,
                    &spl_token::id(),
                ),
                spl_token::instruction::initialize_account(
                    &spl_token::id(),
                    &token_acc.pubkey(),
                    &mint,
                    &token_authority,
                )
                .unwrap(),
            ],
            CommitmentConfig::processed(),
        )
    }

    pub fn mint_tokens(
        client: &RpcClient,
        payer: &Keypair,
        mint_authority: &Keypair,
        mint: &Keypair,
        token_account: &Pubkey,
        amount: u64,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            payer,
            &[payer, &mint_authority],
            &[spl_token::instruction::mint_to(
                &spl_token::id(),
                &mint.pubkey(),
                token_account,
                &mint_authority.pubkey(),
                &[],
                amount,
            )
            .unwrap()],
            CommitmentConfig::processed(),
        )
    }

    /// Utility function for generating VAA's from message data.
    pub fn generate_vaa(
        emitter: Address,
        emitter_chain: u16,
        data: Vec<u8>,
        nonce: u32,
        sequence: u64,
    ) -> (PostVAAData, [u8; 32], [u8; 32]) {
        let mut vaa = PostVAAData {
            version: 0,
            guardian_set_index: 0,

            // Body part
            emitter_chain,
            emitter_address: emitter,
            sequence,
            payload: data,
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
            v.write(&vaa.emitter_address).unwrap();
            v.write_u64::<BigEndian>(vaa.sequence).unwrap();
            v.write_u8(vaa.consistency_level).unwrap();
            v.write(&vaa.payload).unwrap();
            v.into_inner()
        };

        // Hash this body, which is expected to be the same as the hash currently stored in the
        // signature account, binding that set of signatures to this VAA.
        let body: [u8; 32] = {
            let mut h = sha3::Keccak256::default();
            h.write(body.as_slice()).unwrap();
            h.finalize().into()
        };

        let body_hash: [u8; 32] = {
            let mut h = sha3::Keccak256::default();
            h.write(&body).unwrap();
            h.finalize().into()
        };

        (vaa, body, body_hash)
    }

    pub fn post_vaa(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        vaa: PostVAAData,
    ) -> Result<(), ClientError> {
        let instruction =
            bridge::instructions::post_vaa(*program, payer.pubkey(), Pubkey::new_unique(), vaa);

        for account in instruction.accounts.iter().enumerate() {
            println!("{}: {}", account.0, account.1.pubkey);
        }

        execute(
            client,
            payer,
            &[payer],
            &[instruction],
            CommitmentConfig::processed(),
        )?;

        Ok(())
    }

    pub fn post_message(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        emitter: &Keypair,
        message: &Keypair,
        nonce: u32,
        data: Vec<u8>,
        fee: u64,
    ) -> Result<(), ClientError> {
        // Transfer money into the fee collector as it needs a balance/must exist.
        let fee_collector = FeeCollector::<'_>::key(None, program);

        // Capture the resulting message, later functions will need this.
        let instruction = bridge::instructions::post_message(
            *program,
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
            CommitmentConfig::processed(),
        )?;

        Ok(())
    }
}
