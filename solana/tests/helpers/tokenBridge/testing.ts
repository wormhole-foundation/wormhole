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
      if (arbiterFee === undefined) {
        arbiterFee = BigInt(0);
      }
      const totalTokenBalanceChange = balancesAfter.token - balancesBefore.token;
      expect(totalTokenBalanceChange % BigInt(2)).to.equal(BigInt(0));
      const balanceChange = totalTokenBalanceChange / BigInt(2);
      expect(balancesBefore.custodyToken - balancesAfter.custodyToken - arbiterFee).to.equal(
        balanceChange
      );
      expect(
        balancesBefore.forkCustodyToken - balancesAfter.forkCustodyToken - arbiterFee
      ).to.equal(balanceChange);
      return;
    }
    default: {
      throw new Error("impossible TransferDirection");
    }
  }
}

export async function expectCorrectWrappedTokenBalanceChanges(
  connection: Connection,
  token: PublicKey,
  forkedToken: PublicKey,
  balancesBefore: TokenBalances,
  direction: TransferDirection,
  expectedChange: bigint
) {
  const program = getAnchorProgram(connection, localnet());
  const forkedProgram = getAnchorProgram(connection, mainnet());
  const balancesAfter = await getTokenBalances(program, forkedProgram, token, forkedToken);

  switch (direction) {
    case TransferDirection.Out: {
      expect(balancesBefore.token - balancesAfter.token).to.equal(expectedChange);
      expect(balancesBefore.forkToken - balancesAfter.forkToken).to.equal(expectedChange);
      return;
    }
    case TransferDirection.In: {
      expect(balancesAfter.token - balancesBefore.token).to.equal(expectedChange);
      expect(balancesAfter.forkToken - balancesBefore.forkToken).to.equal(expectedChange);
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

  const totalTokenBalanceChange = balancesAfter.token - balancesBefore.token;
  expect(totalTokenBalanceChange % BigInt(2)).to.equal(BigInt(0));
  const balanceChange = totalTokenBalanceChange / BigInt(2);
  expect(balanceChange).to.equal(expectedRelayerFee);
}
