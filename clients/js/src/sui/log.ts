import {
  getTransactionDigest,
  getTransactionSender,
  SuiTransactionBlockResponse,
} from "@mysten/sui.js";
import { getCreatedObjects, getPublishedPackageId } from "./utils";

export const logTransactionDigest = (
  res: SuiTransactionBlockResponse,
  ...args: string[]
) => {
  console.log("Transaction digest", getTransactionDigest(res), ...args);
};

export const logTransactionSender = (res: SuiTransactionBlockResponse) => {
  console.log("Transaction sender", getTransactionSender(res));
};

export const logPublishedPackageId = (res: SuiTransactionBlockResponse) => {
  console.log("Published to", getPublishedPackageId(res));
};

export const logCreatedObjects = (res: SuiTransactionBlockResponse) => {
  console.log(
    "Created objects",
    JSON.stringify(getCreatedObjects(res), null, 2)
  );
};
