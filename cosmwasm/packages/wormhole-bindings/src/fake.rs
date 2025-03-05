use std::{cell::RefCell, collections::BTreeSet, fmt::Debug, rc::Rc};

use anyhow::{anyhow, bail, ensure, Context};
use cosmwasm_std::{
    to_json_binary, Addr, Api, Binary, BlockInfo, CustomQuery, Empty, Querier, Storage,
};
use cw_multi_test::{AppResponse, CosmosRouter, Module};
use k256::ecdsa::{recoverable, signature::Signer, SigningKey};
use schemars::JsonSchema;
use serde::{de::DeserializeOwned, Serialize};
use serde_wormhole::RawMessage;
use wormhole_sdk::{
    token::Message,
    vaa::{digest, Body, Header, Signature},
    Address, Chain, Vaa, GOVERNANCE_EMITTER,
};

use crate::WormholeQuery;

pub fn default_guardian_keys() -> [SigningKey; 7] {
    [
        SigningKey::from_bytes(&[
            93, 217, 189, 224, 168, 81, 157, 93, 238, 38, 143, 8, 182, 94, 69, 77, 232, 199, 238,
            206, 15, 135, 221, 58, 43, 74, 0, 129, 54, 198, 62, 226,
        ])
        .unwrap(),
        SigningKey::from_bytes(&[
            150, 48, 135, 223, 194, 186, 243, 139, 177, 8, 126, 32, 210, 57, 42, 28, 29, 102, 196,
            201, 106, 136, 40, 149, 218, 150, 240, 213, 192, 128, 161, 245,
        ])
        .unwrap(),
        SigningKey::from_bytes(&[
            121, 51, 199, 93, 237, 227, 62, 220, 128, 129, 195, 4, 190, 163, 254, 12, 212, 224,
            188, 76, 141, 242, 229, 121, 192, 5, 161, 176, 136, 99, 83, 53,
        ])
        .unwrap(),
        SigningKey::from_bytes(&[
            224, 180, 4, 114, 215, 161, 184, 12, 218, 96, 20, 141, 154, 242, 46, 230, 167, 165, 54,
            141, 108, 64, 146, 27, 193, 89, 251, 139, 234, 132, 124, 30,
        ])
        .unwrap(),
        SigningKey::from_bytes(&[
            69, 1, 17, 179, 19, 47, 56, 47, 255, 219, 143, 89, 115, 54, 242, 209, 163, 131, 225,
            30, 59, 195, 217, 141, 167, 253, 6, 95, 252, 52, 7, 223,
        ])
        .unwrap(),
        SigningKey::from_bytes(&[
            181, 3, 165, 125, 15, 200, 155, 56, 157, 204, 105, 221, 203, 149, 215, 175, 220, 228,
            200, 37, 169, 39, 68, 127, 132, 196, 203, 232, 155, 55, 67, 253,
        ])
        .unwrap(),
        SigningKey::from_bytes(&[
            72, 81, 175, 107, 23, 108, 178, 66, 32, 53, 14, 117, 233, 33, 114, 102, 68, 89, 83,
            201, 129, 57, 56, 130, 214, 212, 172, 16, 23, 22, 234, 160,
        ])
        .unwrap(),
    ]
}

#[derive(Debug)]
struct Inner {
    index: u32,
    expiration: u64,
    guardians: Vec<SigningKey>,
}

#[derive(Clone, Debug)]
pub struct WormholeKeeper(Rc<RefCell<Inner>>);

impl WormholeKeeper {
    pub fn new() -> WormholeKeeper {
        WormholeKeeper(Rc::new(RefCell::new(Inner {
            index: 0,
            expiration: 0,
            guardians: default_guardian_keys().to_vec(),
        })))
    }

    pub fn sign(&self, msg: &[u8]) -> Vec<Signature> {
        let d = digest(msg).unwrap();
        self.0
            .borrow()
            .guardians
            .iter()
            .map(|g| {
                let sig: recoverable::Signature = g.sign(&d.hash[..]);
                sig.as_ref().try_into().unwrap()
            })
            .enumerate()
            .map(|(idx, sig)| Signature {
                index: idx as u8,
                signature: sig,
            })
            .collect()
    }

    pub fn sign_message(&self, msg: &[u8]) -> Vec<Signature> {
        self.0
            .borrow()
            .guardians
            .iter()
            .map(|g| {
                let sig: recoverable::Signature = g.sign(msg);
                sig.as_ref().try_into().unwrap()
            })
            .enumerate()
            .map(|(idx, sig)| Signature {
                index: idx as u8,
                signature: sig,
            })
            .collect()
    }

    pub fn verify_vaa(&self, vaa: &[u8], block_time: u64) -> anyhow::Result<Empty> {
        let (header, data) = serde_wormhole::from_slice::<(Header, &RawMessage)>(vaa)
            .context("failed to parse VAA header")?;

        let mut signers = BTreeSet::new();
        for s in &header.signatures {
            // Vaa's are double hashed
            let digest = digest(data).context("unable to create digest of vaa body")?;
            self.verify_signature(&[], &digest.hash, header.guardian_set_index, s, block_time)?;
            signers.insert(s.index);
        }

        if signers.len() as u32 >= self.calculate_quorum(header.guardian_set_index, block_time)? {
            Ok(Empty {})
        } else {
            Err(anyhow!("no quorum"))
        }
    }

