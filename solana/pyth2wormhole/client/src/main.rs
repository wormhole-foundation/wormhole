pub mod attestation_cfg;
pub mod cli;

use std::{
    fs::File,
    path::{
        Path,
        PathBuf,
    },
};

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use clap::Clap;
use log::{
    debug,
    error,
    info,
    warn,
    LevelFilter,
};
use solana_client::rpc_client::RpcClient;
use solana_program::{
    hash::Hash,
    instruction::{
        AccountMeta,
        Instruction,
    },
    pubkey::Pubkey,
    system_program,
    sysvar::{
        clock,
        rent,
    },
};
use solana_sdk::{
    commitment_config::CommitmentConfig,
    signature::read_keypair_file,
    transaction::Transaction,
};
use solana_transaction_status::UiTransactionEncoding;
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
    Derive,
    Info,
};
use solitaire_client::{
    AccEntry,
    Keypair,
    SolSigner,
    ToInstruction,
};

use cli::{
    Action,
    Cli,
};

use bridge::{
    accounts::{
        Bridge,
        FeeCollector,
        Sequence,
        SequenceDerivationData,
    },
    types::ConsistencyLevel,
    CHAIN_ID_SOLANA,
};

use pyth2wormhole::{
    attest::{
        P2WEmitter,
        P2W_MAX_BATCH_SIZE,
    },
    config::P2WConfigAccount,
    initialize::InitializeAccounts,
    set_config::SetConfigAccounts,
    types::PriceAttestation,
    AttestData,
    Pyth2WormholeConfig,
};

use crate::attestation_cfg::AttestationConfig;

pub type ErrBox = Box<dyn std::error::Error>;

pub const SEQNO_PREFIX: &'static str = "Program log: Sequence: ";

fn main() -> Result<(), ErrBox> {
    let cli = Cli::parse();
    init_logging(cli.log_level);

    let payer = read_keypair_file(&*shellexpand::tilde(&cli.payer))?;
    let rpc_client = RpcClient::new_with_commitment(cli.rpc_url, CommitmentConfig::finalized());

    let p2w_addr = cli.p2w_addr;

    let latest_blockhash = rpc_client.get_latest_blockhash()?;

    match cli.action {
        Action::Init {
            owner_addr,
            pyth_owner_addr,
            wh_prog,
        } => {
            let tx = handle_init(
                payer,
                p2w_addr,
                owner_addr,
                wh_prog,
                pyth_owner_addr,
                latest_blockhash,
            )?;
            rpc_client.send_and_confirm_transaction_with_spinner(&tx)?;
        }
        Action::SetConfig {
            ref owner,
            new_owner_addr,
            new_wh_prog,
            new_pyth_owner_addr,
        } => {
            let tx = handle_set_config(
                payer,
                p2w_addr,
                read_keypair_file(&*shellexpand::tilde(&owner))?,
                new_owner_addr,
                new_wh_prog,
                new_pyth_owner_addr,
                latest_blockhash,
            )?;
            rpc_client.send_and_confirm_transaction_with_spinner(&tx)?;
        }
        Action::Attest {
            ref attestation_cfg,
        } => {
            // Load the attestation config yaml
            let attestation_cfg: AttestationConfig =
                serde_yaml::from_reader(File::open(attestation_cfg)?)?;

            handle_attest(&rpc_client, payer, p2w_addr, &attestation_cfg)?;
        }
    }

    Ok(())
}

fn handle_init(
    payer: Keypair,
    p2w_addr: Pubkey,
    new_owner_addr: Pubkey,
    wh_prog: Pubkey,
    pyth_owner_addr: Pubkey,
    latest_blockhash: Hash,
) -> Result<Transaction, ErrBox> {
    use AccEntry::*;

    let payer_pubkey = payer.pubkey();

    let accs = InitializeAccounts {
        payer: Signer(payer),
        new_config: Derived(p2w_addr),
    };

    let config = Pyth2WormholeConfig {
        max_batch_size: P2W_MAX_BATCH_SIZE,
        owner: new_owner_addr,
        wh_prog: wh_prog,
        pyth_owner: pyth_owner_addr,
    };
    let ix_data = (pyth2wormhole::instruction::Instruction::Initialize, config);

    let (ix, signers) = accs.to_ix(p2w_addr, ix_data.try_to_vec()?.as_slice())?;

    let tx_signed = Transaction::new_signed_with_payer::<Vec<&Keypair>>(
        &[ix],
        Some(&payer_pubkey),
        signers.iter().collect::<Vec<_>>().as_ref(),
        latest_blockhash,
    );
    Ok(tx_signed)
}

