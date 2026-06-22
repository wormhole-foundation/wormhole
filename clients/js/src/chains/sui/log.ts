import {
  getCreatedObjects,
  getPublishedPackageId,
  SuiTransactionResult,
} from "./utils";

export const logTransactionDigest = (
  res: SuiTransactionResult,
  ...args: string[]
) => {
  console.log("Transaction digest", res.digest, ...args);
};

export const logTransactionSender = (res: SuiTransactionResult) => {
  console.log("Transaction sender", res.sender);
};

export const logPublishedPackageId = (res: SuiTransactionResult) => {
  console.log("Published to", getPublishedPackageId(res));
};

export const logCreatedObjects = (res: SuiTransactionResult) => {
  console.log(
    "Created objects",
    JSON.stringify(getCreatedObjects(res), null, 2)
  );
};
