use solana_program::pubkey::Pubkey;
use std::io::{Cursor, Write, Read};
use crate::error::Error;
use byteorder::{ReadBytesExt, WriteBytesExt, BigEndian};

#[derive(Clone, Debug, PartialEq)]
pub enum GovernanceCommand {
    UpdateGuardianSet(BodyUpdateGuardianSet),
    UpgradeContract(BodyContractUpgrade),
}

impl GovernanceCommand {
    fn action_id(&self) -> u8 {
        match self {
            GovernanceCommand::UpdateGuardianSet(_) => 0x01,
            GovernanceCommand::UpgradeContract(_) => 0x02,
        }
    }

    fn deserialize(data: &Vec<u8>) -> Result<GovernanceCommand, Error> {
        let mut payload_data = Cursor::new(data);
        let action = payload_data.read_u8()?;

        let payload = match action {
            0x01 => {
                GovernanceCommand::UpdateGuardianSet(BodyUpdateGuardianSet::deserialize(&mut payload_data)?)
            }
            0x02 => GovernanceCommand::UpgradeContract(BodyContractUpgrade::deserialize(&mut payload_data)?),
            _ => {
                return Err(Error::InvalidVAAAction);
            }
        };

        Ok(payload)
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        match self {
            GovernanceCommand::UpdateGuardianSet(b) => b.serialize(),
            GovernanceCommand::UpgradeContract(b) => b.serialize(),
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct BodyUpdateGuardianSet {
    pub new_index: u32,
    pub new_keys: Vec<[u8; 20]>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct BodyContractUpgrade {
    pub chain_id: u8,
    pub buffer: Pubkey,
}

impl BodyContractUpgrade {
    fn deserialize(data: &mut Cursor<&Vec<u8>>) -> Result<BodyContractUpgrade, Error> {
        let chain_id = data.read_u8()?;
        let mut key: [u8; 32] = [0; 32];
        data.read(&mut key[..])?;

        Ok(BodyContractUpgrade {
            chain_id,
            buffer: Pubkey::new(&key[..]),
        })
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());
        v.write_u8(self.chain_id)?;
        v.write(&self.buffer.to_bytes())?;

        Ok(v.into_inner())
    }
}

impl BodyUpdateGuardianSet {
    fn deserialize(data: &mut Cursor<&Vec<u8>>) -> Result<BodyUpdateGuardianSet, Error> {
        let new_index = data.read_u32::<BigEndian>()?;

        let keys_len = data.read_u8()?;
        let mut keys = Vec::with_capacity(keys_len as usize);
        for _ in 0..keys_len {
            let mut key: [u8; 20] = [0; 20];
            data.read(&mut key)?;
            keys.push(key);
        }

        Ok(BodyUpdateGuardianSet {
            new_index,
            new_keys: keys,
        })
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());
        v.write_u32::<BigEndian>(self.new_index)?;
        v.write_u8(self.new_keys.len() as u8)?;

        for k in self.new_keys.iter() {
            v.write(k)?;
        }

        Ok(v.into_inner())
    }
}
