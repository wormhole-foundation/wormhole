import { TransactionResponse } from "@solana/web3.js";
import { TxInfo } from "@terra-money/terra.js";
import { BigNumber, ContractReceipt } from "ethers";
import { Implementation__factory } from "../ethers-contracts";

export function parseSequenceFromLogEth(
  receipt: ContractReceipt,
  bridgeAddress: string
): string {
  // TODO: dangerous!(?)
  const bridgeLog = receipt.logs.filter((l) => {
    return l.address === bridgeAddress;
  })[0];
  const {
    args: { sequence },
  } = Implementation__factory.createInterface().parseLog(bridgeLog);
  return sequence.toString();
}

export function parseSequencesFromLogEth(
  receipt: ContractReceipt,
  bridgeAddress: string
): string[] {
  // TODO: dangerous!(?)
  const bridgeLogs = receipt.logs.filter((l) => {
    return l.address === bridgeAddress;
  });
  return bridgeLogs.map((bridgeLog) => {
    const {
      args: { sequence },
    } = Implementation__factory.createInterface().parseLog(bridgeLog);
    return sequence.toString();
  });
}

export function parseSequenceFromLogTerra(info: TxInfo): string {
  // Scan for the Sequence attribute in all the outputs of the transaction.
  // TODO: Make this not horrible.
  let sequence = "";
  const jsonLog = JSON.parse(info.raw_log);
  jsonLog.map((row: any) => {
    row.events.map((event: any) => {
      event.attributes.map((attribute: any) => {
        if (attribute.key === "message.sequence") {
          sequence = attribute.value;
        }
      });
    });
  });
  return sequence.toString();
}

export function parseSequencesFromLogTerra(info: TxInfo): string[] {
  // Scan for the Sequence attribute in all the outputs of the transaction.
  // TODO: Make this not horrible.
  const sequences: string[] = [];
  const jsonLog = JSON.parse(info.raw_log);
  jsonLog.map((row: any) => {
    row.events.map((event: any) => {
      event.attributes.map((attribute: any) => {
        if (attribute.key === "message.sequence") {
          sequences.push(attribute.value.toString());
        }
      });
    });
  });
  return sequences;
}

export function parseSequenceFromLogInjective(info: any): string {
  // Scan for the Sequence attribute in all the outputs of the transaction.
  // TODO: Make this not horrible.
  let sequence = "";
  const jsonLog = JSON.parse(info.rawLog);
  jsonLog.map((row: any) => {
    row.events.map((event: any) => {
      event.attributes.map((attribute: any) => {
        if (attribute.key === "message.sequence") {
          sequence = attribute.value;
        }
      });
    });
  });
  return sequence.toString();
}

const SOLANA_SEQ_LOG = "Program log: Sequence: ";
export function parseSequenceFromLogSolana(info: TransactionResponse) {
  // TODO: better parsing, safer
  const sequence = info.meta?.logMessages
    ?.filter((msg) => msg.startsWith(SOLANA_SEQ_LOG))?.[0]
    ?.replace(SOLANA_SEQ_LOG, "");
  if (!sequence) {
    throw new Error("sequence not found");
  }
  return sequence.toString();
}

export function parseSequencesFromLogSolana(info: TransactionResponse) {
  // TODO: better parsing, safer
  return info.meta?.logMessages
    ?.filter((msg) => msg.startsWith(SOLANA_SEQ_LOG))
    .map((msg) => msg.replace(SOLANA_SEQ_LOG, ""));
}

export function parseSequenceFromLogAlgorand(
  result: Record<string, any>
): string {
  let sequence = "";
  if (result["inner-txns"]) {
    const innerTxns: [] = result["inner-txns"];
    class iTxn {
      "local-state-delta": [[Object]];
      logs: Buffer[] | undefined;
      "pool-eror": string;
      txn: { txn: [Object] } | undefined;
    }
    innerTxns.forEach((txn: iTxn) => {
      if (txn.logs) {
        sequence = BigNumber.from(txn.logs[0].slice(0, 8)).toString();
      }
    });
  }
  return sequence;
}
