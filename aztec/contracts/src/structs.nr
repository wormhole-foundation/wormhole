use dep::aztec::{protocol_types::traits::{Deserialize, Packable, Serialize}, prelude::AztecAddress};
use std::meta::derive;
use dep::keccak256::keccak256;

#[derive(Deserialize, Packable, Serialize)]
pub struct Provider {
    pub chain_id: u16,
    // pub governance_chain_id: u16,
    // pub governance_contract: Field, // bytes
    pub evm_chain_id: u16,
}

#[derive(Deserialize, Packable, Serialize)]
pub struct Guardian_PK {
    pub value0: u8, // Noir does not support packable fixed-length arrays
    pub value1: u8, // see issue: https://github.com/noir-lang/noir/issues/8165
    pub value2: u8, 
    pub value3: u8, 
    pub value4: u8, 
    pub value5: u8, 
    pub value6: u8, 
    pub value7: u8, 
    pub value8: u8, 
    pub value9: u8, 
    pub value10: u8, 
    pub value11: u8, 
    pub value12: u8, 
    pub value13: u8, 
    pub value14: u8, 
    pub value15: u8, 
    pub value16: u8, 
    pub value17: u8, 
    pub value18: u8, 
    pub value19: u8, 
}

impl Guardian_PK {
    pub fn new(value: [u8; 20]) -> Self {
        Guardian_PK { 
            value0: value[0],
            value1: value[1],
            value2: value[2],
            value3: value[3],
            value4: value[4],
            value5: value[5],
            value6: value[6],
            value7: value[7],
            value8: value[8],
            value9: value[9],
            value10: value[10],
            value11: value[11],
            value12: value[12],
            value13: value[13],
            value14: value[14],
            value15: value[15],
            value16: value[16],
            value17: value[17],
            value18: value[18],
            value19: value[19],
         }
    }

    pub fn default() -> Self {
        Guardian_PK::new([0;20]) // Default to 20 bytes of zero
    }

    pub fn as_array(Guardian_PK: Guardian_PK) -> [u8; 20] {
        [Guardian_PK.value0, Guardian_PK.value1, Guardian_PK.value2, Guardian_PK.value3, Guardian_PK.value4, Guardian_PK.value5, Guardian_PK.value6, Guardian_PK.value7, Guardian_PK.value8, Guardian_PK.value9, Guardian_PK.value10, Guardian_PK.value11, Guardian_PK.value12, Guardian_PK.value13, Guardian_PK.value14, Guardian_PK.value15, Guardian_PK.value16, Guardian_PK.value17, Guardian_PK.value18, Guardian_PK.value19]
    }

    // pub fn from_signature(signature: [u8; 64], message: [u8; 32]) -> Self {
        // pk.ecrecover(signature, message);
        // TODO: NEED AN IMPLEMENTATION FOR ECRECOVER THAT DOES NOT NEED PK ALREADY
    // }
}

// Testing Guardian_PK
#[test]
fn test_should_create_guardian_pk() {
    let g = Guardian_PK::new([1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20]);
    assert_eq(g.value0, 1);
    assert_eq(g.value1, 2);
    assert_eq(g.value2, 3);
    assert_eq(g.value3, 4);
    assert_eq(g.value4, 5);
    assert_eq(g.value5, 6);
    assert_eq(g.value6, 7);
    assert_eq(g.value7, 8);
    assert_eq(g.value8, 9);
    assert_eq(g.value9, 10);
    assert_eq(g.value10, 11);
    assert_eq(g.value11, 12);
    assert_eq(g.value12, 13);
    assert_eq(g.value13, 14);
    assert_eq(g.value14, 15);
    assert_eq(g.value15, 16);
    assert_eq(g.value16, 17);
    assert_eq(g.value17, 18);
    assert_eq(g.value18, 19);
    assert_eq(g.value19, 20);
}

#[test]
fn test_should_create_default_guardian_pk() {
    let g = Guardian_PK::default();
    assert_eq(g.value0, 0);
    assert_eq(g.value1, 0);
    assert_eq(g.value2, 0);
    assert_eq(g.value3, 0);
    assert_eq(g.value4, 0);
    assert_eq(g.value5, 0);
    assert_eq(g.value6, 0);
    assert_eq(g.value7, 0);
    assert_eq(g.value8, 0);
    assert_eq(g.value9, 0);
    assert_eq(g.value10, 0);
    assert_eq(g.value11, 0);
    assert_eq(g.value12, 0);
    assert_eq(g.value13, 0);
    assert_eq(g.value14, 0);
    assert_eq(g.value15, 0);
    assert_eq(g.value16, 0);
    assert_eq(g.value17, 0);
    assert_eq(g.value18, 0);
    assert_eq(g.value19, 0);
}

#[test]
fn test_should_return_guardian_pk_as_correct_array() {
    let g = Guardian_PK::new([1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20]);
    let g_array = Guardian_PK::as_array(g);
    assert_eq(g_array, [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20]);
}

#[derive(Deserialize, Packable, Serialize)]
pub struct Guardian {
    pub address: Guardian_PK,
}

impl Guardian {
    pub fn new(address: [u8;20] ) -> Self {
        Guardian { address: Guardian_PK::new(address) }
    }
    pub fn default() -> Self {
        Guardian { address: Guardian_PK::default() } // Default to 20 bytes of zero
    }

    pub fn get_address(g: Self) -> [u8; 20] {
        Guardian_PK::as_array(g.address)
    }
}

// Testing Guardian
#[test]
fn test_create_guardian() {
    let g = Guardian::new([1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20]);
    assert_eq(g.address.value0, 1);
    assert_eq(g.address.value1, 2);
    assert_eq(g.address.value2, 3);
    assert_eq(g.address.value3, 4);
    assert_eq(g.address.value4, 5);
    assert_eq(g.address.value5, 6);
    assert_eq(g.address.value6, 7);
    assert_eq(g.address.value7, 8);
    assert_eq(g.address.value8, 9);
    assert_eq(g.address.value9, 10);
    assert_eq(g.address.value10, 11);
    assert_eq(g.address.value11, 12);
    assert_eq(g.address.value12, 13);
    assert_eq(g.address.value13, 14);
    assert_eq(g.address.value14, 15);
    assert_eq(g.address.value15, 16);
    assert_eq(g.address.value16, 17);
    assert_eq(g.address.value17, 18);
    assert_eq(g.address.value18, 19);
    assert_eq(g.address.value19, 20);
}

#[test]
fn test_get_guardian_address() {
    let g = Guardian::default();
    let address = Guardian::get_address(g);
    assert_eq(address[0], 0);
    assert_eq(address[1], 0);
    assert_eq(address[2], 0);
    assert_eq(address[3], 0);
    assert_eq(address[4], 0);
    assert_eq(address[5], 0);
    assert_eq(address[6], 0);
    assert_eq(address[7], 0);
    assert_eq(address[8], 0);
    assert_eq(address[9], 0);
    assert_eq(address[10], 0);
    assert_eq(address[11], 0);
    assert_eq(address[12], 0);
    assert_eq(address[13], 0);
    assert_eq(address[14], 0);
    assert_eq(address[15], 0);
    assert_eq(address[16], 0);
    assert_eq(address[17], 0);
    assert_eq(address[18], 0);
    assert_eq(address[19], 0);
}

#[derive(Deserialize, Serialize)]
pub struct Signature {
    pub r: [u8; 32],
    pub s: [u8; 32],
    pub v: u8,
    pub guardian_index: u8
}

impl Signature {
    pub fn new(r: [u8; 32], s: [u8; 32], v: u8, guardian_index: u8) -> Self {
        Signature { r, s, v, guardian_index }
    }

    pub fn default() -> Self {
        Signature { r: [0; 32], s: [0; 32], v: 0, guardian_index: 0 }
    }

    pub fn from_bytes(bytes: [u8; 66]) -> Self {
        let r = [bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5], bytes[6], bytes[7],
                  bytes[8], bytes[9], bytes[10], bytes[11], bytes[12], bytes[13], bytes[14], bytes[15],
                  bytes[16], bytes[17], bytes[18], bytes[19], bytes[20], bytes[21], bytes[22], bytes[23],
                  bytes[24], bytes[25], bytes[26], bytes[27], bytes[28], bytes[29], bytes[30], bytes[31]];
        let s = [bytes[32], bytes[33], bytes[34], bytes[35], bytes[36], bytes[37], bytes[38], bytes[39],
                  bytes[40], bytes[41], bytes[42], bytes[43], bytes[44], bytes[45], bytes[46], bytes[47],
                  bytes[48], bytes[49], bytes[50], bytes[51], bytes[52], bytes[53], bytes[54], bytes[55],
                  bytes[56], bytes[57], bytes[58], bytes[59], bytes[60], bytes[61], bytes[62], bytes[63]];
        // The 65th byte is the recovery id (v)
        let v = bytes[64];
        Signature { r, s, v, guardian_index: bytes[65] }
    }

    // assume using keccak256 for hashing so message is 32 bytes
    pub fn verify_signature(pub_key_x: [u8;32], pub_key_y: [u8;32], signature: [u8;64], msg: [u8;32]) -> bool {
        std::ecdsa_secp256k1::verify_signature(
            pub_key_x,
            pub_key_y,
            signature,
            msg
        )
    }
}

