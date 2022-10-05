import { Ed25519Keypair, JsonRpcProvider, RawSigner } from '@mysten/sui.js';
const { execSync } = require('child_process');
// TODO - load in existing key, instead of creating new keypair
// Generate a new Keypair
//let key = new Uint8Array(Buffer.from("e773764df3e90eb216d413f5ed0cb824f65f0191ec138a4c976709f22a413120df8b8031300ccad14927d30570c3cdbf1a14cea3a54ea454da8fa4992116031a"));
//let key = ['e7', '73', '76', '4d', 'f3', 'e9', '0e', 'b2', '16', 'd4', '13', 'f5', 'ed', '0c', 'b8', '24', 'f6', '5f', '01', '91', 'ec', '13', '8a', '4c', '97', '67', '09', 'f2', '2a', '41', '31', '20', 'df', '8b', '80', '31', '30', '0c', 'ca', 'd1', '49', '27', 'd3', '05', '70', 'c3', 'cd', 'bf', '1a', '14', 'ce', 'a3', 'a5', '4e', 'a4', '54', 'da', '8f', 'a4', '99', '21', '16', '03', '1a']console.log(key)
//let key = new Uint8Array([231,115,118,77,243,233,14,178,22,212,19,245,237,12,184,36,246,95,1,145,236,19,138,76,151,103,9,242,42,65,49,32,223,139,128,49,48,12,202,209,73,39,211,5,112,195,205,191,26,20,206,163,165,78,164,84,218,143,164,153,33,22,3,26]);
//console.log(key.length)

let public_key = [
  154,  56, 140, 134, 120, 162, 110,  52,
   30, 108, 169,  96, 215,  23, 123, 126,
  233, 138,  98,  34,  89, 117, 106,  25,
  204, 164,  65,  85,  96,  18, 208, 192
];

let secret_key = new Uint8Array([
  146,  47,  52,  79, 146, 231, 120, 242,  46, 208,   5,
  171, 112, 164, 133, 136, 157,  61,  48, 136,  84, 209,
  255,  85, 178, 190, 100, 179, 198, 127,  29, 244, 154,
   56, 140, 134, 120, 162, 110,  52,  30, 108, 169,  96,
  215,  23, 123, 126, 233, 138,  98,  34,  89, 117, 106,
   25, 204, 164,  65,  85,  96,  18, 208, 192
])


//let keypair = new Ed25519Keypair();
//console.log(keypair)
const keypair = Ed25519Keypair.fromSecretKey(secret_key);
//let x = "00e773764df3e90eb216d413f5ed0cb824f65f0191ec138a4c976709f22a413120df8b8031300ccad14927d30570c3cdbf1a14cea3a54ea454da8fa4992116031a";
//console.log(x.length)
//const keypair = new Ed25519Keypair();
console.log(keypair)
const signer = new RawSigner(
  keypair,
  //new JsonRpcProvider('https://localhost:37645') //
  new JsonRpcProvider('http://127.0.0.1:5001')
);// http://127.0.0.1:5003
async function main(){
    const packagePath = "../wormhole"

    const compiledModules = JSON.parse(
      execSync(
        `sui move build --dump-bytecode-as-base64 --path ${packagePath}`,
        { encoding: 'utf-8' }
      )
    );
    const publishTxn = await signer.publish({
      compiledModules,
      gasBudget: 10000,
    });
    console.log('publishTxn', publishTxn);
}

main()
