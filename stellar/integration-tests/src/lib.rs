use secp256k1::{Message, SecretKey, ecdsa::RecoverableSignature};
use serde_json::Value;
use std::{process::Command, thread::sleep, time::Duration};
use tiny_keccak::{Hasher, Keccak};

pub fn run(cmd: &mut Command) -> String {
    let out = cmd.output().expect("failed to spawn command");
    if !out.status.success() {
        panic!(
            "command failed: {:?}\nstdout:\n{}\nstderr:\n{}",
            cmd,
            String::from_utf8_lossy(&out.stdout),
            String::from_utf8_lossy(&out.stderr)
        );
    }
    String::from_utf8(out.stdout).expect("stdout not utf8")
}

pub fn rpc_call(rpc_url: &str, body: &str) -> Value {
    let out = run(Command::new("curl")
        .arg("-s")
        .arg("-X")
        .arg("POST")
        .arg(rpc_url)
        .arg("-H")
        .arg("Content-Type: application/json")
        .arg("-d")
        .arg(body));
    let v: Value = serde_json::from_str(&out).expect("rpc response not json");
    if v["error"].is_object() {
        panic!("RPC error: {}\nRequest: {}", v["error"], body);
    }
    v
}

pub fn keccak256(data: &[u8]) -> [u8; 32] {
    let mut hasher = Keccak::v256();
    let mut out = [0u8; 32];
    hasher.update(data);
    hasher.finalize(&mut out);
    out
}

pub fn eth_address_from_privkey(privkey: &[u8; 32]) -> [u8; 20] {
    let sk = SecretKey::from_secret_bytes(*privkey).unwrap();
    let pk = secp256k1::PublicKey::from_secret_key(&sk);
    let pk_serialized = pk.serialize_uncompressed();
    let hash = keccak256(&pk_serialized[1..]);
    let mut addr = [0u8; 20];
    addr.copy_from_slice(&hash[12..]);
    addr
}

pub struct TestContext {
    pub network: String,
    pub admin_identity: String,
    pub rpc_url: String,
    pub wasm_path: String,
}

impl Default for TestContext {
    fn default() -> Self {
        Self::new()
    }
}

impl TestContext {
    pub fn new() -> Self {
        Self {
            network: std::env::var("STELLAR_NETWORK").unwrap_or_else(|_| "local".to_string()),
            admin_identity: std::env::var("STELLAR_IDENTITY").expect("STELLAR_IDENTITY not set"),
            rpc_url: std::env::var("SOROBAN_RPC_URL").expect("SOROBAN_RPC_URL not set"),
            wasm_path: std::env::var("WORMHOLE_WASM_PATH").expect("WORMHOLE_WASM_PATH not set"),
        }
    }

    /// Deploy the wormhole contract with constructor args (initial_guardians
    /// and governance_emitter). The contract is initialized via
    /// __constructor at deploy time; there is no separate initialize.
    pub fn deploy_contract(
        &self,
        initial_guardians: &[String],
        governance_emitter: &str,
    ) -> String {
        let guardians_json = format!(
            "[{}]",
            initial_guardians
                .iter()
                .map(|g| format!("\"{}\"", g))
                .collect::<Vec<_>>()
                .join(",")
        );
        run(Command::new("stellar").args([
            "contract",
            "deploy",
            "--network",
            &self.network,
            "--source",
            &self.admin_identity,
            "--wasm",
            &self.wasm_path,
            "--",
            "--initial_guardians",
            &guardians_json,
            "--governance_emitter",
            governance_emitter,
        ]))
        .trim()
        .to_string()
    }

    pub fn fund_identity(&self, name: &str) {
        run(Command::new("stellar").args([
            "keys",
            "fund",
            "--network",
            &self.network,
            name,
        ]));
    }

    pub fn get_identity_address(&self, name: &str) -> String {
        run(Command::new("stellar").args(["keys", "address", name]))
            .trim()
            .to_string()
    }

    pub fn setup_identity(&self, name: &str) -> String {
        let _ = Command::new("stellar").args(["keys", "rm", name]).output();
        run(Command::new("stellar").args([
            "keys",
            "generate",
            "--network",
            &self.network,
            name,
        ]));
        let addr = run(Command::new("stellar").args(["keys", "address", name]))
            .trim()
            .to_string();
        self.fund_identity(name);
        addr
    }

    pub fn deploy_native_asset(&self) {
        let _ = Command::new("stellar")
            .args([
                "contract",
                "asset",
                "deploy",
                "--asset",
                "native",
                "--network",
                &self.network,
                "--source",
                &self.admin_identity,
            ])
            .output();
    }

