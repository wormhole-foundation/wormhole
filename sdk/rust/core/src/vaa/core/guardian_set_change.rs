use {
    super::Action,
    crate::{
        vaa::parse_fixed,
        GovHeader,
    },
    nom::{
        combinator::flat_map,
        multi::count,
        number::{
            complete::{
                u32,
                u8,
            },
            Endianness,
        },
        IResult,
    },
};


#[derive(Debug, PartialEq, Eq)]
pub struct GuardianSetChange {
    pub header:                 GovHeader,
    pub new_guardian_set_index: u32,
    pub new_guardian_set:       Vec<[u8; 20]>,
}

impl GuardianSetChange {
    #[inline]
    pub fn parse(i: &[u8], header: GovHeader) -> IResult<&[u8], Action> {
        let (i, new_guardian_set_index) = u32(Endianness::Big)(i)?;
        let (i, new_guardian_set) = flat_map(u8, |c| count(parse_fixed, c.into()))(i)?;
        Ok((
            i,
            Action::GuardianSetChange(Self {
                header,
                new_guardian_set_index,
                new_guardian_set,
            }),
        ))
    }
}
