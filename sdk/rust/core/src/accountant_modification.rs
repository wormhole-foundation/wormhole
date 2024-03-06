use serde::{Deserialize, Deserializer, Serialize, Serializer};

#[repr(u8)]
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash, Copy)]
pub enum ModificationKind {
    Unknown = 0,
    Add = 1,
    Subtract = 2,
}

impl From<u8> for ModificationKind {
    fn from(other: u8) -> ModificationKind {
        match other {
            1 => ModificationKind::Add,
            2 => ModificationKind::Subtract,
            _ => ModificationKind::Unknown,
        }
    }
}

impl From<ModificationKind> for u8 {
    fn from(other: ModificationKind) -> u8 {
        match other {
            ModificationKind::Unknown => 0,
            ModificationKind::Add => 1,
            ModificationKind::Subtract => 2,
        }
    }
}

impl Serialize for ModificationKind {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_u8((*self).into())
    }
}

impl<'de> Deserialize<'de> for ModificationKind {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        <u8 as Deserialize>::deserialize(deserializer).map(Self::from)
    }
}