    /// Native XLM token contract ID for the current network (differs per
    /// network, e.g. local vs testnet).
    pub fn get_native_asset_id(&self) -> String {
        run(Command::new("stellar").args([
            "contract",
            "id",
            "asset",
            "--asset",
            "native",
            "--network",
            &self.network,
        ]))
        .trim()
        .to_string()
    }

    pub fn invoke(&self, source: &str, id: &str, func: &str, args: &[&str]) -> String {
        let mut cmd = Command::new("stellar");
        cmd.args([
            "contract",
            "invoke",
            "--network",
            &self.network,
            "--source",
            source,
            "--id",
            id,
            "--",
            func,
        ]);
        cmd.args(args);
        run(&mut cmd)
    }

    pub fn get_balance(&self, asset_id: &str, address: &str) -> i128 {
        let out = self.invoke(
            &self.admin_identity,
            asset_id,
            "balance",
            &["--id", address],
        );
        out.trim()
            .trim_matches('"')
            .parse::<i128>()
            .expect("failed to parse balance")
    }
}

pub fn craft_governance_payload(action: u8, action_payload: &[u8]) -> Vec<u8> {
    let mut payload = Vec::new();
    let mut module = [0u8; 32];
    module[28..32].copy_from_slice(b"Core");
    payload.extend_from_slice(&module);
    payload.push(action);
    payload.extend_from_slice(&61u16.to_be_bytes()); // Chain ID: Stellar
    payload.extend_from_slice(action_payload);
    payload
}

pub fn assemble_vaa(
    guardian_set_index: u32,
    signatures: Vec<(u8, [u8; 64], u8)>,
    body: &[u8],
) -> Vec<u8> {
    let mut vaa = Vec::new();
    vaa.push(1); // Version
    vaa.extend_from_slice(&guardian_set_index.to_be_bytes());
    vaa.push(signatures.len() as u8);
    for (guardian_index, compact_sig, recovery_id) in signatures {
        vaa.push(guardian_index);
        vaa.extend_from_slice(&compact_sig);
        vaa.push(recovery_id);
    }
    vaa.extend_from_slice(body);
    vaa
}

pub fn craft_vaa_body(
    emitter_chain: u16,
    emitter_address: [u8; 32],
    nonce: u32,
    sequence: u64,
    payload: &[u8],
) -> Vec<u8> {
    let mut body = Vec::new();
    body.extend_from_slice(&0u32.to_be_bytes());
    body.extend_from_slice(&nonce.to_be_bytes());
    body.extend_from_slice(&emitter_chain.to_be_bytes());
    body.extend_from_slice(&emitter_address);
    body.extend_from_slice(&sequence.to_be_bytes());
    body.push(1);
    body.extend_from_slice(payload);
    body
}

pub fn sign_vaa_body(body: &[u8], privkey: [u8; 32]) -> (u8, [u8; 64]) {
    let body_hash = keccak256(&keccak256(body));
    let sk = SecretKey::from_secret_bytes(privkey).unwrap();
    let msg = Message::from_digest(body_hash);
    let sig = RecoverableSignature::sign_ecdsa_recoverable(msg, &sk);
    let (recid, compact) = sig.serialize_compact();
    (recid.to_u8(), compact)
}

pub fn find_event(rpc_url: &str, contract_id: &str, topic_filters: &[Vec<&str>]) -> bool {
    for _ in 0..15 {
        let latest = rpc_call(
            rpc_url,
            r#"{"jsonrpc":"2.0","id":1,"method":"getLatestLedger","params":{}}"#,
        );
        let latest_seq = latest["result"]["sequence"]
            .as_u64()
            .expect("latest ledger sequence missing");
        let start_ledger = latest_seq.saturating_sub(100).max(1);

        let events = rpc_call(
            rpc_url,
            &format!(
                r#"{{
                  "jsonrpc":"2.0","id":1,"method":"getEvents",
                  "params":{{
                    "startLedger": {start},
                    "endLedger": {end},
                    "filters": [{{"type":"contract","contractIds":["{cid}"]}}]
                  }}
                }}"#,
                start = start_ledger,
                end = latest_seq,
                cid = contract_id
            ),
        );

        let records = events["result"]["events"]
            .as_array()
            .expect("events result missing events array");
        for ev in records {
            let ev_str = ev.to_string();
            if topic_filters
                .iter()
                .all(|alternatives| alternatives.iter().any(|&s| ev_str.contains(s)))
            {
                return true;
            }
        }
        sleep(Duration::from_secs(1));
    }
    false
}
