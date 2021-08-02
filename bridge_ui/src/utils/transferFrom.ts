import { ethers } from "ethers";
import { arrayify, formatUnits, parseUnits } from "ethers/lib/utils";
import { Bridge__factory, TokenImplementation__factory } from "../ethers-contracts";
import { ChainId, CHAIN_ID_ETH, CHAIN_ID_SOLANA, ETH_TOKEN_BRIDGE_ADDRESS, SOL_TOKEN_BRIDGE_ADDRESS } from "./consts";

// TODO: this should probably be extended from the context somehow so that the signatures match
// TODO: allow for / handle cancellation?
// TODO: overall better input checking and error handling
export function transferFromEth(provider: ethers.providers.Web3Provider | undefined, tokenAddress: string, amount: string, recipientChain: ChainId, recipientAddress: Uint8Array | undefined) {
  if (!provider || !recipientAddress) return;
  const signer = provider.getSigner();
  if (!signer) return;
  //TODO: check if token attestation exists on the target chain
  //TODO: don't hardcode, fetch decimals / share them with balance, how do we determine recipient chain?
  //TODO: more catches
  const amountParsed = parseUnits(amount, 18);
  signer.getAddress().then((signerAddress) => {
    console.log("Signer:", signerAddress);
    console.log("Token:", tokenAddress)
    const token = TokenImplementation__factory.connect(
      tokenAddress,
      signer
    );
    token
      .allowance(signerAddress, ETH_TOKEN_BRIDGE_ADDRESS)
      .then((allowance) => {
        console.log("Allowance", allowance.toString()); //TODO: should we check that this is zero and warn if it isn't?
        token
          .approve(ETH_TOKEN_BRIDGE_ADDRESS, amountParsed)
          .then((transaction) => {
            console.log(transaction);
            const fee = 0; // for now, this won't do anything, we may add later
            const nonceConst = Math.random() * 100000;
            const nonceBuffer = Buffer.alloc(4);
            nonceBuffer.writeUInt32LE(nonceConst, 0);
            console.log("Initiating transfer");
            console.log("Amount:", formatUnits(amountParsed, 18));
            console.log("To chain:", recipientChain);
            console.log("To address:", recipientAddress);
            console.log("Fees:", fee);
            console.log("Nonce:", nonceBuffer);
            const bridge = Bridge__factory.connect(
              ETH_TOKEN_BRIDGE_ADDRESS,
              signer
            );
            bridge
              .transferTokens(
                tokenAddress,
                amountParsed,
                recipientChain,
                recipientAddress,
                fee,
                nonceBuffer
              )
              .then((v) => console.log("Success:", v))
              .catch((r) => console.error(r)); //TODO: integrate toast messages
          });
      });
  });
}

// TODO: need to check transfer native vs transfer wrapped
// TODO: switch out targetProvider for generic address (this likely involves getting these in their respective contexts)
export function transferFromSolana(fromAddress: string | undefined, tokenAddress: string, amount: string, targetProvider: ethers.providers.Web3Provider | undefined, targetChain: ChainId) {
  if (!fromAddress || !targetProvider) return;
  const targetSigner = targetProvider.getSigner();
  if (!targetSigner) return;
  targetSigner.getAddress().then(targetAddressStr => {
    const targetAddress = arrayify(targetAddressStr)
    const nonceConst = Math.random() * 100000;
    const nonceBuffer = Buffer.alloc(4);
    nonceBuffer.writeUInt32LE(nonceConst, 0);
    const nonce = nonceBuffer.readUInt32LE(0)
    // TODO: check decimals
    // should we avoid BigInt?
    const amountParsed = BigInt(amount)
    const fee = BigInt(0)  // for now, this won't do anything, we may add later
    console.log('bridge:',SOL_TOKEN_BRIDGE_ADDRESS)
    console.log('from:',fromAddress)
    console.log('token:',tokenAddress)
    console.log('nonce:',nonce)
    console.log('amount:',amountParsed)
    console.log('fee:',fee)
    console.log('target:',targetAddressStr,targetAddress)
    console.log('chain:',targetChain)
    // TODO: program_id vs bridge_id?
    import("token-bridge").then(({transfer_native_ix})=>{
      const ix = transfer_native_ix(SOL_TOKEN_BRIDGE_ADDRESS,SOL_TOKEN_BRIDGE_ADDRESS,fromAddress,fromAddress,tokenAddress,nonce,amountParsed,fee,targetAddress,targetChain)
      console.log(ix)
    })
  })
}

const transferFrom = {
  [CHAIN_ID_ETH]: transferFromEth,
  [CHAIN_ID_SOLANA]: transferFromSolana
}

export default transferFrom