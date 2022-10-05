// TODO - fund account using Sui sdk...
// run sui-faucet
import fetch from 'node-fetch';

async function requestGas() {
  try {
    // 👇️ const response: Response
    const response = await fetch('http://127.0.0.1:5003', {
      method: 'POST',
      body: JSON.stringify("/gas"),
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Error! status: ${response.status}`);//0x312b6969b43d2ec7f421f385894a288e0eb0336c
    }
// curl -X POST -d '{"recipient": "0x312b6969b43d2ec7f421f385894a288e0eb0336c"}' -H 'Content-Type: application/json' http://127.0.0.1:5003/gas
    // 👇️ const result: CreateUserResponse
    const result = await response.json()

    console.log('result is: ', JSON.stringify(result, null, 4));

    return result;
  } catch (error) {
    if (error instanceof Error) {
      console.log('error message: ', error.message);
      return error.message;
    } else {
      console.log('unexpected error: ', error);
      return 'An unexpected error occurred';
    }
  }
}

requestGas();