    pub fn verify_signature(
        &self,
        prefix: &[u8],
        data: &[u8],
        index: u32,
        sig: &Signature,
        block_time: u64,
    ) -> anyhow::Result<Empty> {
        let this = self.0.borrow();
        ensure!(this.index == index, "invalid guardian set");
        ensure!(
            this.expiration == 0 || block_time < this.expiration,
            "guardian set expired"
        );
        let mut prepended = Vec::with_capacity(prefix.len() + data.len());
        prepended.extend_from_slice(prefix);
        prepended.extend_from_slice(data);

        let d = digest(prepended.as_slice()).context("failed to calculate digest for data")?;
        if let Some(g) = this.guardians.get(sig.index as usize) {
            let s = recoverable::Signature::try_from(&sig.signature[..])
                .context("failed to decode signature")?;
            let verifying_key = s
                .recover_verifying_key_from_digest_bytes(&d.hash.into())
                .context("failed to recover verifying key")?;
            ensure!(
                g.verifying_key() == verifying_key,
                "failed to verify signature"
            );
            Ok(Empty {})
        } else {
            Err(anyhow!("invalid guardian index"))
        }
    }

    pub fn calculate_quorum(&self, index: u32, block_time: u64) -> anyhow::Result<u32> {
        let this = self.0.borrow();
        ensure!(this.index == index, "invalid guardian set");
        ensure!(
            this.expiration == 0 || block_time < this.expiration,
            "guardian set expired"
        );

        Ok(((this.guardians.len() as u32 * 10 / 3) * 2) / 10 + 1)
    }

    pub fn query(&self, request: WormholeQuery, block: &BlockInfo) -> anyhow::Result<Binary> {
        match request {
            WormholeQuery::VerifyVaa { vaa } => self
                .verify_vaa(&vaa, block.height)
                .and_then(|e| to_json_binary(&e).map_err(From::from)),
            WormholeQuery::VerifyMessageSignature {
                prefix,
                data,
                guardian_set_index,
                signature,
            } => self
                .verify_signature(&prefix, &data, guardian_set_index, &signature, block.height)
                .and_then(|e| to_json_binary(&e).map_err(From::from)),
            WormholeQuery::CalculateQuorum { guardian_set_index } => self
                .calculate_quorum(guardian_set_index, block.height)
                .and_then(|q| to_json_binary(&q).map_err(From::from)),
        }
    }

    pub fn expiration(&self) -> u64 {
        self.0.borrow().expiration
    }

    pub fn set_expiration(&self, expiration: u64) {
        self.0.borrow_mut().expiration = expiration;
    }

    pub fn guardian_set_index(&self) -> u32 {
        self.0.borrow().index
    }

    pub fn set_index(&self, index: u32) {
        self.0.borrow_mut().index = index;
    }

    pub fn num_guardians(&self) -> usize {
        self.0.borrow().guardians.len()
    }
}

impl Default for WormholeKeeper {
    fn default() -> Self {
        Self::new()
    }
}

impl From<Vec<SigningKey>> for WormholeKeeper {
    fn from(guardians: Vec<SigningKey>) -> Self {
        WormholeKeeper(Rc::new(RefCell::new(Inner {
            index: 0,
            expiration: 0,
            guardians,
        })))
    }
}

impl Module for WormholeKeeper {
    type ExecT = Empty;
    type QueryT = WormholeQuery;
    type SudoT = Empty;

    fn execute<ExecC, QueryC>(
        &self,
        _api: &dyn Api,
        _storage: &mut dyn Storage,
        _router: &dyn CosmosRouter<ExecC = ExecC, QueryC = QueryC>,
        _block: &BlockInfo,
        sender: Addr,
        msg: Self::ExecT,
    ) -> anyhow::Result<AppResponse>
    where
        ExecC: Debug + Clone + PartialEq + JsonSchema + DeserializeOwned + 'static,
        QueryC: CustomQuery + DeserializeOwned + 'static,
    {
        bail!("Unexpected exec msg {msg:?} from {sender}")
    }

    fn sudo<ExecC, QueryC>(
        &self,
        _api: &dyn Api,
        _storage: &mut dyn Storage,
        _router: &dyn CosmosRouter<ExecC = ExecC, QueryC = QueryC>,
        _block: &BlockInfo,
        msg: Self::SudoT,
    ) -> anyhow::Result<AppResponse> {
        bail!("Unexpected sudo msg {msg:?}")
    }

    fn query(
        &self,
        _api: &dyn Api,
        _storage: &dyn Storage,
        _querier: &dyn Querier,
        block: &BlockInfo,
        request: Self::QueryT,
    ) -> anyhow::Result<Binary> {
        self.query(request, block)
    }
}

pub fn create_gov_vaa_body<Payload>(i: usize, payload: Payload) -> Body<Payload> {
    Body {
        timestamp: i as u32,
        nonce: i as u32,
        emitter_chain: Chain::Solana,
        emitter_address: GOVERNANCE_EMITTER,
        sequence: i as u64,
        consistency_level: 0,
        payload,
    }
}

pub fn create_vaa_body(
    i: usize,
    emitter_chain: impl Into<Chain>,
    emitter_address: Address,
    payload: Message,
) -> Body<Message> {
    Body {
        timestamp: i as u32,
        nonce: i as u32,
        emitter_chain: emitter_chain.into(),
        emitter_address,
        sequence: i as u64,
        consistency_level: 32,
        payload,
    }
}

pub trait SignVaa<M> {
    fn sign_vaa(self, wh: &WormholeKeeper) -> (Vaa<M>, Binary);
}

impl<M: Serialize> SignVaa<M> for Body<M> {
    fn sign_vaa(self, wh: &WormholeKeeper) -> (Vaa<M>, Binary) {
        let data = serde_wormhole::to_vec(&self).unwrap();
        let signatures = wh.sign(&data);

        let header = Header {
            version: 1,
            guardian_set_index: wh.guardian_set_index(),
            signatures,
        };

        let v: Vaa<M> = (header, self).into();
        let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();

        (v, data)
    }
}
