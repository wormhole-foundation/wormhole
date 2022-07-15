import nearAPI from "near-api-js";

export function logNearGas(result: any, comment: string) {
  const { totalGasBurned, totalTokensBurned } = result.receipts_outcome.reduce(
    (acc: any, receipt: any) => {
      acc.totalGasBurned += receipt.outcome.gas_burnt;
      acc.totalTokensBurned += nearAPI.utils.format.formatNearAmount(
        receipt.outcome.tokens_burnt
      );
      return acc;
    },
    {
      totalGasBurned: result.transaction_outcome.outcome.gas_burnt,
      totalTokensBurned: nearAPI.utils.format.formatNearAmount(
        result.transaction_outcome.outcome.tokens_burnt
      ),
    }
  );
  console.log(
    comment,
    "totalGasBurned",
    totalGasBurned,
    "totalTokensBurned",
    totalTokensBurned
  );
}
