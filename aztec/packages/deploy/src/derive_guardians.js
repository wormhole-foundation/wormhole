// derive_guardians_fixed.js
import secp256k1 from 'secp256k1';
import pkg from 'js-sha3';
const { keccak256 } = pkg;

const guardianData = [
  {
    "name": "guardian-0",
    "public": "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
    "private": "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0"
  },
  {
    "name": "guardian-1",
    "public": "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c",
    "private": "c3b2e45c422a1602333a64078aeb42637370b0f48fe385f9cfa6ad54a8e0c47e"
  },
  {
    "name": "guardian-2",
    "public": "0x58076F561CC62A47087B567C86f986426dFCD000",
    "private": "9f790d3f08bc4b5cd910d4278f3deb406e57bb5e924906ccd52052bb078ccd47"
  },
  {
    "name": "guardian-3",
    "public": "0xBd6e9833490F8fA87c733A183CD076a6cBD29074",
    "private": "b20cc49d6f2c82a5e6519015fc18aa3e562867f85f872c58f1277cfbd2a0c8e4"
  },
  {
    "name": "guardian-4",
    "public": "0xb853FCF0a5C78C1b56D15fCE7a154e6ebe9ED7a2",
    "private": "eded5a2fdcb5bbbfa5b07f2a91393813420e7ac30a72fc935b6df36f8294b855"
  },
  {
    "name": "guardian-5",
    "public": "0xAF3503dBD2E37518ab04D7CE78b630F98b15b78a",
    "private": "00d39587c3556f289677a837c7f3c0817cb7541ce6e38a243a4bdc761d534c5e"
  },
  {
    "name": "guardian-6",
    "public": "0x785632deA5609064803B1c8EA8bB2c77a6004Bd1",
    "private": "da534d61a8da77b232f3a2cee55c0125e2b3e33a5cd8247f3fe9e72379445c3b"
  },
  {
    "name": "guardian-7",
    "public": "0x09a281a698C0F5BA31f158585B41F4f33659e54D",
    "private": "cdbabfc2118eb00bc62c88845f3bbd03cb67a9e18a055101588ca9b36387006c"
  },
  {
    "name": "guardian-8",
    "public": "0x3178443AB76a60E21690DBfB17f7F59F09Ae3Ea1",
    "private": "c83d36423820e7350428dc4abe645cb2904459b7d7128adefe16472fdac397ba"
  },
  {
    "name": "guardian-9",
    "public": "0x647ec26ae49b14060660504f4DA1c2059E1C5Ab6",
    "private": "1cbf4e1388b81c9020500fefc83a7a81f707091bb899074db1bfce4537428112"
  },
  {
    "name": "guardian-10",
    "public": "0x810AC3D8E1258Bd2F004a94Ca0cd4c68Fc1C0611",
    "private": "17646a6ba14a541957fc7112cc973c0b3f04fce59484a92c09bb45a0b57eb740"
  },
  {
    "name": "guardian-11",
    "public": "0x80610e96d645b12f47ae5cf4546b18538739e90F",
    "private": "eb94ff04accbfc8195d44b45e7c7da4c6993b2fbbfc4ef166a7675a905df9891"
  },
  {
    "name": "guardian-12",
    "public": "0x2edb0D8530E31A218E72B9480202AcBaeB06178d",
    "private": "053a6527124b309d914a47f5257a995e9b0ad17f14659f90ed42af5e6e262b6a"
  },
  {
    "name": "guardian-13",
    "public": "0xa78858e5e5c4705CdD4B668FFe3Be5bae4867c9D",
    "private": "3fbf1e46f6da69e62aed5670f279e818889aa7d8f1beb7fd730770fd4f8ea3d7"
  },
  {
    "name": "guardian-14",
    "public": "0x5Efe3A05Efc62D60e1D19fAeB56A80223CDd3472",
    "private": "53b05697596ba04067e40be8100c9194cbae59c90e7870997de57337497172e9"
  },
  {
    "name": "guardian-15",
    "public": "0xD791b7D32C05aBB1cc00b6381FA0c4928f0c56fC",
    "private": "4e95cb2ff3f7d5e963631ad85c28b1b79cb370f21c67cbdd4c2ffb0bf664aa06"
  },
  {
    "name": "guardian-16",
    "public": "0x14Bc029B8809069093D712A3fd4DfAb31963597e",
    "private": "01b8c448ce2c1d43cfc5938d3a57086f88e3dc43bb8b08028ecb7a7924f4676f"
  },
  {
    "name": "guardian-17",
    "public": "0x246Ab29FC6EBeDf2D392a51ab2Dc5C59d0902A03",
    "private": "1db31a6ba3bcd54d2e8a64f8a2415064265d291593450c6eb7e9a6a986bd9400"
  },
  {
    "name": "guardian-18",
    "public": "0x132A84dFD920b35a3D0BA5f7A0635dF298F9033e",
    "private": "70d8f1c9534a0ab61a020366b831a494057a289441c07be67e4288c44bc6cd5d"
  }
];

