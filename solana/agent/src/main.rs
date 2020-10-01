use std::env;

use std::{io::Write, mem::size_of};

use std::str::FromStr;

use byteorder::{BigEndian, LittleEndian, ReadBytesExt, WriteBytesExt};
use solana_client::{
    client_error::ClientError, rpc_client::RpcClient, rpc_config::RpcSendTransactionConfig,
};
use solana_sdk::commitment_config::{CommitmentConfig, CommitmentLevel};

use solana_sdk::instruction::Instruction;

use solana_sdk::{
    pubkey::Pubkey,
    signature::{read_keypair_file, write_keypair_file, Keypair, Signature, Signer},
    system_instruction::create_account,
    transaction::Transaction,
};

use tokio::sync::mpsc;

use tonic::{transport::Server, Code, Request, Response, Status};

use service::{
    agent_server::{Agent, AgentServer},
    lockup_event::Event,
    Empty, LockupEvent, LockupEventNew, LockupEventVaaPosted, SubmitVaaRequest, SubmitVaaResponse,
    WatchLockupsRequest,
};
use spl_bridge::{
    instruction::{post_vaa, verify_signatures, VerifySigPayload, CHAIN_ID_SOLANA},
    state::{Bridge, GuardianSet, SignatureState, TransferOutProposal},
    vaa::VAA,
};

use crate::monitor::PubsubClient;

mod monitor;

pub mod service {
    include!(concat!(env!("OUT_DIR"), concat!("/", "agent.v1", ".rs")));
}

pub struct AgentImpl {
    url: String,
    bridge: Pubkey,

    rpc_url: String,
    key: Keypair,
}

pub struct SignatureItem {
    signature: [u8; 64 + 1],
    key: [u8; 20],
    index: u8,
}

#[tonic::async_trait]
impl Agent for AgentImpl {
    async fn submit_vaa(
        &self,
        request: Request<SubmitVaaRequest>,
    ) -> Result<Response<SubmitVaaResponse>, Status> {
        // Hack to clone keypair
        let b = self.key.to_bytes();
        let key = Keypair::from_bytes(&b).unwrap();
        let bridge = self.bridge.clone();

        let rpc_url = self.rpc_url.clone();

        // we need to spawn an extra thread because tokio does not allow nested runtimes
        std::thread::spawn(move || {
            let rpc = RpcClient::new(rpc_url);

            let sig_key = solana_sdk::signature::Keypair::new();

            let mut vaa = match VAA::deserialize(&request.get_ref().vaa) {
                Ok(v) => v,
                Err(e) => {
                    return Err(Status::new(
                        Code::InvalidArgument,
                        format!("could not parse VAA: {}", e),
                    ));
                }
            };
            let verify_txs = pack_sig_verification_txs(&rpc, &bridge, &vaa, &key, &sig_key)?;

            // Strip signatures
            vaa.signatures = Vec::new();
            let ix = match post_vaa(
                &bridge,
                &key.pubkey(),
                &sig_key.pubkey(),
                vaa.serialize().unwrap(),
            ) {
                Ok(v) => v,
                Err(e) => {
                    return Err(Status::new(
                        Code::InvalidArgument,
                        format!("could not create post_vaa instruction: {}", e),
                    ));
                }
            };
            let mut transaction2 = Transaction::new_with_payer(&[ix], Some(&key.pubkey()));

            for (mut tx, signers) in verify_txs {
                match sign_and_send(&rpc, &mut tx, signers) {
                    Ok(_) => (),
                    Err(e) => {
                        return Err(Status::new(
                            Code::Unavailable,
                            format!("tx sending failed: {}", e),
                        ));
                    }
                };
            }

            match sign_and_send(&rpc, &mut transaction2, vec![&key]) {
                Ok(s) => Ok(Response::new(SubmitVaaResponse {
                    signature: s.to_string(),
                })),
                Err(e) => Err(Status::new(
                    Code::Unavailable,
                    format!("tx sending failed: {}", e),
                )),
            }
        })
        .join()
        .unwrap()

        //check_fee_payer_balance(
        //    config,
        //    minimum_balance_for_rent_exemption
        //        + fee_calculator.calculate_fee(&transaction.message()),
        //)?;
    }

    type WatchLockupsStream = mpsc::Receiver<Result<LockupEvent, Status>>;

