import { gql } from "@apollo/client";

interface Item {
  alias?: string;
  contract: string;
  msg: object;
}

const stringify = (msg: object) => JSON.stringify(msg).replace(/"/g, '\\"');

const aliasItem = ({ alias, contract, msg }: Item) =>
  `
    ${alias ? alias : contract}: WasmContractsContractAddressStore(
      ContractAddress: "${contract}"
      QueryMsg: "${stringify(msg)}"
    ) {
      Height
      Result
    }`;

export const alias = (list: Item[]) => gql`
  query {
    ${list.map(aliasItem)}
  }
`;