#[derive(Deserialize, Serialize)]
pub struct Body {
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain_id: u16,
    pub emitter_address: AztecAddress,
    pub sequence: u64,
    pub consistency_level: u8,
    pub payload: [u8; 1024], // Fixed size for byte arrays in Noir, adjust as needed
}

impl Body {
    pub fn new(
        timestamp: u32,
        nonce: u32,
        emitter_chain_id: u16,
        emitter_address: AztecAddress,
        sequence: u64,
        consistency_level: u8,
        payload: [u8; 1024],
    ) -> Self {
        Body {
            timestamp,
            nonce,
            emitter_chain_id,
            emitter_address,
            sequence,
            consistency_level,
            payload
        }
    }
    pub fn default() -> Self {
        Body {
            timestamp: 0,
            nonce: 0,
            emitter_chain_id: 0,
            emitter_address: AztecAddress::zero(),
            sequence: 0,
            consistency_level: 0,
            payload: [0; 1024],
        }
    }

    fn from_bytes(bytes: [u8; 1060]) -> Body {
        let timestamp: u32 = u32_from_u8s_le([bytes[0], bytes[1], bytes[2], bytes[3]]);
        let nonce: u32 = u32_from_u8s_le([bytes[4], bytes[5], bytes[6], bytes[7]]);
        let emitter_chain_id: u16 = u16_from_u8s_le([bytes[8], bytes[9]]);

        // parse emitter bytes into aztec address
        let emitter_address_bytes: [u8; 32] = [
            bytes[10], bytes[11], bytes[12], bytes[13], bytes[14], bytes[15], bytes[16], bytes[17],
            bytes[18], bytes[19], bytes[20], bytes[21], bytes[22], bytes[23], bytes[24], bytes[25],
            bytes[26], bytes[27], bytes[28], bytes[29], bytes[30], bytes[31], bytes[32], bytes[33],
            bytes[34], bytes[35], bytes[36], bytes[37], bytes[38], bytes[39], bytes[40], bytes[41],
        ];

        let emitter_address_field: Field = Field::from_le_bytes(emitter_address_bytes);

        let emitter_address: AztecAddress = AztecAddress {
            inner: emitter_address_field,
        };
        
        let sequence: u64 = u64_from_u8s_le([bytes[42], bytes[43], bytes[44], bytes[45], bytes[46], bytes[47], bytes[48], bytes[49]]);
        let consistency_level: u8 = bytes[50];
        let payload: [u8; 1024] = [
            bytes[51], bytes[52], bytes[53], bytes[54], bytes[55], bytes[56], bytes[57], bytes[58],
            bytes[59], bytes[60], bytes[61], bytes[62], bytes[63], bytes[64], bytes[65], bytes[66],
            bytes[67], bytes[68], bytes[69], bytes[70], bytes[71], bytes[72], bytes[73], bytes[74],
            bytes[75], bytes[76], bytes[77], bytes[78], bytes[79], bytes[80], bytes[81], bytes[82],
            bytes[83], bytes[84], bytes[85], bytes[86], bytes[87], bytes[88], bytes[89], bytes[90],
            bytes[91], bytes[92], bytes[93], bytes[94], bytes[95], bytes[96], bytes[97], bytes[98],
            bytes[99], bytes[100], bytes[101], bytes[102], bytes[103], bytes[104], bytes[105], bytes[106],
            bytes[107], bytes[108], bytes[109], bytes[110], bytes[111], bytes[112], bytes[113], bytes[114],
            bytes[115], bytes[116], bytes[117], bytes[118], bytes[119], bytes[120], bytes[121], bytes[122],
            bytes[123], bytes[124], bytes[125], bytes[126], bytes[127], bytes[128], bytes[129], bytes[130],
            bytes[131], bytes[132], bytes[133], bytes[134], bytes[135], bytes[136], bytes[137], bytes[138],
            bytes[139], bytes[140], bytes[141], bytes[142], bytes[143], bytes[144], bytes[145], bytes[146],
            bytes[147], bytes[148], bytes[149], bytes[150], bytes[151], bytes[152], bytes[153], bytes[154],
            bytes[155], bytes[156], bytes[157], bytes[158], bytes[159], bytes[160], bytes[161], bytes[162],
            bytes[163], bytes[164], bytes[165], bytes[166], bytes[167], bytes[168], bytes[169], bytes[170],
            bytes[171], bytes[172], bytes[173], bytes[174], bytes[175], bytes[176], bytes[177], bytes[178],
            bytes[179], bytes[180], bytes[181], bytes[182], bytes[183], bytes[184], bytes[185], bytes[186],
            bytes[187], bytes[188], bytes[189], bytes[190], bytes[191], bytes[192], bytes[193], bytes[194],
            bytes[195], bytes[196], bytes[197], bytes[198], bytes[199], bytes[200], bytes[201], bytes[202],
            bytes[203], bytes[204], bytes[205], bytes[206], bytes[207], bytes[208], bytes[209], bytes[210],
            bytes[211], bytes[212], bytes[213], bytes[214], bytes[215], bytes[216], bytes[217], bytes[218],
            bytes[219], bytes[220], bytes[221], bytes[222], bytes[223], bytes[224], bytes[225], bytes[226],
            bytes[227], bytes[228], bytes[229], bytes[230], bytes[231], bytes[232], bytes[233], bytes[234],
            bytes[235], bytes[236], bytes[237], bytes[238], bytes[239], bytes[240], bytes[241], bytes[242],
            bytes[243], bytes[244], bytes[245], bytes[246], bytes[247], bytes[248], bytes[249], bytes[250],
            bytes[251], bytes[252], bytes[253], bytes[254], bytes[255], bytes[256], bytes[257], bytes[258],
            bytes[259], bytes[260], bytes[261], bytes[262], bytes[263], bytes[264], bytes[265], bytes[266],
            bytes[267], bytes[268], bytes[269], bytes[270], bytes[271], bytes[272], bytes[273], bytes[274],
            bytes[275], bytes[276], bytes[277], bytes[278], bytes[279], bytes[280], bytes[281], bytes[282],
            bytes[283], bytes[284], bytes[285], bytes[286], bytes[287], bytes[288], bytes[289], bytes[290],
            bytes[291], bytes[292], bytes[293], bytes[294], bytes[295], bytes[296], bytes[297], bytes[298],
            bytes[299], bytes[300], bytes[301], bytes[302], bytes[303], bytes[304], bytes[305], bytes[306],
            bytes[307], bytes[308], bytes[309], bytes[310], bytes[311], bytes[312], bytes[313], bytes[314],
            bytes[315], bytes[316], bytes[317], bytes[318], bytes[319], bytes[320], bytes[321], bytes[322],
            bytes[323], bytes[324], bytes[325], bytes[326], bytes[327], bytes[328], bytes[329], bytes[330],
            bytes[331], bytes[332], bytes[333], bytes[334], bytes[335], bytes[336], bytes[337], bytes[338],
            bytes[339], bytes[340], bytes[341], bytes[342], bytes[343], bytes[344], bytes[345], bytes[346],
            bytes[347], bytes[348], bytes[349], bytes[350], bytes[351], bytes[352], bytes[353], bytes[354],
            bytes[355], bytes[356], bytes[357], bytes[358], bytes[359], bytes[360], bytes[361], bytes[362],
            bytes[363], bytes[364], bytes[365], bytes[366], bytes[367], bytes[368], bytes[369], bytes[370],
            bytes[371], bytes[372], bytes[373], bytes[374], bytes[375], bytes[376], bytes[377], bytes[378],
            bytes[379], bytes[380], bytes[381], bytes[382], bytes[383], bytes[384], bytes[385], bytes[386],
            bytes[387], bytes[388], bytes[389], bytes[390], bytes[391], bytes[392], bytes[393], bytes[394],
            bytes[395], bytes[396], bytes[397], bytes[398], bytes[399], bytes[400], bytes[401], bytes[402],
            bytes[403], bytes[404], bytes[405], bytes[406], bytes[407], bytes[408], bytes[409], bytes[410],
            bytes[411], bytes[412], bytes[413], bytes[414], bytes[415], bytes[416], bytes[417], bytes[418],
            bytes[419], bytes[420], bytes[421], bytes[422], bytes[423], bytes[424], bytes[425], bytes[426],
            bytes[427], bytes[428], bytes[429], bytes[430], bytes[431], bytes[432], bytes[433], bytes[434],
            bytes[435], bytes[436], bytes[437], bytes[438], bytes[439], bytes[440], bytes[441], bytes[442],
            bytes[443], bytes[444], bytes[445], bytes[446], bytes[447], bytes[448], bytes[449], bytes[450],
            bytes[451], bytes[452], bytes[453], bytes[454], bytes[455], bytes[456], bytes[457], bytes[458],
            bytes[459], bytes[460], bytes[461], bytes[462], bytes[463], bytes[464], bytes[465], bytes[466],
            bytes[467], bytes[468], bytes[469], bytes[470], bytes[471], bytes[472], bytes[473], bytes[474],
            bytes[475], bytes[476], bytes[477], bytes[478], bytes[479], bytes[480], bytes[481], bytes[482],
            bytes[483], bytes[484], bytes[485], bytes[486], bytes[487], bytes[488], bytes[489], bytes[490],
            bytes[491], bytes[492], bytes[493], bytes[494], bytes[495], bytes[496], bytes[497], bytes[498],
            bytes[499], bytes[500], bytes[501], bytes[502], bytes[503], bytes[504], bytes[505], bytes[506],
            bytes[507], bytes[508], bytes[509], bytes[510], bytes[511], bytes[512], bytes[513], bytes[514],
            bytes[515], bytes[516], bytes[517], bytes[518], bytes[519], bytes[520], bytes[521], bytes[522],
            bytes[523], bytes[524], bytes[525], bytes[526], bytes[527], bytes[528], bytes[529], bytes[530],
            bytes[531], bytes[532], bytes[533], bytes[534], bytes[535], bytes[536], bytes[537], bytes[538],
            bytes[539], bytes[540], bytes[541], bytes[542], bytes[543], bytes[544], bytes[545], bytes[546],
            bytes[547], bytes[548], bytes[549], bytes[550], bytes[551], bytes[552], bytes[553], bytes[554],
            bytes[555], bytes[556], bytes[557], bytes[558], bytes[559], bytes[560], bytes[561], bytes[562],
            bytes[563], bytes[564], bytes[565], bytes[566], bytes[567], bytes[568], bytes[569], bytes[570],
            bytes[571], bytes[572], bytes[573], bytes[574], bytes[575], bytes[576], bytes[577], bytes[578],
            bytes[579], bytes[580], bytes[581], bytes[582], bytes[583], bytes[584], bytes[585], bytes[586],
            bytes[587], bytes[588], bytes[589], bytes[590], bytes[591], bytes[592], bytes[593], bytes[594],
            bytes[595], bytes[596], bytes[597], bytes[598], bytes[599], bytes[600], bytes[601], bytes[602],
            bytes[603], bytes[604], bytes[605], bytes[606], bytes[607], bytes[608], bytes[609], bytes[610],
            bytes[611], bytes[612], bytes[613], bytes[614], bytes[615], bytes[616], bytes[617], bytes[618],
            bytes[619], bytes[620], bytes[621], bytes[622], bytes[623], bytes[624], bytes[625], bytes[626],
            bytes[627], bytes[628], bytes[629], bytes[630], bytes[631], bytes[632], bytes[633], bytes[634],
            bytes[635], bytes[636], bytes[637], bytes[638], bytes[639], bytes[640], bytes[641], bytes[642],
            bytes[643], bytes[644], bytes[645], bytes[646], bytes[647], bytes[648], bytes[649], bytes[650],
            bytes[651], bytes[652], bytes[653], bytes[654], bytes[655], bytes[656], bytes[657], bytes[658],
            bytes[659], bytes[660], bytes[661], bytes[662], bytes[663], bytes[664], bytes[665], bytes[666],
            bytes[667], bytes[668], bytes[669], bytes[670], bytes[671], bytes[672], bytes[673], bytes[674],
            bytes[675], bytes[676], bytes[677], bytes[678], bytes[679], bytes[680], bytes[681], bytes[682],
            bytes[683], bytes[684], bytes[685], bytes[686], bytes[687], bytes[688], bytes[689], bytes[690],
            bytes[691], bytes[692], bytes[693], bytes[694], bytes[695], bytes[696], bytes[697], bytes[698],
            bytes[699], bytes[700], bytes[701], bytes[702], bytes[703], bytes[704], bytes[705], bytes[706],
            bytes[707], bytes[708], bytes[709], bytes[710], bytes[711], bytes[712], bytes[713], bytes[714],
            bytes[715], bytes[716], bytes[717], bytes[718], bytes[719], bytes[720], bytes[721], bytes[722],
            bytes[723], bytes[724], bytes[725], bytes[726], bytes[727], bytes[728], bytes[729], bytes[730],
            bytes[731], bytes[732], bytes[733], bytes[734], bytes[735], bytes[736], bytes[737], bytes[738],
            bytes[739], bytes[740], bytes[741], bytes[742], bytes[743], bytes[744], bytes[745], bytes[746],
            bytes[747], bytes[748], bytes[749], bytes[750], bytes[751], bytes[752], bytes[753], bytes[754],
            bytes[755], bytes[756], bytes[757], bytes[758], bytes[759], bytes[760], bytes[761], bytes[762],
            bytes[763], bytes[764], bytes[765], bytes[766], bytes[767], bytes[768], bytes[769], bytes[770],
            bytes[771], bytes[772], bytes[773], bytes[774], bytes[775], bytes[776], bytes[777], bytes[778],
            bytes[779], bytes[780], bytes[781], bytes[782], bytes[783], bytes[784], bytes[785], bytes[786],
            bytes[787], bytes[788], bytes[789], bytes[790], bytes[791], bytes[792], bytes[793], bytes[794],
            bytes[795], bytes[796], bytes[797], bytes[798], bytes[799], bytes[800], bytes[801], bytes[802],
            bytes[803], bytes[804], bytes[805], bytes[806], bytes[807], bytes[808], bytes[809], bytes[810],
            bytes[811], bytes[812], bytes[813], bytes[814], bytes[815], bytes[816], bytes[817], bytes[818],
            bytes[819], bytes[820], bytes[821], bytes[822], bytes[823], bytes[824], bytes[825], bytes[826],
            bytes[827], bytes[828], bytes[829], bytes[830], bytes[831], bytes[832], bytes[833], bytes[834],
            bytes[835], bytes[836], bytes[837], bytes[838], bytes[839], bytes[840], bytes[841], bytes[842],
            bytes[843], bytes[844], bytes[845], bytes[846], bytes[847], bytes[848], bytes[849], bytes[850],
            bytes[851], bytes[852], bytes[853], bytes[854], bytes[855], bytes[856], bytes[857], bytes[858],
            bytes[859], bytes[860], bytes[861], bytes[862], bytes[863], bytes[864], bytes[865], bytes[866],
            bytes[867], bytes[868], bytes[869], bytes[870], bytes[871], bytes[872], bytes[873], bytes[874],
            bytes[875], bytes[876], bytes[877], bytes[878], bytes[879], bytes[880], bytes[881], bytes[882],
            bytes[883], bytes[884], bytes[885], bytes[886], bytes[887], bytes[888], bytes[889], bytes[890],
            bytes[891], bytes[892], bytes[893], bytes[894], bytes[895], bytes[896], bytes[897], bytes[898],
            bytes[899], bytes[900], bytes[901], bytes[902], bytes[903], bytes[904], bytes[905], bytes[906],
            bytes[907], bytes[908], bytes[909], bytes[910], bytes[911], bytes[912], bytes[913], bytes[914],
            bytes[915], bytes[916], bytes[917], bytes[918], bytes[919], bytes[920], bytes[921], bytes[922],
            bytes[923], bytes[924], bytes[925], bytes[926], bytes[927], bytes[928], bytes[929], bytes[930],
            bytes[931], bytes[932], bytes[933], bytes[934], bytes[935], bytes[936], bytes[937], bytes[938],
            bytes[939], bytes[940], bytes[941], bytes[942], bytes[943], bytes[944], bytes[945], bytes[946],
            bytes[947], bytes[948], bytes[949], bytes[950], bytes[951], bytes[952], bytes[953], bytes[954],
            bytes[955], bytes[956], bytes[957], bytes[958], bytes[959], bytes[960], bytes[961], bytes[962],
            bytes[963], bytes[964], bytes[965], bytes[966], bytes[967], bytes[968], bytes[969], bytes[970],
            bytes[971], bytes[972], bytes[973], bytes[974], bytes[975], bytes[976], bytes[977], bytes[978],
            bytes[979], bytes[980], bytes[981], bytes[982], bytes[983], bytes[984], bytes[985], bytes[986],
            bytes[987], bytes[988], bytes[989], bytes[990], bytes[991], bytes[992], bytes[993], bytes[994],
            bytes[995], bytes[996], bytes[997], bytes[998], bytes[999], bytes[1000], bytes[1001], bytes[1002],
            bytes[1003], bytes[1004], bytes[1005], bytes[1006], bytes[1007], bytes[1008], bytes[1009], bytes[1010],
            bytes[1011], bytes[1012], bytes[1013], bytes[1014], bytes[1015], bytes[1016], bytes[1017], bytes[1018],
            bytes[1019], bytes[1020], bytes[1021], bytes[1022], bytes[1023], bytes[1024], bytes[1025], bytes[1026],
            bytes[1027], bytes[1028], bytes[1029], bytes[1030], bytes[1031], bytes[1032], bytes[1033], bytes[1034],
            bytes[1035], bytes[1036], bytes[1037], bytes[1038], bytes[1039], bytes[1040], bytes[1041], bytes[1042],
            bytes[1043], bytes[1044], bytes[1045], bytes[1046], bytes[1047], bytes[1048], bytes[1049], bytes[1050],
            bytes[1051], bytes[1052], bytes[1053], bytes[1054], bytes[1055], bytes[1056], bytes[1057], bytes[1058],
            bytes[1059], bytes[1060], bytes[1061], bytes[1062], bytes[1063], bytes[1064], bytes[1065], bytes[1066], 
            bytes[1067], bytes[1068], bytes[1069], bytes[1070], bytes[1071], bytes[1072], bytes[1073], bytes[1074],];
        
        Body {
            timestamp,
            nonce,
            emitter_chain_id,
            emitter_address,
            sequence,
            consistency_level,
            payload
        }
    }
}

