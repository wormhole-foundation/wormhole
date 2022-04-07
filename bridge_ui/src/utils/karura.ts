import axios from "axios";

export async function getKaruraGasParams(rpc: string): Promise<{
  gasPrice: number;
  gasLimit: number;
}> {
  const gasLimit = 21000000;
  const storageLimit = 64001;
  const res = (
    await axios.post(rpc, {
      id: 0,
      jsonrpc: "2.0",
      method: "eth_getEthGas",
      params: [
        {
          gasLimit,
          storageLimit,
        },
      ],
    })
  ).data.result;

  return {
    gasLimit: parseInt(res.gasLimit, 16),
    gasPrice: parseInt(res.gasPrice, 16),
  };
}