    async fn watch_lockups(
        &self,
        _req: Request<WatchLockupsRequest>,
    ) -> Result<Response<Self::WatchLockupsStream>, Status> {
        let (mut tx, rx) = mpsc::channel(1);
        let url = self.url.clone();
        let bridge = self.bridge.clone();
        let rpc_url = self.rpc_url.clone();

        tokio::spawn(async move {
            let _rpc = RpcClient::new(rpc_url.to_string());
            let sub = PubsubClient::program_subscribe(&url, &bridge).unwrap();
            // looping and sending our response using stream
            loop {
                let item = sub.1.recv();
                match item {
                    Ok(v) => {
                        // We only want to track lockups
                        if v.value.account.data.len() != size_of::<TransferOutProposal>() {
                            continue;
                        }

                        println!("lockup changed in slot: {}", v.context.slot);

                        let b = match Bridge::unpack_immutable::<TransferOutProposal>(
                            v.value.account.data.as_slice(),
                        ) {
                            Ok(v) => v,
                            Err(e) => {
                                println!("failed to deserialize lockup: {}", e);
                                continue;
                            }
                        };

                        let mut amount_b: [u8; 32] = [0; 32];
                        b.amount.to_big_endian(&mut amount_b);

                        let event = if b.vaa_time == 0 {
                            // The Lockup was created
                            LockupEvent {
                                slot: v.context.slot,
                                lockup_address: v.value.pubkey.to_string(),
                                time: b.lockup_time as u64,
                                event: Some(Event::New(LockupEventNew {
                                    nonce: b.nonce,
                                    source_chain: CHAIN_ID_SOLANA as u32,
                                    target_chain: b.to_chain_id as u32,
                                    source_address: b.source_address.to_vec(),
                                    target_address: b.foreign_address.to_vec(),
                                    token_chain: b.asset.chain as u32,
                                    token_address: b.asset.address.to_vec(),
                                    token_decimals: b.asset.decimals as u32,
                                    amount: amount_b.to_vec(),
                                })),
                            }
                        } else {
                            // The VAA was submitted
                            LockupEvent {
                                slot: v.context.slot,
                                lockup_address: v.value.pubkey.to_string(),
                                time: b.lockup_time as u64,
                                event: Some(Event::VaaPosted(LockupEventVaaPosted {
                                    nonce: b.nonce,
                                    source_chain: CHAIN_ID_SOLANA as u32,
                                    target_chain: b.to_chain_id as u32,
                                    source_address: b.source_address.to_vec(),
                                    target_address: b.foreign_address.to_vec(),
                                    token_chain: b.asset.chain as u32,
                                    token_address: b.asset.address.to_vec(),
                                    token_decimals: b.asset.decimals as u32,
                                    amount: amount_b.to_vec(),
                                    vaa: b.vaa.to_vec(),
                                })),
                            }
                        };

                        let mut amount_b: [u8; 32] = [0; 32];
                        b.amount.to_big_endian(&mut amount_b);

                        if let Err(e) = tx.send(Ok(event)).await {
                            println!("sending event failed: {}", e);
                            return;
                        };
                        // We need to push a second message to flush the channel
                        // https://github.com/hyperium/tonic/issues/378
                        if let Err(e) = tx
                            .send(Ok(LockupEvent {
                                slot: 0,
                                time: 0,
                                lockup_address: String::from(""),
                                event: Some(Event::Empty(Empty {})),
                            }))
                            .await
                        {
                            println!("sending event failed: {}", e);
                            return;
                        };
                    }
                    Err(e) => {
                        println!("watcher died: {}", e);
                        tx.send(Err(Status::new(Code::Aborted, "watcher died")))
                            .await;
                        return;
                    }
                };
            }
        });

        Ok(Response::new(rx))
    }
}