pub struct VAA {
    pub version: u8,
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain_id: u16,
    pub emitter_address: AztecAddress, // emitter address is chain-specific
    pub sequence: u64,
    pub consistency_level: u8,
    pub payload: [u8; 1024], // Fixed size for byte arrays in Noir, adjust as needed
    pub guardian_set_index: u32,
    pub signatures: [Signature; 13], // Assuming max 13 signatures
    pub hash: [u8; 32], 
}

impl VAA {
    pub fn default() -> Self {
        VAA {
            version: 0,
            timestamp: 0,
            nonce: 0,
            emitter_chain_id: 0,
            emitter_address: AztecAddress::zero(),
            sequence: 0,
            consistency_level: 0,
            payload: [0; 1024],
            guardian_set_index: 0,
            signatures: [Signature::new([0;32], [0;32], 0, 0); 13],
            hash: [0; 32]
        }
    }

    pub fn new(
        version: u8,
        timestamp: u32,
        nonce: u32,
        emitter_chain_id: u16,
        emitter_address: AztecAddress,
        sequence: u64,
        consistency_level: u8,
        payload: [u8; 1024],
        guardian_set_index: u32,
        signatures: [Signature; 13],
        hash: [u8; 32]
    ) -> Self {
        VAA {
            version,
            timestamp,
            nonce,
            emitter_chain_id,
            emitter_address,
            sequence,
            consistency_level,
            payload,
            guardian_set_index,
            signatures,
            hash
        }
    }