function derivedAddressFromPubKey(pubKey) {
  // Remove 0x04 prefix, hash with keccak256, take last 20 bytes  
  const publicKeyBytes = pubKey.slice(1); // Remove 0x04 prefix
  const hash = keccak256(publicKeyBytes);
  const hashBuffer = Buffer.from(hash, 'hex');
  return hashBuffer.slice(-20); // Last 20 bytes for Ethereum address
}

console.log('// Generated Guardian Public Keys with Verification\n');
console.log('[');

let allValid = true;

guardianData.forEach((guardian, index) => {
  try {
    const privKeyBuffer = Buffer.from(guardian.private, 'hex');
    
    // Validate private key length
    if (privKeyBuffer.length !== 32) {
      throw new Error(`Invalid private key length: ${privKeyBuffer.length}`);
    }
    
    const pubKey = secp256k1.publicKeyCreate(privKeyBuffer, false); // uncompressed format
    
    // Verify the derived address matches the expected address
    const derivedAddr = derivedAddressFromPubKey(pubKey);
    const expectedAddr = Buffer.from(guardian.public.slice(2), 'hex');
    
    if (!derivedAddr.equals(expectedAddr)) {
      console.error(`❌ Guardian ${index}: Address mismatch!`);
      console.error(`   Expected: ${guardian.public}`);
      console.error(`   Derived:  0x${derivedAddr.toString('hex')}`);
      allValid = false;
    } else {
      console.log(`        // ✅ Guardian ${index}: ${guardian.public} (verified)`);
    }
    
    // Remove the first byte (0x04) and split into X and Y coordinates
    const x = pubKey.slice(1, 33); // X coordinate (32 bytes)
    const y = pubKey.slice(33, 65); // Y coordinate (32 bytes)
    
    // Convert address to byte array
    const addressHex = guardian.public.slice(2); // Remove '0x'
    const addressBytes = Buffer.from(addressHex, 'hex');
    
    console.log(`        Guardian::new(`);
    console.log(`            [${Array.from(addressBytes).map(b => `0x${b.toString(16).padStart(2, '0')}`).join(', ')}],`);
    console.log(`            // Public key X`);
    console.log(`            [${Array.from(x).map(b => `0x${b.toString(16).padStart(2, '0')}`).join(', ')}],`);
    console.log(`            // Public key Y`);
    console.log(`            [${Array.from(y).map(b => `0x${b.toString(16).padStart(2, '0')}`).join(', ')}]`);
    console.log(`        )${index < guardianData.length - 1 ? ',' : ''}`);
    
  } catch (error) {
    console.error(`❌ Error processing guardian ${index}:`, error);
    allValid = false;
  }
});

console.log('    ]');
console.log('\n// Verification Summary:');
console.log(`// Total guardians: ${guardianData.length}`);
console.log(`// All addresses verified: ${allValid ? '✅ YES' : '❌ NO'}`);

if (allValid) {
  console.log('\n// 🎉 All guardian keys are valid! Copy the array above into your Rust code.');
} else {
  console.log('\n// ⚠️  Some guardian keys failed verification. Check the errors above.');
}