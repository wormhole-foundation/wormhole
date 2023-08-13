import { Connection, PublicKey } from "@solana/web3.js";
import { TokenBalances, getTokenBalances } from "../utils";
import { expect } from "chai";
import { getAnchorProgram, getProgramId, localnet, mainnet } from ".";

export enum TransferDirection {
  Out,
  In,
}

export async function expectCorrectTokenBalanceChanges(
  connection: Connection,
  token: PublicKey,
  balancesBefore: TokenBalances,
  direction: TransferDirection,
  arbiterFee?: bigint
) {
  const program = getAnchorProgram(connection, localnet());
  const forkedProgram = getAnchorProgram(connection, mainnet());
  const balancesAfter = await getTokenBalances(program, forkedProgram, token);

  switch (direction) {
    case TransferDirection.Out: {
      const totalTokenBalanceChange = balancesBefore.token - balancesAfter.token;
      expect(totalTokenBalanceChange % BigInt(2)).to.equal(BigInt(0));
      const balanceChange = totalTokenBalanceChange / BigInt(2);
      expect(balancesAfter.custodyToken - balancesBefore.custodyToken).to.equal(balanceChange);
      expect(balancesAfter.forkCustodyToken - balancesBefore.forkCustodyToken).to.equal(
        balanceChange
      );
      return;
    }
    case TransferDirection.In: {
      const totalTokenBalanceChange = balancesAfter.token - balancesBefore.token;
      expect(totalTokenBalanceChange % BigInt(2)).to.equal(BigInt(0));
      const balanceChange = totalTokenBalanceChange / BigInt(2);
      expect(balancesBefore.custodyToken - balancesAfter.custodyToken).to.equal(balanceChange);
      expect(balancesBefore.forkCustodyToken - balancesAfter.forkCustodyToken).to.equal(
        balanceChange
      );
      return;
    }
    default: {
      throw new Error("impossible TransferDirection");
    }
  }
}

export async function expectCorrectRelayerBalanceChanges(
  connection: Connection,
  token: PublicKey,
  balancesBefore: TokenBalances,
  expectedRelayerFee: bigint
) {
  const program = getAnchorProgram(connection, localnet());
  const forkedProgram = getAnchorProgram(connection, mainnet());
  const balancesAfter = await getTokenBalances(program, forkedProgram, token);

  const totalTokenBalanceChange = balancesBefore.token - balancesAfter.token;
  expect(totalTokenBalanceChange % BigInt(2)).to.equal(BigInt(0));
  const balanceChange = totalTokenBalanceChange / BigInt(2);
  expect(balanceChange).to.equal(expectedRelayerFee);
}