    pub fn get_guardian_set_index(vaa: Self) -> u32{
        vaa.guardian_set_index
    }

    pub fn get_timestamp(vaa: Self) -> u32 {
        vaa.timestamp
    }

    pub fn get_payload(vaa: Self) -> [u8; 1024] {
        vaa.payload
    }

    pub fn get_hash(vaa: Self) -> [u8; 32] {
        vaa.hash
    }

    pub fn get_emitter_chain_id(vaa: Self) -> u16 {
        vaa.emitter_chain_id
    }

    pub fn get_sequence(vaa: Self) -> u64 {
        vaa.sequence
    }

    pub fn get_consistency_level(vaa: Self) -> u8 {
        vaa.consistency_level
    }

    pub fn parse(bytes: [u8; 1918]) -> Self {
        // Assuming the bytes are in the correct order and size
        let version = bytes[0];
        assert(version == 1, "Unsupported version");
        let guardian_set_index: u32 = u32_from_u8s_le([bytes[1], bytes[2], bytes[3], bytes[4]]);
    
        let signatures_len = bytes[5];

        assert(signatures_len == 13, "Unsupported number of signatures");

        let mut signatures: [Signature; 13] = [Signature::default(); 13]; // Assuming max 13 signatures
        
        for i in 0..13 {
            let sig: Signature = Signature::from_bytes([bytes[6 + (i*66)], bytes[7 + (i*66)], bytes[8 + (i*66)], bytes[9 + (i*66)], bytes[10 + (i*66)], bytes[11 + (i*66)], bytes[12 + (i*66)], bytes[13 + (i*66)], bytes[14 + (i*66)], bytes[15 + (i*66)], bytes[16 + (i*66)], bytes[17 + (i*66)], bytes[18 + (i*66)], bytes[19 + (i*66)], bytes[20 + (i*66)], bytes[21 + (i*66)], bytes[22 + (i*66)], bytes[23 + (i*66)], bytes[24 + (i*66)], bytes[25 + (i*66)], bytes[26 + (i*66)], bytes[27 + (i*66)], bytes[28 + (i*66)], bytes[29 + (i*66)], bytes[30 + (i*66)], bytes[31 + (i*66)], bytes[32 + (i*66)], bytes[33 + (i*66)], bytes[34 + (i*66)], bytes[35 + (i*66)], bytes[36 + (i*66)], bytes[37 + (i*66)], bytes[38 + (i*66)], bytes[39 + (i*66)], bytes[40 + (i*66)], bytes[41 + (i*66)], bytes[42 + (i*66)], bytes[43 + (i*66)], bytes[44 + (i*66)], bytes[45 + (i*66)], bytes[46 + (i*66)], bytes[47 + (i*66)], bytes[48 + (i*66)], bytes[49 + (i*66)], bytes[50 + (i*66)], bytes[51 + (i*66)], bytes[52 + (i*66)], bytes[53 + (i*66)], bytes[54 + (i*66)], bytes[55 + (i*66)], bytes[56 + (i*66)], bytes[57 + (i*66)], bytes[58 + (i*66)], bytes[59 + (i*66)], bytes[60 + (i*66)], bytes[61 + (i*66)], bytes[62 + (i*66)], bytes[63 + (i*66)], bytes[64 + (i*66)], bytes[65 + (i*66)], bytes[66 + (i*66)], bytes[67 + (i*66)], bytes[68 + (i*66)], bytes[69 + (i*66)], bytes[70 + (i*66)], bytes[71 + (i*66)]]);
            signatures[i] = sig;
        }

        // body = timestamp + nonce + emitter_chain_id + emitter_address + sequence + consistency_level + payload
        let body = Body::from_bytes([
            bytes[858], bytes[859], bytes[860], bytes[861], bytes[862], bytes[863], bytes[864], bytes[865], bytes[866], bytes[867], bytes[868], bytes[869], bytes[870], bytes[871], bytes[872], bytes[873], bytes[874], bytes[875], bytes[876], bytes[877], bytes[878], bytes[879], bytes[880], bytes[881], bytes[882], bytes[883], bytes[884], bytes[885], bytes[886], bytes[887], bytes[888], bytes[889], bytes[890], bytes[891], bytes[892], bytes[893], bytes[894], bytes[895], bytes[896], bytes[897], bytes[898], bytes[899], bytes[900], bytes[901], bytes[902], bytes[903], bytes[904], bytes[905], bytes[906], bytes[907], bytes[908], bytes[909], bytes[910], bytes[911], bytes[912], bytes[913], bytes[914], bytes[915], bytes[916], bytes[917], bytes[918], bytes[919], bytes[920], bytes[921], bytes[922], bytes[923], bytes[924], bytes[925], bytes[926], bytes[927], bytes[928], bytes[929], bytes[930], bytes[931], bytes[932], bytes[933], bytes[934], bytes[935], bytes[936], bytes[937], bytes[938], bytes[939], bytes[940], bytes[941], bytes[942], bytes[943], bytes[944], bytes[945], bytes[946], bytes[947], bytes[948], bytes[949], bytes[950], bytes[951], bytes[952], bytes[953], bytes[954], bytes[955], bytes[956], bytes[957], bytes[958], bytes[959], bytes[960], bytes[961], bytes[962], bytes[963], bytes[964], bytes[965], bytes[966], bytes[967], bytes[968], bytes[969], bytes[970], bytes[971], bytes[972], bytes[973], bytes[974], bytes[975], bytes[976], bytes[977], bytes[978], bytes[979], bytes[980], bytes[981], bytes[982], bytes[983], bytes[984], bytes[985], bytes[986], bytes[987], bytes[988], bytes[989], bytes[990], bytes[991], bytes[992], bytes[993], bytes[994], bytes[995], bytes[996], bytes[997], bytes[998], bytes[999], bytes[1000], bytes[1001], bytes[1002], bytes[1003], bytes[1004], bytes[1005], bytes[1006], bytes[1007], bytes[1008], bytes[1009], bytes[1010], bytes[1011], bytes[1012], bytes[1013], bytes[1014], bytes[1015], bytes[1016], bytes[1017], bytes[1018], bytes[1019], bytes[1020], bytes[1021], bytes[1022], bytes[1023], bytes[1024], bytes[1025], bytes[1026], bytes[1027], bytes[1028], bytes[1029], bytes[1030], bytes[1031], bytes[1032], bytes[1033], bytes[1034], bytes[1035], bytes[1036], bytes[1037], bytes[1038], bytes[1039], bytes[1040], bytes[1041], bytes[1042], bytes[1043], bytes[1044], bytes[1045], bytes[1046], bytes[1047], bytes[1048], bytes[1049], bytes[1050], bytes[1051], bytes[1052], bytes[1053], bytes[1054], bytes[1055], bytes[1056], bytes[1057], bytes[1058], bytes[1059], bytes[1060], bytes[1061], bytes[1062], bytes[1063], bytes[1064], bytes[1065], bytes[1066], bytes[1067], bytes[1068], bytes[1069], bytes[1070], bytes[1071], bytes[1072], bytes[1073], bytes[1074], bytes[1075], bytes[1076], bytes[1077], bytes[1078], bytes[1079], bytes[1080], bytes[1081], bytes[1082], bytes[1083], bytes[1084], bytes[1085], bytes[1086], bytes[1087], bytes[1088], bytes[1089], bytes[1090], bytes[1091], bytes[1092], bytes[1093], bytes[1094], bytes[1095], bytes[1096], bytes[1097], bytes[1098], bytes[1099], bytes[1100], bytes[1101], bytes[1102], bytes[1103], bytes[1104], bytes[1105], bytes[1106], bytes[1107], bytes[1108], bytes[1109], bytes[1110], bytes[1111], bytes[1112], bytes[1113], bytes[1114], bytes[1115], bytes[1116], bytes[1117], bytes[1118], bytes[1119], bytes[1120], bytes[1121], bytes[1122], bytes[1123], bytes[1124], bytes[1125], bytes[1126], bytes[1127], bytes[1128], bytes[1129], bytes[1130], bytes[1131], bytes[1132], bytes[1133], bytes[1134], bytes[1135], bytes[1136], bytes[1137], bytes[1138], bytes[1139], bytes[1140], bytes[1141], bytes[1142], bytes[1143], bytes[1144], bytes[1145], bytes[1146], bytes[1147], bytes[1148], bytes[1149], bytes[1150], bytes[1151], bytes[1152], bytes[1153], bytes[1154], bytes[1155], bytes[1156], bytes[1157], bytes[1158], bytes[1159], bytes[1160], bytes[1161], bytes[1162], bytes[1163], bytes[1164], bytes[1165], bytes[1166], bytes[1167], bytes[1168], bytes[1169], bytes[1170], bytes[1171], bytes[1172], bytes[1173], bytes[1174], bytes[1175], bytes[1176], bytes[1177], bytes[1178], bytes[1179], bytes[1180], bytes[1181], bytes[1182], bytes[1183], bytes[1184], bytes[1185], bytes[1186], bytes[1187], bytes[1188], bytes[1189], bytes[1190], bytes[1191], bytes[1192], bytes[1193], bytes[1194], bytes[1195], bytes[1196], bytes[1197], bytes[1198], bytes[1199], bytes[1200], bytes[1201], bytes[1202], bytes[1203], bytes[1204], bytes[1205], bytes[1206], bytes[1207], bytes[1208], bytes[1209], bytes[1210], bytes[1211], bytes[1212], bytes[1213], bytes[1214], bytes[1215], bytes[1216], bytes[1217], bytes[1218], bytes[1219], bytes[1220], bytes[1221], bytes[1222], bytes[1223], bytes[1224], bytes[1225], bytes[1226], bytes[1227], bytes[1228], bytes[1229], bytes[1230], bytes[1231], bytes[1232], bytes[1233], bytes[1234], bytes[1235], bytes[1236], bytes[1237], bytes[1238], bytes[1239], bytes[1240], bytes[1241], bytes[1242], bytes[1243], bytes[1244], bytes[1245], bytes[1246], bytes[1247], bytes[1248], bytes[1249], bytes[1250], bytes[1251], bytes[1252], bytes[1253], bytes[1254], bytes[1255], bytes[1256], bytes[1257], bytes[1258], bytes[1259], bytes[1260], bytes[1261], bytes[1262], bytes[1263], bytes[1264], bytes[1265], bytes[1266], bytes[1267], bytes[1268], bytes[1269], bytes[1270], bytes[1271], bytes[1272], bytes[1273], bytes[1274], bytes[1275], bytes[1276], bytes[1277], bytes[1278], bytes[1279], bytes[1280], bytes[1281], bytes[1282], bytes[1283], bytes[1284], bytes[1285], bytes[1286], bytes[1287], bytes[1288], bytes[1289], bytes[1290], bytes[1291], bytes[1292], bytes[1293], bytes[1294], bytes[1295], bytes[1296], bytes[1297], bytes[1298], bytes[1299], bytes[1300], bytes[1301], bytes[1302], bytes[1303], bytes[1304], bytes[1305], bytes[1306], bytes[1307], bytes[1308], bytes[1309], bytes[1310], bytes[1311], bytes[1312], bytes[1313], bytes[1314], bytes[1315], bytes[1316], bytes[1317], bytes[1318], bytes[1319], bytes[1320], bytes[1321], bytes[1322], bytes[1323], bytes[1324], bytes[1325], bytes[1326], bytes[1327], bytes[1328], bytes[1329], bytes[1330], bytes[1331], bytes[1332], bytes[1333], bytes[1334], bytes[1335], bytes[1336], bytes[1337], bytes[1338], bytes[1339], bytes[1340], bytes[1341], bytes[1342], bytes[1343], bytes[1344], bytes[1345], bytes[1346], bytes[1347], bytes[1348], bytes[1349], bytes[1350], bytes[1351], bytes[1352], bytes[1353], bytes[1354], bytes[1355], bytes[1356], bytes[1357], bytes[1358], bytes[1359], bytes[1360], bytes[1361], bytes[1362], bytes[1363], bytes[1364], bytes[1365], bytes[1366], bytes[1367], bytes[1368], bytes[1369], bytes[1370], bytes[1371], bytes[1372], bytes[1373], bytes[1374], bytes[1375], bytes[1376], bytes[1377], bytes[1378], bytes[1379], bytes[1380], bytes[1381], bytes[1382], bytes[1383], bytes[1384], bytes[1385], bytes[1386], bytes[1387], bytes[1388], bytes[1389], bytes[1390], bytes[1391], bytes[1392], bytes[1393], bytes[1394], bytes[1395], bytes[1396], bytes[1397], bytes[1398], bytes[1399], bytes[1400], bytes[1401], bytes[1402], bytes[1403], bytes[1404], bytes[1405], bytes[1406], bytes[1407], bytes[1408], bytes[1409], bytes[1410], bytes[1411], bytes[1412], bytes[1413], bytes[1414], bytes[1415], bytes[1416], bytes[1417], bytes[1418], bytes[1419], bytes[1420], bytes[1421], bytes[1422], bytes[1423], bytes[1424], bytes[1425], bytes[1426], bytes[1427], bytes[1428], bytes[1429], bytes[1430], bytes[1431], bytes[1432], bytes[1433], bytes[1434], bytes[1435], bytes[1436], bytes[1437], bytes[1438], bytes[1439], bytes[1440], bytes[1441], bytes[1442], bytes[1443], bytes[1444], bytes[1445], bytes[1446], bytes[1447], bytes[1448], bytes[1449], bytes[1450], bytes[1451], bytes[1452], bytes[1453], bytes[1454], bytes[1455], bytes[1456], bytes[1457], bytes[1458], bytes[1459], bytes[1460], bytes[1461], bytes[1462], bytes[1463], bytes[1464], bytes[1465], bytes[1466], bytes[1467], bytes[1468], bytes[1469], bytes[1470], bytes[1471], bytes[1472], bytes[1473], bytes[1474], bytes[1475], bytes[1476], bytes[1477], bytes[1478], bytes[1479], bytes[1480], bytes[1481], bytes[1482], bytes[1483], bytes[1484], bytes[1485], bytes[1486], bytes[1487], bytes[1488], bytes[1489], bytes[1490], bytes[1491], bytes[1492], bytes[1493], bytes[1494], bytes[1495], bytes[1496], bytes[1497], bytes[1498], bytes[1499], bytes[1500], bytes[1501], bytes[1502], bytes[1503], bytes[1504], bytes[1505], bytes[1506], bytes[1507], bytes[1508], bytes[1509], bytes[1510], bytes[1511], bytes[1512], bytes[1513], bytes[1514], bytes[1515], bytes[1516], bytes[1517], bytes[1518], bytes[1519], bytes[1520], bytes[1521], bytes[1522], bytes[1523], bytes[1524], bytes[1525], bytes[1526], bytes[1527], bytes[1528], bytes[1529], bytes[1530], bytes[1531], bytes[1532], bytes[1533], bytes[1534], bytes[1535], bytes[1536], bytes[1537], bytes[1538], bytes[1539], bytes[1540], bytes[1541], bytes[1542], bytes[1543], bytes[1544], bytes[1545], bytes[1546], bytes[1547], bytes[1548], bytes[1549], bytes[1550], bytes[1551], bytes[1552], bytes[1553], bytes[1554], bytes[1555], bytes[1556], bytes[1557], bytes[1558], bytes[1559], bytes[1560], bytes[1561], bytes[1562], bytes[1563], bytes[1564], bytes[1565], bytes[1566], bytes[1567], bytes[1568], bytes[1569], bytes[1570], bytes[1571], bytes[1572], bytes[1573], bytes[1574], bytes[1575], bytes[1576], bytes[1577], bytes[1578], bytes[1579], bytes[1580], bytes[1581], bytes[1582], bytes[1583], bytes[1584], bytes[1585], bytes[1586], bytes[1587], bytes[1588], bytes[1589], bytes[1590], bytes[1591], bytes[1592], bytes[1593], bytes[1594], bytes[1595], bytes[1596], bytes[1597], bytes[1598], bytes[1599], bytes[1600], bytes[1601], bytes[1602], bytes[1603], bytes[1604], bytes[1605], bytes[1606], bytes[1607], bytes[1608], bytes[1609], bytes[1610], bytes[1611], bytes[1612], bytes[1613], bytes[1614], bytes[1615], bytes[1616], bytes[1617], bytes[1618], bytes[1619], bytes[1620], bytes[1621], bytes[1622], bytes[1623], bytes[1624], bytes[1625], bytes[1626], bytes[1627], bytes[1628], bytes[1629], bytes[1630], bytes[1631], bytes[1632], bytes[1633], bytes[1634], bytes[1635], bytes[1636], bytes[1637], bytes[1638], bytes[1639], bytes[1640], bytes[1641], bytes[1642], bytes[1643], bytes[1644], bytes[1645], bytes[1646], bytes[1647], bytes[1648], bytes[1649], bytes[1650], bytes[1651], bytes[1652], bytes[1653], bytes[1654], bytes[1655], bytes[1656], bytes[1657], bytes[1658], bytes[1659], bytes[1660], bytes[1661], bytes[1662], bytes[1663], bytes[1664], bytes[1665], bytes[1666], bytes[1667], bytes[1668], bytes[1669], bytes[1670], bytes[1671], bytes[1672], bytes[1673], bytes[1674], bytes[1675], bytes[1676], bytes[1677], bytes[1678], bytes[1679], bytes[1680], bytes[1681], bytes[1682], bytes[1683], bytes[1684], bytes[1685], bytes[1686], bytes[1687], bytes[1688], bytes[1689], bytes[1690], bytes[1691], bytes[1692], bytes[1693], bytes[1694], bytes[1695], bytes[1696], bytes[1697], bytes[1698], bytes[1699], bytes[1700], bytes[1701], bytes[1702], bytes[1703], bytes[1704], bytes[1705], bytes[1706], bytes[1707], bytes[1708], bytes[1709], bytes[1710], bytes[1711], bytes[1712], bytes[1713], bytes[1714], bytes[1715], bytes[1716], bytes[1717], bytes[1718], bytes[1719], bytes[1720], bytes[1721], bytes[1722], bytes[1723], bytes[1724], bytes[1725], bytes[1726], bytes[1727], bytes[1728], bytes[1729], bytes[1730], bytes[1731], bytes[1732], bytes[1733], bytes[1734], bytes[1735], bytes[1736], bytes[1737], bytes[1738], bytes[1739], bytes[1740], bytes[1741], bytes[1742], bytes[1743], bytes[1744], bytes[1745], bytes[1746], bytes[1747], bytes[1748], bytes[1749], bytes[1750], bytes[1751], bytes[1752], bytes[1753], bytes[1754], bytes[1755], bytes[1756], bytes[1757], bytes[1758], bytes[1759], bytes[1760], bytes[1761], bytes[1762], bytes[1763], bytes[1764], bytes[1765], bytes[1766], bytes[1767], bytes[1768], bytes[1769], bytes[1770], bytes[1771], bytes[1772], bytes[1773], bytes[1774], bytes[1775], bytes[1776], bytes[1777], bytes[1778], bytes[1779], bytes[1780], bytes[1781], bytes[1782], bytes[1783], bytes[1784], bytes[1785], bytes[1786], bytes[1787], bytes[1788], bytes[1789], bytes[1790], bytes[1791], bytes[1792], bytes[1793], bytes[1794], bytes[1795], bytes[1796], bytes[1797], bytes[1798], bytes[1799], bytes[1800], bytes[1801], bytes[1802], bytes[1803], bytes[1804], bytes[1805], bytes[1806], bytes[1807], bytes[1808], bytes[1809], bytes[1810], bytes[1811], bytes[1812], bytes[1813], bytes[1814], bytes[1815], bytes[1816], bytes[1817], bytes[1818], bytes[1819], bytes[1820], bytes[1821], bytes[1822], bytes[1823], bytes[1824], bytes[1825], bytes[1826], bytes[1827], bytes[1828], bytes[1829], bytes[1830], bytes[1831], bytes[1832], bytes[1833], bytes[1834], bytes[1835], bytes[1836], bytes[1837], bytes[1838], bytes[1839], bytes[1840], bytes[1841], bytes[1842], bytes[1843], bytes[1844], bytes[1845], bytes[1846], bytes[1847], bytes[1848], bytes[1849], bytes[1850], bytes[1851], bytes[1852], bytes[1853], bytes[1854], bytes[1855], bytes[1856], bytes[1857], bytes[1858], bytes[1859], bytes[1860], bytes[1861], bytes[1862], bytes[1863], bytes[1864], bytes[1865], bytes[1866], bytes[1867], bytes[1868], bytes[1869], bytes[1870], bytes[1871], bytes[1872], bytes[1873], bytes[1874], bytes[1875], bytes[1876], bytes[1877], bytes[1878], bytes[1879], bytes[1880], bytes[1881], bytes[1882], bytes[1883], bytes[1884], bytes[1885], bytes[1886], bytes[1887], bytes[1888], bytes[1889], bytes[1890], bytes[1891], bytes[1892], bytes[1893], bytes[1894], bytes[1895], bytes[1896], bytes[1897], bytes[1898], bytes[1899], bytes[1900], bytes[1901], bytes[1902], bytes[1903], bytes[1904], bytes[1905], bytes[1906], bytes[1907], bytes[1908], bytes[1909], bytes[1910], bytes[1911], bytes[1912], bytes[1913], bytes[1914], bytes[1915], bytes[1916], bytes[1917]
        ]);

        let hash1: [u8; 32] = keccak256(
            [
            bytes[858], bytes[859], bytes[860], bytes[861], bytes[862], bytes[863], bytes[864], bytes[865], bytes[866], bytes[867], bytes[868], bytes[869], bytes[870], bytes[871], bytes[872], bytes[873], bytes[874], bytes[875], bytes[876], bytes[877], bytes[878], bytes[879], bytes[880], bytes[881], bytes[882], bytes[883], bytes[884], bytes[885], bytes[886], bytes[887], bytes[888], bytes[889], bytes[890], bytes[891], bytes[892], bytes[893], bytes[894], bytes[895], bytes[896], bytes[897], bytes[898], bytes[899], bytes[900], bytes[901], bytes[902], bytes[903], bytes[904], bytes[905], bytes[906], bytes[907], bytes[908], bytes[909], bytes[910], bytes[911], bytes[912], bytes[913], bytes[914], bytes[915], bytes[916], bytes[917], bytes[918], bytes[919], bytes[920], bytes[921], bytes[922], bytes[923], bytes[924], bytes[925], bytes[926], bytes[927], bytes[928], bytes[929], bytes[930], bytes[931], bytes[932], bytes[933], bytes[934], bytes[935], bytes[936], bytes[937], bytes[938], bytes[939], bytes[940], bytes[941], bytes[942], bytes[943], bytes[944], bytes[945], bytes[946], bytes[947], bytes[948], bytes[949], bytes[950], bytes[951], bytes[952], bytes[953], bytes[954], bytes[955], bytes[956], bytes[957], bytes[958], bytes[959], bytes[960], bytes[961], bytes[962], bytes[963], bytes[964], bytes[965], bytes[966], bytes[967], bytes[968], bytes[969], bytes[970], bytes[971], bytes[972], bytes[973], bytes[974], bytes[975], bytes[976], bytes[977], bytes[978], bytes[979], bytes[980], bytes[981], bytes[982], bytes[983], bytes[984], bytes[985], bytes[986], bytes[987], bytes[988], bytes[989], bytes[990], bytes[991], bytes[992], bytes[993], bytes[994], bytes[995], bytes[996], bytes[997], bytes[998], bytes[999], bytes[1000], bytes[1001], bytes[1002], bytes[1003], bytes[1004], bytes[1005], bytes[1006], bytes[1007], bytes[1008], bytes[1009], bytes[1010], bytes[1011], bytes[1012], bytes[1013], bytes[1014], bytes[1015], bytes[1016], bytes[1017], bytes[1018], bytes[1019], bytes[1020], bytes[1021], bytes[1022], bytes[1023], bytes[1024], bytes[1025], bytes[1026], bytes[1027], bytes[1028], bytes[1029], bytes[1030], bytes[1031], bytes[1032], bytes[1033], bytes[1034], bytes[1035], bytes[1036], bytes[1037], bytes[1038], bytes[1039], bytes[1040], bytes[1041], bytes[1042], bytes[1043], bytes[1044], bytes[1045], bytes[1046], bytes[1047], bytes[1048], bytes[1049], bytes[1050], bytes[1051], bytes[1052], bytes[1053], bytes[1054], bytes[1055], bytes[1056], bytes[1057], bytes[1058], bytes[1059], bytes[1060], bytes[1061], bytes[1062], bytes[1063], bytes[1064], bytes[1065], bytes[1066], bytes[1067], bytes[1068], bytes[1069], bytes[1070], bytes[1071], bytes[1072], bytes[1073], bytes[1074], bytes[1075], bytes[1076], bytes[1077], bytes[1078], bytes[1079], bytes[1080], bytes[1081], bytes[1082], bytes[1083], bytes[1084], bytes[1085], bytes[1086], bytes[1087], bytes[1088], bytes[1089], bytes[1090], bytes[1091], bytes[1092], bytes[1093], bytes[1094], bytes[1095], bytes[1096], bytes[1097], bytes[1098], bytes[1099], bytes[1100], bytes[1101], bytes[1102], bytes[1103], bytes[1104], bytes[1105], bytes[1106], bytes[1107], bytes[1108], bytes[1109], bytes[1110], bytes[1111], bytes[1112], bytes[1113], bytes[1114], bytes[1115], bytes[1116], bytes[1117], bytes[1118], bytes[1119], bytes[1120], bytes[1121], bytes[1122], bytes[1123], bytes[1124], bytes[1125], bytes[1126], bytes[1127], bytes[1128], bytes[1129], bytes[1130], bytes[1131], bytes[1132], bytes[1133], bytes[1134], bytes[1135], bytes[1136], bytes[1137], bytes[1138], bytes[1139], bytes[1140], bytes[1141], bytes[1142], bytes[1143], bytes[1144], bytes[1145], bytes[1146], bytes[1147], bytes[1148], bytes[1149], bytes[1150], bytes[1151], bytes[1152], bytes[1153], bytes[1154], bytes[1155], bytes[1156], bytes[1157], bytes[1158], bytes[1159], bytes[1160], bytes[1161], bytes[1162], bytes[1163], bytes[1164], bytes[1165], bytes[1166], bytes[1167], bytes[1168], bytes[1169], bytes[1170], bytes[1171], bytes[1172], bytes[1173], bytes[1174], bytes[1175], bytes[1176], bytes[1177], bytes[1178], bytes[1179], bytes[1180], bytes[1181], bytes[1182], bytes[1183], bytes[1184], bytes[1185], bytes[1186], bytes[1187], bytes[1188], bytes[1189], bytes[1190], bytes[1191], bytes[1192], bytes[1193], bytes[1194], bytes[1195], bytes[1196], bytes[1197], bytes[1198], bytes[1199], bytes[1200], bytes[1201], bytes[1202], bytes[1203], bytes[1204], bytes[1205], bytes[1206], bytes[1207], bytes[1208], bytes[1209], bytes[1210], bytes[1211], bytes[1212], bytes[1213], bytes[1214], bytes[1215], bytes[1216], bytes[1217], bytes[1218], bytes[1219], bytes[1220], bytes[1221], bytes[1222], bytes[1223], bytes[1224], bytes[1225], bytes[1226], bytes[1227], bytes[1228], bytes[1229], bytes[1230], bytes[1231], bytes[1232], bytes[1233], bytes[1234], bytes[1235], bytes[1236], bytes[1237], bytes[1238], bytes[1239], bytes[1240], bytes[1241], bytes[1242], bytes[1243], bytes[1244], bytes[1245], bytes[1246], bytes[1247], bytes[1248], bytes[1249], bytes[1250], bytes[1251], bytes[1252], bytes[1253], bytes[1254], bytes[1255], bytes[1256], bytes[1257], bytes[1258], bytes[1259], bytes[1260], bytes[1261], bytes[1262], bytes[1263], bytes[1264], bytes[1265], bytes[1266], bytes[1267], bytes[1268], bytes[1269], bytes[1270], bytes[1271], bytes[1272], bytes[1273], bytes[1274], bytes[1275], bytes[1276], bytes[1277], bytes[1278], bytes[1279], bytes[1280], bytes[1281], bytes[1282], bytes[1283], bytes[1284], bytes[1285], bytes[1286], bytes[1287], bytes[1288], bytes[1289], bytes[1290], bytes[1291], bytes[1292], bytes[1293], bytes[1294], bytes[1295], bytes[1296], bytes[1297], bytes[1298], bytes[1299], bytes[1300], bytes[1301], bytes[1302], bytes[1303], bytes[1304], bytes[1305], bytes[1306], bytes[1307], bytes[1308], bytes[1309], bytes[1310], bytes[1311], bytes[1312], bytes[1313], bytes[1314], bytes[1315], bytes[1316], bytes[1317], bytes[1318], bytes[1319], bytes[1320], bytes[1321], bytes[1322], bytes[1323], bytes[1324], bytes[1325], bytes[1326], bytes[1327], bytes[1328], bytes[1329], bytes[1330], bytes[1331], bytes[1332], bytes[1333], bytes[1334], bytes[1335], bytes[1336], bytes[1337], bytes[1338], bytes[1339], bytes[1340], bytes[1341], bytes[1342], bytes[1343], bytes[1344], bytes[1345], bytes[1346], bytes[1347], bytes[1348], bytes[1349], bytes[1350], bytes[1351], bytes[1352], bytes[1353], bytes[1354], bytes[1355], bytes[1356], bytes[1357], bytes[1358], bytes[1359], bytes[1360], bytes[1361], bytes[1362], bytes[1363], bytes[1364], bytes[1365], bytes[1366], bytes[1367], bytes[1368], bytes[1369], bytes[1370], bytes[1371], bytes[1372], bytes[1373], bytes[1374], bytes[1375], bytes[1376], bytes[1377], bytes[1378], bytes[1379], bytes[1380], bytes[1381], bytes[1382], bytes[1383], bytes[1384], bytes[1385], bytes[1386], bytes[1387], bytes[1388], bytes[1389], bytes[1390], bytes[1391], bytes[1392], bytes[1393], bytes[1394], bytes[1395], bytes[1396], bytes[1397], bytes[1398], bytes[1399], bytes[1400], bytes[1401], bytes[1402], bytes[1403], bytes[1404], bytes[1405], bytes[1406], bytes[1407], bytes[1408], bytes[1409], bytes[1410], bytes[1411], bytes[1412], bytes[1413], bytes[1414], bytes[1415], bytes[1416], bytes[1417], bytes[1418], bytes[1419], bytes[1420], bytes[1421], bytes[1422], bytes[1423], bytes[1424], bytes[1425], bytes[1426], bytes[1427], bytes[1428], bytes[1429], bytes[1430], bytes[1431], bytes[1432], bytes[1433], bytes[1434], bytes[1435], bytes[1436], bytes[1437], bytes[1438], bytes[1439], bytes[1440], bytes[1441], bytes[1442], bytes[1443], bytes[1444], bytes[1445], bytes[1446], bytes[1447], bytes[1448], bytes[1449], bytes[1450], bytes[1451], bytes[1452], bytes[1453], bytes[1454], bytes[1455], bytes[1456], bytes[1457], bytes[1458], bytes[1459], bytes[1460], bytes[1461], bytes[1462], bytes[1463], bytes[1464], bytes[1465], bytes[1466], bytes[1467], bytes[1468], bytes[1469], bytes[1470], bytes[1471], bytes[1472], bytes[1473], bytes[1474], bytes[1475], bytes[1476], bytes[1477], bytes[1478], bytes[1479], bytes[1480], bytes[1481], bytes[1482], bytes[1483], bytes[1484], bytes[1485], bytes[1486], bytes[1487], bytes[1488], bytes[1489], bytes[1490], bytes[1491], bytes[1492], bytes[1493], bytes[1494], bytes[1495], bytes[1496], bytes[1497], bytes[1498], bytes[1499], bytes[1500], bytes[1501], bytes[1502], bytes[1503], bytes[1504], bytes[1505], bytes[1506], bytes[1507], bytes[1508], bytes[1509], bytes[1510], bytes[1511], bytes[1512], bytes[1513], bytes[1514], bytes[1515], bytes[1516], bytes[1517], bytes[1518], bytes[1519], bytes[1520], bytes[1521], bytes[1522], bytes[1523], bytes[1524], bytes[1525], bytes[1526], bytes[1527], bytes[1528], bytes[1529], bytes[1530], bytes[1531], bytes[1532], bytes[1533], bytes[1534], bytes[1535], bytes[1536], bytes[1537], bytes[1538], bytes[1539], bytes[1540], bytes[1541], bytes[1542], bytes[1543], bytes[1544], bytes[1545], bytes[1546], bytes[1547], bytes[1548], bytes[1549], bytes[1550], bytes[1551], bytes[1552], bytes[1553], bytes[1554], bytes[1555], bytes[1556], bytes[1557], bytes[1558], bytes[1559], bytes[1560], bytes[1561], bytes[1562], bytes[1563], bytes[1564], bytes[1565], bytes[1566], bytes[1567], bytes[1568], bytes[1569], bytes[1570], bytes[1571], bytes[1572], bytes[1573], bytes[1574], bytes[1575], bytes[1576], bytes[1577], bytes[1578], bytes[1579], bytes[1580], bytes[1581], bytes[1582], bytes[1583], bytes[1584], bytes[1585], bytes[1586], bytes[1587], bytes[1588], bytes[1589], bytes[1590], bytes[1591], bytes[1592], bytes[1593], bytes[1594], bytes[1595], bytes[1596], bytes[1597], bytes[1598], bytes[1599], bytes[1600], bytes[1601], bytes[1602], bytes[1603], bytes[1604], bytes[1605], bytes[1606], bytes[1607], bytes[1608], bytes[1609], bytes[1610], bytes[1611], bytes[1612], bytes[1613], bytes[1614], bytes[1615], bytes[1616], bytes[1617], bytes[1618], bytes[1619], bytes[1620], bytes[1621], bytes[1622], bytes[1623], bytes[1624], bytes[1625], bytes[1626], bytes[1627], bytes[1628], bytes[1629], bytes[1630], bytes[1631], bytes[1632], bytes[1633], bytes[1634], bytes[1635], bytes[1636], bytes[1637], bytes[1638], bytes[1639], bytes[1640], bytes[1641], bytes[1642], bytes[1643], bytes[1644], bytes[1645], bytes[1646], bytes[1647], bytes[1648], bytes[1649], bytes[1650], bytes[1651], bytes[1652], bytes[1653], bytes[1654], bytes[1655], bytes[1656], bytes[1657], bytes[1658], bytes[1659], bytes[1660], bytes[1661], bytes[1662], bytes[1663], bytes[1664], bytes[1665], bytes[1666], bytes[1667], bytes[1668], bytes[1669], bytes[1670], bytes[1671], bytes[1672], bytes[1673], bytes[1674], bytes[1675], bytes[1676], bytes[1677], bytes[1678], bytes[1679], bytes[1680], bytes[1681], bytes[1682], bytes[1683], bytes[1684], bytes[1685], bytes[1686], bytes[1687], bytes[1688], bytes[1689], bytes[1690], bytes[1691], bytes[1692], bytes[1693], bytes[1694], bytes[1695], bytes[1696], bytes[1697], bytes[1698], bytes[1699], bytes[1700], bytes[1701], bytes[1702], bytes[1703], bytes[1704], bytes[1705], bytes[1706], bytes[1707], bytes[1708], bytes[1709], bytes[1710], bytes[1711], bytes[1712], bytes[1713], bytes[1714], bytes[1715], bytes[1716], bytes[1717], bytes[1718], bytes[1719], bytes[1720], bytes[1721], bytes[1722], bytes[1723], bytes[1724], bytes[1725], bytes[1726], bytes[1727], bytes[1728], bytes[1729], bytes[1730], bytes[1731], bytes[1732], bytes[1733], bytes[1734], bytes[1735], bytes[1736], bytes[1737], bytes[1738], bytes[1739], bytes[1740], bytes[1741], bytes[1742], bytes[1743], bytes[1744], bytes[1745], bytes[1746], bytes[1747], bytes[1748], bytes[1749], bytes[1750], bytes[1751], bytes[1752], bytes[1753], bytes[1754], bytes[1755], bytes[1756], bytes[1757], bytes[1758], bytes[1759], bytes[1760], bytes[1761], bytes[1762], bytes[1763], bytes[1764], bytes[1765], bytes[1766], bytes[1767], bytes[1768], bytes[1769], bytes[1770], bytes[1771], bytes[1772], bytes[1773], bytes[1774], bytes[1775], bytes[1776], bytes[1777], bytes[1778], bytes[1779], bytes[1780], bytes[1781], bytes[1782], bytes[1783], bytes[1784], bytes[1785], bytes[1786], bytes[1787], bytes[1788], bytes[1789], bytes[1790], bytes[1791], bytes[1792], bytes[1793], bytes[1794], bytes[1795], bytes[1796], bytes[1797], bytes[1798], bytes[1799], bytes[1800], bytes[1801], bytes[1802], bytes[1803], bytes[1804], bytes[1805], bytes[1806], bytes[1807], bytes[1808], bytes[1809], bytes[1810], bytes[1811], bytes[1812], bytes[1813], bytes[1814], bytes[1815], bytes[1816], bytes[1817], bytes[1818], bytes[1819], bytes[1820], bytes[1821], bytes[1822], bytes[1823], bytes[1824], bytes[1825], bytes[1826], bytes[1827], bytes[1828], bytes[1829], bytes[1830], bytes[1831], bytes[1832], bytes[1833], bytes[1834], bytes[1835], bytes[1836], bytes[1837], bytes[1838], bytes[1839], bytes[1840], bytes[1841], bytes[1842], bytes[1843], bytes[1844], bytes[1845], bytes[1846], bytes[1847], bytes[1848], bytes[1849], bytes[1850], bytes[1851], bytes[1852], bytes[1853], bytes[1854], bytes[1855], bytes[1856], bytes[1857], bytes[1858], bytes[1859], bytes[1860], bytes[1861], bytes[1862], bytes[1863], bytes[1864], bytes[1865], bytes[1866], bytes[1867], bytes[1868], bytes[1869], bytes[1870], bytes[1871], bytes[1872], bytes[1873], bytes[1874], bytes[1875], bytes[1876], bytes[1877], bytes[1878], bytes[1879], bytes[1880], bytes[1881], bytes[1882], bytes[1883], bytes[1884], bytes[1885], bytes[1886], bytes[1887], bytes[1888], bytes[1889], bytes[1890], bytes[1891], bytes[1892], bytes[1893], bytes[1894], bytes[1895], bytes[1896], bytes[1897], bytes[1898], bytes[1899], bytes[1900], bytes[1901], bytes[1902], bytes[1903], bytes[1904], bytes[1905], bytes[1906], bytes[1907], bytes[1908], bytes[1909], bytes[1910], bytes[1911], bytes[1912], bytes[1913], bytes[1914], bytes[1915], bytes[1916], bytes[1917]
            ],
            1060
        );

        // hash = keccak256(keccak256(body));
        let hash : [u8; 32] = keccak256(
            [hash1[0], hash1[1], hash1[2], hash1[3], hash1[4], hash1[5], hash1[6], hash1[7], hash1[8], hash1[9], hash1[10], hash1[11], hash1[12], hash1[13], hash1[14], hash1[15], hash1[16], hash1[17], hash1[18], hash1[19], hash1[20], hash1[21], hash1[22], hash1[23], hash1[24], hash1[25], hash1[26], hash1[27], hash1[28], hash1[29], hash1[30], hash1[31]],
            32
        );
            
        VAA::new(
            version,
            body.timestamp,
            body.nonce,
            body.emitter_chain_id,
            body.emitter_address,
            body.sequence,
            body.consistency_level,
            body.payload,
            guardian_set_index,
            signatures,
            hash
        )
    }

