export function redeemVaaMsg(hexVaa: string) {
  return {
    type: "x/tokenBridge/Redeem", //TODO correct type
    value: {
      vaa: hexVaa,
    },
  };
}