fn pack_sig_verification_txs<'a>(
    rpc: &RpcClient,
    bridge: &Pubkey,
    vaa: &VAA,
    sender_keypair: &'a Keypair,
    sign_keypair: &'a Keypair,
) -> Result<Vec<(Transaction, Vec<&'a Keypair>)>, Status> {
    // Load guardian set
    let bridge_key = Bridge::derive_bridge_id(bridge).unwrap();
    let guardian_key =
        Bridge::derive_guardian_set_id(bridge, &bridge_key, vaa.guardian_set_index).unwrap();
    let guardian_account = rpc
        .get_account_with_commitment(
            &guardian_key,
            CommitmentConfig {
                commitment: CommitmentLevel::Single,
            },
        )
        .unwrap()
        .value
        .unwrap_or_default();
    let data = guardian_account.data;
    let guardian_set: &GuardianSet = Bridge::unpack_immutable(data.as_slice()).unwrap();

    // Map signatures to guardian set
    let mut signature_items: Vec<SignatureItem> = Vec::new();
    for s in vaa.signatures.iter() {
        let mut item = SignatureItem {
            signature: [0; 64 + 1],
            key: [0; 20],
            index: s.index,
        };

        item.signature[0..32].copy_from_slice(&s.r);
        item.signature[32..64].copy_from_slice(&s.s);
        item.signature[64] = s.v;
        item.key = guardian_set.keys[s.index as usize];

        signature_items.push(item);
    }

    let vaa_hash = match vaa.body_hash() {
        Ok(v) => v,
        Err(e) => {
            return Err(Status::new(
                Code::InvalidArgument,
                format!("could get vaa body hash: {}", e),
            ));
        }
    };
    let vaa_body = match vaa.signature_body() {
        Ok(v) => v,
        Err(e) => {
            return Err(Status::new(
                Code::InvalidArgument,
                format!("could get vaa body: {}", e),
            ));
        }
    };

    let mut verify_txs: Vec<(Transaction, Vec<&Keypair>)> = Vec::new();
    for (tx_index, chunk) in signature_items.chunks(6).enumerate() {
        let mut secp_payload = Vec::new();
        let mut signature_status = [-1i8; 20];

        let data_offset = 1 + chunk.len() * 11;
        let message_offset = data_offset + chunk.len() * 85;

        // 1 number of signatures
        secp_payload.write_u8(chunk.len() as u8);

        let secp_ix_index = if tx_index == 0 { 1u8 } else { 0u8 };
        // Secp signature info description (11 bytes * n)
        for (i, s) in chunk.iter().enumerate() {
            secp_payload.write_u16::<LittleEndian>((data_offset + 85 * i) as u16);
            secp_payload.write_u8(secp_ix_index);
            secp_payload.write_u16::<LittleEndian>((data_offset + 85 * i + 65) as u16);
            secp_payload.write_u8(secp_ix_index);
            secp_payload.write_u16::<LittleEndian>(message_offset as u16);
            secp_payload.write_u16::<LittleEndian>(vaa_body.len() as u16);
            secp_payload.write_u8(secp_ix_index);
            signature_status[s.index as usize] = i as i8;
        }

        // Write signatures and addresses
        for s in chunk.iter() {
            secp_payload.write(&s.signature);
            secp_payload.write(&s.key);
        }

        // Write body
        secp_payload.write(&vaa_body);

        let secp_ix = Instruction {
            program_id: solana_sdk::secp256k1_program::id(),
            data: secp_payload,
            accounts: vec![],
        };

        let payload = VerifySigPayload {
            signers: signature_status,
            hash: vaa_hash,
        };
        let verify_ix = match verify_signatures(
            &bridge,
            &sign_keypair.pubkey(),
            vaa.guardian_set_index,
            &payload,
        ) {
            Ok(v) => v,
            Err(e) => {
                return Err(Status::new(
                    Code::InvalidArgument,
                    format!("could not create verify instruction: {}", e),
                ));
            }
        };

        if tx_index == 0 {
            // Instruction for creating the signature status account
            let min_sig_rent = rpc
                .get_minimum_balance_for_rent_exemption(size_of::<SignatureState>())
                .unwrap();
            let create_ix = create_account(
                &sender_keypair.pubkey(),
                &sign_keypair.pubkey(),
                min_sig_rent,
                size_of::<SignatureState>() as u64,
                bridge,
            );

            verify_txs.push((
                Transaction::new_with_payer(
                    &[create_ix, secp_ix, verify_ix],
                    Some(&sender_keypair.pubkey()),
                ),
                vec![sender_keypair, sign_keypair],
            ))
        } else {
            verify_txs.push((
                Transaction::new_with_payer(&[secp_ix, verify_ix], Some(&sender_keypair.pubkey())),
                vec![sender_keypair],
            ))
        }
    }

    Ok(verify_txs)
}

fn sign_and_send(
    rpc: &RpcClient,
    tx: &mut Transaction,
    keys: Vec<&Keypair>,
) -> Result<Signature, ClientError> {
    let (recent_blockhash, _fee_calculator) = rpc.get_recent_blockhash()?;

    tx.sign(&keys, recent_blockhash);

    rpc.send_and_confirm_transaction_with_spinner_and_config(
        &tx,
        CommitmentConfig {
            commitment: CommitmentLevel::Single,
        },
        RpcSendTransactionConfig {
            skip_preflight: false,
            preflight_commitment: Some(CommitmentLevel::SingleGossip),
        },
    )
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args: Vec<String> = env::args().collect();

    // TODO use clap
    if args.len() < 6 {
        println!("<bridge> <rpc_host> <rpc_port> <ws_port> <port>");
        return Ok(());
    }

    let bridge = &args[1];
    let host = &args[2];
    let rpc_port: u16 = args[3].parse()?;
    let ws_port: u16 = args[4].parse()?;
    let port: u16 = args[5].parse()?;

    let addr = format!("0.0.0.0:{}", port).parse().unwrap();

    let keypair = {
        if let Ok(k) = read_keypair_file("id.json") {
            k
        } else {
            let k = Keypair::new();
            write_keypair_file(&k, "id.json").unwrap();
            k
        }
    };

    println!("Agent using account: {}", keypair.pubkey());

    let agent = AgentImpl {
        url: String::from(format!("ws://{}:{}", host, ws_port)),
        rpc_url: format!("http://{}:{}", host, rpc_port),
        bridge: Pubkey::from_str(bridge).unwrap(),
        key: keypair,
    };

    println!("Agent listening on {}", addr);

    Server::builder()
        .add_service(AgentServer::new(agent))
        .serve(addr)
        .await?;

    Ok(())
}