    // TODO - verify VAA signatures
    // fn verify(vaa: Self, guardian_set: GuardianSet) {
        // // CALLED FROM STATE so must check that gs is active otherwise throw error
        // let guardians: [Guardian; 19] = guardian_set.get_guardians();
        // let hash: [u8; 32] = vaa.hash;

        // let mut last_idx = 0;

        // let signatures: [Signature; 13] = vaa.signatures;

        // // verify each signature
        // for i in 0..13 {
        //     let sig_r = signatures[i].r;
        //     let sig_s = signatures[i].s;
        //     let recovery_id = signatures[i].v;
        //     let guardian_idx = signatures[i].guardian_index;

        //     assert(guardian_idx > last_idx, "guardian index is not increasing");
        //     last_idx = guardian_idx;



            // let guardian: Guardian = guardians[guardian_idx as usize];
            // let guardian_address: [u8; 20] = guardian.get_address();

            // TODO: check signature in guardian matches guardian pk generated from message + recovery id + signature
        // }
}

pub fn u32_from_u8s_le(bytes: [u8; 4]) -> u32 {
    let mut result = 0;
    for i in 0..4 {
        result |= (bytes[i as u32] as u32) << (i * 8);
    }
    result
}

pub fn u16_from_u8s_le(bytes: [u8; 2]) -> u16 {
    let mut result = 0;
    for i in 0..2 {
        result |= (bytes[i as u32] as u16) << (i * 8);
    }
    result
}

