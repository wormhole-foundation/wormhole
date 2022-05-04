require("dotenv").config({ path: ".env" });
const {
  AccountId,
  PrivateKey,
  Client,
  ContractId,
  ContractCallQuery,
  ContractExecuteTransaction,
  ContractFunctionParameters,
  ContractInfoQuery,
  Hbar,
  TokenId,
} = require("@hashgraph/sdk");
const Web3 = require("web3");
const web3 = new Web3("ws://localhost:8545");
const axios = require("axios");

// Configure accounts and client
const operatorId = AccountId.fromString(process.env.OPERATOR_ID)
const operatorKey = PrivateKey.fromString(process.env.OPERATOR_PVKEY)

const TOPIC_CONTRACT_UPGRADE = "0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b"
const TOPIC_LOG_MSG = "0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2"

async function getEventsFromMirror(contractName, contractAddress, jsonFile) {
  const contractId = "0.0.34400899"
  // const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
	console.log("Getting event(s) from mirror for contract " + contractName + ", address " + contractAddress + ", contract id " + contractId.toString());
  const delay = (ms) => new Promise((res) => setTimeout(res, ms));
	// console.log(`Waiting 10s to allow transaction propagation to mirror`);
	// await delay(10000);

  //https://docs.hedera.com/guides/docs/mirror-node-api/rest-api
  //const url = `https://testnet.mirrornode.hedera.com/api/v1/contracts/${contractId.toString()}/results/logs?order=asc`;
  //const url = `https://testnet.mirrornode.hedera.com/api/v1/contracts/${contractId.toString()}/results/logs?limit=1&order=desc`;

  // It seems like on start up, we could do something like this to get the latest index number (or maybe the last x (limit=x))
  //const startUpQuery = `https://testnet.mirrornode.hedera.com/api/v1/contracts/${contractId.toString()}/results/logs?limit=25&order=desc`;

  // Then we could do this every interval where "index=gt:<lastIndex>".
  // If a query returns 25 (default is limit=25), then we should query again right away.
  const query = `https://testnet.mirrornode.hedera.com/api/v1/contracts/${contractId.toString()}/results/logs?index=gt:0&order=asc`;
  console.log("URL: " + query)

  const json = require(jsonFile);

  const POLL_INTERVAL=2000

  let lastIndex = 0
  while (true) {
    axios
      .get(query)
      .then(function (response) {
        const jsonResponse = response.data;

        jsonResponse.logs.forEach((log) => {
          console.log("BOINK: log: %o", log)
          console.log("BOINK: topic[1]: " + log.topics.slice(1))

          if (log.topics.length >= 1 && log.topics[0] === TOPIC_CONTRACT_UPGRADE) {
            console.log("Contract Upgrade")
          } else if (log.topics.length >= 1 && log.topics[0] === TOPIC_LOG_MSG) {
            console.log("Log Event: %o", event); 
            const event = decodeEvent("LogMessagePublished", log.data, log.topics.slice(1), json.abi);
            console.log("   Decoded log Event: %o", event);        
          } else {
            console.log("Something else")
          }
        });
      })
      .catch(function (err) {
        console.error(err);
      });

      await delay(POLL_INTERVAL);
  }
}

function decodeEvent(eventName, log, topics, abi) {
  const eventAbi = abi.find((event) => event.name === eventName && event.type === "event");
  const decodedLog = web3.eth.abi.decodeLog(eventAbi.inputs, log, topics);
  return decodedLog;
}

async function main() {
  await getEventsFromMirror("Wormhole", "0x00000000000000000000000000000000020cea83", "../build/contracts/Implementation.json")
  console.log("All done.") 
}

main()
/*
briley@gusc1a-ossdev-brl1:~/git/wormhole2/ethereum$ OPERATOR_ID="0.0.34399286" OPERATOR_PVKEY="302e020100300506032b657004220420af71a8a658dbbd297c131dedf7bb24dce87ea527d7e9a862b43a918bd5e337af" node hedera_scripts/events.js 
Getting event(s) from mirror for contract Wormhole, address 0x00000000000000000000000000000000020cea83, contract id 0.0.34400899
URL: https://testnet.mirrornode.hedera.com/api/v1/contracts/0.0.34400899/results/logs?order=asc
BOINK: log: {
  address: '0x00000000000000000000000000000000020cea83',
  bloom: '0x00000020000200000000000000000000420000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000040000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
  contract_id: '0.0.34400899',
  data: '0x',
  index: 0,
  topics: [
    '0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b',
    '0x00000000000000000000000000000000000000000000000000000000020cea7f',
    [length]: 2
  ],
  root_contract_id: '0.0.34400899',
  timestamp: '1651713546.891624424'
}
BOINK: topic[1]: 0x00000000000000000000000000000000000000000000000000000000020cea7f
Contract Upgrade
BOINK: log: {
  address: '0x00000000000000000000000000000000020cea83',
  bloom: '0x00000020000200000000000000000000420000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000020000000000000002000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
  contract_id: '0.0.34400899',
  data: '0x',
  index: 1,
  topics: [
    '0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b',
    '0x00000000000000000000000000000000000000000000000000000000020cea81',
    [length]: 2
  ],
  root_contract_id: '0.0.34400899',
  timestamp: '1651713546.891624424'
}
BOINK: topic[1]: 0x00000000000000000000000000000000000000000000000000000000020cea81
Contract Upgrade

URL: https://testnet.mirrornode.hedera.com/api/v1/contracts/0.0.34400899/results/logs?limit=1&order=desc
BOINK: log: {
  address: '0x00000000000000000000000000000000020cea83',
  bloom: '0x00000020000200000000000000000000420000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000020000000000000002000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
  contract_id: '0.0.34400899',
  data: '0x',
  index: 1,
  topics: [
    '0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b',
    '0x00000000000000000000000000000000000000000000000000000000020cea81',
    [length]: 2
  ],
  root_contract_id: '0.0.34400899',
  timestamp: '1651713546.891624424'
}
BOINK: topic[1]: 0x00000000000000000000000000000000000000000000000000000000020cea81
Contract Upgrade
*/