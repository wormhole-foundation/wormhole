import { CosmWasmClient } from "@cosmjs/cosmwasm-stargate";

const WORMCHAIN_URL = "https://wormchain.jumpisolated.com";
const ACCOUNTANT_CONTRACT_ADDRESS =
  "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465";
const PAGE_LIMIT = 2000; // throws a gas limit error over this

export type Account = {
  key: {
    chain_id: number;
    token_chain: number;
    token_address: string;
  };
  balance: string;
};

const getAccountantAccounts = async (): Promise<Account[]> => {
  const cosmWasmClient = await CosmWasmClient.connect(WORMCHAIN_URL);
  let accounts: Account[] = [];
  let response;
  let start_after = undefined;
  do {
    response = await cosmWasmClient.queryContractSmart(
      ACCOUNTANT_CONTRACT_ADDRESS,
      {
        all_accounts: {
          limit: PAGE_LIMIT,
          start_after,
        },
      }
    );
    accounts = [...accounts, ...response.accounts];
    start_after =
      response.accounts.length &&
      response.accounts[response.accounts.length - 1].key;
  } while (response.accounts.length === PAGE_LIMIT);
  return accounts;
};

(async () => {
  const accounts = await getAccountantAccounts();
  console.log(JSON.stringify(accounts));
})();