pub fn u64_from_u8s_le(bytes: [u8; 8]) -> u64 {
    let mut result = 0;
    for i in 0..8 {
        result |= (bytes[i as u32] as u64) << (i * 8);
    }
    result
}

pub struct ContractUpgrade {
    module: [u8; 32],
    action: u8,
    chain: u16,
    new_contract: Field // Addresses in Noir are typically represented as Field elements
}

pub struct GuardianSetUpgrade {
    module: [u8; 32],
    action: u8,
    chain: u16,
    // new_guardian_set: GuardianSet,
    new_guardian_set_index: u32
}

pub struct SetMessageFee {
    module: [u8; 32],
    action: u8,
    chain: u16,
    message_fee: Field
}

pub struct TransferFees {
    module: [u8; 32],
    action: u8,
    chain: u16,
    amount: Field,
    recipient: [u8; 32]
}

pub struct RecoverChainId {
    module: [u8; 32],
    action: u8,
    evm_chain_id: Field,
    new_chain_id: u16
}

#[derive(Deserialize, Packable, Serialize)]
pub struct EmitterRegistry {
    pub next_id: u64,
}

pub struct EmitterCapability {
    pub emitter: u64,
    pub sequence: u64,
}

impl EmitterRegistry {
    pub fn new() -> Self {
        EmitterRegistry { next_id: 0 }
    }

