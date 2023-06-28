// run this script with truffle exec
//  node_modules/.bin/truffle exec scripts/test_query.js

const jsonfile = require("jsonfile");
const QueryResponseABI = jsonfile.readFileSync(
  "../build/contracts/QueryResponse.json"
).abi;

const responseBytes =
  "0x010000ff0c222dc9e3655ec38e212e9792bf1860356d1277462b6bf747db865caca6fc08e6317b64ee3245264e371146b1d315d38c867fe1f69614368dc4430bb560f2000000005301dd9914c6010005010000004600000009307832613631616334020d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde030d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000418160ddd01000501000000b90000000002a61ac4c1adff9f6e180309e7d0d94c063338ddc61c1c4474cd6957c960efe659534d040005ff312e4f90c002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d57726170706564204d6174696300000000000000000000000000000000000000000000200000000000000000000000000000000000000000007ae5649beabeddf889364a";

const sigBytes =
  "ba36cd576a0f9a8a37ec5ea6a174857922f2f170cd7ec62edcbe74b1cc7258d301e8690cfd627e608d63b5d165e2190ba081bb84f5cf473fd353109e152f72fa00";
const sigs = [
  [
    "0x" + sigBytes.substring(0, 64),
    "0x" + sigBytes.substring(64, 128),
    "0x" + (parseInt(sigBytes.substring(128, 130), 16) + 27).toString(16), // last byte plus magic 27
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
    console.log("hash:", hashResult);

    const digestResult = await initialized.methods
      .getResponseDigest(responseBytes)
      .call();
    console.log("digest:", digestResult);

    const verify = await initialized.methods
      .verifyQueryResponseSignatures(responseBytes, sigs)
      .call();
    console.log("verify result:", verify);

    const response = await initialized.methods
      .parseAndVerifyQueryResponse(responseBytes, sigs)
      .call();
    console.log("response:", response);

    for (let idx = 0; idx < response.responses.length; ++idx) {
      const pcr = response.responses[idx];
      if (pcr.queryType !== "1") {
        console.error(
          "eth query result" + idx + " has an invalid query type:",
          pcr.queryType
        );
      } else {
        const ethResult = await initialized.methods
          .parseEthCallQueryResponse(pcr)
          .call();
        console.log("eth query result" + idx + ":", ethResult);
      }
    }

    console.log("Test complete");

    callback();
  } catch (e) {
    callback(e);
  }
};