fn handle_set_config(
    payer: Keypair,
    p2w_addr: Pubkey,
    owner: Keypair,
    new_owner_addr: Pubkey,
    new_wh_prog: Pubkey,
    new_pyth_owner_addr: Pubkey,
    latest_blockhash: Hash,
) -> Result<Transaction, ErrBox> {
    use AccEntry::*;

    let payer_pubkey = payer.pubkey();

    let accs = SetConfigAccounts {
        payer: Signer(payer),
        current_owner: Signer(owner),
        config: Derived(p2w_addr),
    };

    let config = Pyth2WormholeConfig {
        max_batch_size: P2W_MAX_BATCH_SIZE,
        owner: new_owner_addr,
        wh_prog: new_wh_prog,
        pyth_owner: new_pyth_owner_addr,
    };
    let ix_data = (pyth2wormhole::instruction::Instruction::SetConfig, config);

    let (ix, signers) = accs.to_ix(p2w_addr, ix_data.try_to_vec()?.as_slice())?;

    let tx_signed = Transaction::new_signed_with_payer::<Vec<&Keypair>>(
        &[ix],
        Some(&payer_pubkey),
        signers.iter().collect::<Vec<_>>().as_ref(),
        latest_blockhash,
    );
    Ok(tx_signed)
}

fn handle_attest(
    rpc_client: &RpcClient, // Needed for reading Pyth account data
    payer: Keypair,
    p2w_addr: Pubkey,
    attestation_cfg: &AttestationConfig,
) -> Result<(), ErrBox> {
    // Derive seeded accounts
    let emitter_addr = P2WEmitter::key(None, &p2w_addr);

    info!("Using emitter addr {}", emitter_addr);

    let p2w_config_addr = P2WConfigAccount::<{ AccountState::Initialized }>::key(None, &p2w_addr);

    let config = Pyth2WormholeConfig::try_from_slice(
        rpc_client.get_account_data(&p2w_config_addr)?.as_slice(),
    )?;

    let seq_addr = Sequence::key(
        &SequenceDerivationData {
            emitter_key: &emitter_addr,
        },
        &config.wh_prog,
    );

    // Read the current max batch size from the contract's settings
    let max_batch_size = config.max_batch_size;

    let batch_count = {
        let whole_batches = attestation_cfg.symbols.len() / config.max_batch_size as usize;

        // Include  partial batch if there is a remainder
        if attestation_cfg.symbols.len() % config.max_batch_size as usize > 0 {
            whole_batches + 1
        } else {
            whole_batches
        }
    };

    debug!("Symbol config:\n{:#?}", attestation_cfg);

    info!(
        "{} symbols read, max batch size {}, dividing into {} batches",
        attestation_cfg.symbols.len(),
        max_batch_size,
        batch_count
    );

    let mut errors = Vec::new();

    for (idx, symbols) in attestation_cfg
        .symbols
        .as_slice()
        .chunks(max_batch_size as usize)
        .enumerate()
    {
        let batch_no = idx + 1;
        let sym_msg_keypair = Keypair::new();
        info!(
            "Batch {}/{} contents: {:?}",
            batch_no,
            batch_count,
            symbols
                .iter()
                .map(|s| s
                    .name
                    .clone()
                    .unwrap_or(format!("unnamed product {:?}", s.product_addr)))
                .collect::<Vec<_>>()
        );

        let mut sym_metas_vec: Vec<_> = symbols
            .iter()
            .map(|s| {
                vec![
                    AccountMeta::new_readonly(s.product_addr, false),
                    AccountMeta::new_readonly(s.price_addr, false),
                ]
            })
            .flatten()
            .collect();

        // Align to max batch size with null accounts
        let mut blank_accounts =
            vec![
                AccountMeta::new_readonly(Pubkey::new_from_array([0u8; 32]), false);
                2 * (max_batch_size as usize - symbols.len())
            ];
        sym_metas_vec.append(&mut blank_accounts);

        // Arrange Attest accounts
        let mut acc_metas = vec![
            // payer
            AccountMeta::new(payer.pubkey(), true),
            // system_program
            AccountMeta::new_readonly(system_program::id(), false),
            // config
            AccountMeta::new_readonly(p2w_config_addr, false),
        ];

        // Insert max_batch_size metas
        acc_metas.append(&mut sym_metas_vec);

        // Continue with other pyth2wormhole accounts
        let mut acc_metas_remainder = vec![
            // clock
            AccountMeta::new_readonly(clock::id(), false),
            // wh_prog
            AccountMeta::new_readonly(config.wh_prog, false),
            // wh_bridge
            AccountMeta::new(
                Bridge::<{ AccountState::Initialized }>::key(None, &config.wh_prog),
                false,
            ),
            // wh_message
            AccountMeta::new(sym_msg_keypair.pubkey(), true),
            // wh_emitter
            AccountMeta::new_readonly(emitter_addr, false),
            // wh_sequence
            AccountMeta::new(seq_addr, false),
            // wh_fee_collector
            AccountMeta::new(FeeCollector::<'_>::key(None, &config.wh_prog), false),
            AccountMeta::new_readonly(rent::id(), false),
        ];

        acc_metas.append(&mut acc_metas_remainder);

        let ix_data = (
            pyth2wormhole::instruction::Instruction::Attest,
            AttestData {
                consistency_level: ConsistencyLevel::Finalized,
            },
        );

        let ix = Instruction::new_with_bytes(p2w_addr, ix_data.try_to_vec()?.as_slice(), acc_metas);

        // Execute the transaction, obtain the resulting sequence
        // number. The and_then() calls enforce error handling
        // location near loop end.
        let res = rpc_client
            .get_latest_blockhash()
            .and_then(|latest_blockhash| {
                let tx_signed = Transaction::new_signed_with_payer::<Vec<&Keypair>>(
                    &[ix],
                    Some(&payer.pubkey()),
                    &vec![&payer, &sym_msg_keypair],
                    latest_blockhash,
                );
                rpc_client.send_and_confirm_transaction_with_spinner(&tx_signed)
            })
            .and_then(|sig| rpc_client.get_transaction(&sig, UiTransactionEncoding::Json))
            .map_err(|e| -> ErrBox { e.into() })
            .and_then(|this_tx| {
                this_tx
                    .transaction
                    .meta
                    .and_then(|meta| meta.log_messages)
                    .and_then(|logs| {
                        let mut seqno = None;
                        for log in logs {
                            if log.starts_with(SEQNO_PREFIX) {
                                seqno = Some(log.replace(SEQNO_PREFIX, ""));
                                break;
                            }
                        }
                        seqno
                    })
                    .ok_or_else(|| format!("No seqno in program logs").into())
            });

        // Individual batch errors mustn't prevent other batches from being sent.
        match res {
            Ok(seqno) => {
                println!("Sequence number: {}", seqno);
                info!("Batch {}/{}: OK, seqno {}", batch_no, batch_count, seqno);
            }
            Err(e) => {
                let msg = format!(
                    "Batch {}/{} tx error: {}",
                    batch_no,
                    batch_count,
                    e.to_string()
                );
                error!("{}", &msg);

                errors.push(msg)
            }
        }
    }

    if errors.len() > 0 {
        let err_list = errors.join("\n");

        Err(format!("{} of {} batches failed:\n{}", errors.len(), batch_count, err_list).into())
    } else {
        Ok(())
    }
}

fn init_logging(verbosity: u32) {
    use LevelFilter::*;
    let filter = match verbosity {
        0..=1 => Error,
        2 => Warn,
        3 => Info,
        4 => Debug,
        _other => Trace,
    };

    env_logger::builder().filter_level(filter).init();
}