    pub fn get_next_id(&mut self) -> u64 {
        let id = self.next_id;
        self.next_id += 1;
        id
    }
}

impl EmitterCapability {
    pub fn new(registry: &mut EmitterRegistry) -> Self {
        let emitter = registry.get_next_id();
        registry.next_id += 1;
        EmitterCapability { emitter, sequence: 0 }
    }

    pub fn use_sequence(&mut self) -> u64 {
        let sequence = self.sequence;
        self.sequence += 1;
        sequence
    }
}

#[derive(Deserialize, Packable, Serialize)]
pub struct WormholeStorage {
    pub provider: Provider,
    pub guardian_set_index: u64,
    pub guardian_set_expiry: u64,
    pub message_fee: u128,
    pub emitter_registry: EmitterRegistry,
}

impl WormholeStorage {
    pub fn init(provider: Provider) -> Self {
        WormholeStorage {
            provider: provider,
            guardian_set_index: 0,
            guardian_set_expiry: 86400,
            message_fee: 1,
            emitter_registry: EmitterRegistry::new(),
        }
    }

    // Getters

    pub fn get_provider(state: Self) -> Provider {
        state.provider
    }

    pub fn get_message_fee(state: Self) -> u128 {
        state.message_fee
    }

    pub fn get_chain_id(state: Self) -> u16 {
        state.provider.chain_id
    }

    pub fn get_guardian_set_index(state: Self) -> u64 {
        state.guardian_set_index
    }

    // Setters

    pub fn set_guardian_set_expiry(mut state: Self, guardian_set_expiry: u64) -> Self {
        state.guardian_set_expiry = guardian_set_expiry;
        state
    }
    pub fn get_guardian_set_expiry(state: Self) -> u64 {
        state.guardian_set_expiry
    }

    pub fn set_guardian_set_index(mut state: Self, guardian_set_index: u64) -> Self {
        state.guardian_set_index = guardian_set_index;
        state
    }

    pub fn set_chain_id(mut state: Self, chain_id: u16) -> Self {
        state.provider.chain_id = chain_id;
        state
    }

    pub fn set_provider(mut state: Self, provider: Provider) {
        state.provider = provider;
    }

    pub fn set_message_fee(mut state: Self, message_fee: u128) -> Self {
        state.message_fee = message_fee;
        state
    }
}
