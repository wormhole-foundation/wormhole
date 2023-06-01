// run this script with truffle exec

const jsonfile = require("jsonfile");
const QueryResponseABI = jsonfile.readFileSync(
  "../build/contracts/QueryResponse.json"
).abi;
const responseBytes =
  "0x00004d2fded93c872040330a7d4a60cb4431d6c929c720437ed345daaff928f786f45fa31c825bad2714a69ca3f7d1324f8f9d51dc452fdfacff65ff4c6ad7e7390301010005000000000d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde03000000066c6174657374000000000295f40396c790fb8cb9407de03f61daa46ef15a3c20d301e09af14c850185294c07580c6477b4cf000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d57726170706564204d6174696300000000000000000000000000000000000000";
const sigs = [
  [
    "0xb8320f25a42b2b4a832e18e5b68dc12f269d7d8a8d9d43c76107d74d4d7202eb",
    "0x6d8258da2a60de95ebce1283ba6ec124a2301a864e35979ebb48af5981e0fae8",
    "0x1b", // zero add 27
    "0x00",
  ],
];
const expectedHash =
  "0x3c4628ca459c0ee5344d91146776f46627bdbf189f4f045d9dedf480861c05f3";
const expectedDigest =
  "0x616674308c1ab1b468665f21fd3808a8fc5807a4ca9859b681d2e3f7ace97cc2";

module.exports = async function(callback) {
  try {
    const QueryResponse = await artifacts.require("QueryResponse");
    //Query deploy
    const queryAddress = (
      await QueryResponse.new("0xC89Ce4735882C9F0f0FE26686c53074E09B0D550")
    ).address;

    console.log("QueryResponse deployed at: " + queryAddress);

    const initialized = new web3.eth.Contract(QueryResponseABI, queryAddress);

    const hashResult = await initialized.methods
      .getResponseHash(responseBytes)
      .call();
    console.log(hashResult);

    const digestResult = await initialized.methods
      .getResponseDigest(responseBytes)
      .call();
    console.log(digestResult);

    const verify = await initialized.methods
      .verifyQueryResponse(responseBytes, sigs)
      .call();
    console.log(verify);

    const result = await initialized.methods
      .processStringResult(responseBytes, sigs)
      .call();
    console.log(result);

    callback();
  } catch (e) {
    callback(e);
  }
};
