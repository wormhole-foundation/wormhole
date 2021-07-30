import { ethers } from "ethers";
import { formatUnits, parseUnits } from "ethers/lib/utils";
import { Bridge__factory, TokenImplementation__factory } from "../ethers-contracts";
import { ChainId, CHAIN_ID_ETH, ETH_TOKEN_BRIDGE_ADDRESS } from "./consts";

// TODO: this should probably be extended from the context somehow so that the signatures match
// TODO: allow for / handle cancellation?
// TODO: overall better input checking and error handling
function transferFromEth(provider: ethers.providers.Web3Provider | undefined, tokenAddress: string, amount: string, recipientChain: ChainId, recipientAddress: Uint8Array | undefined) {
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

const transferFrom = {
  [CHAIN_ID_ETH]: transferFromEth
}

export default transferFrom