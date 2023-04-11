import {
  getPublishedObjectChanges,
  getTransactionDigest,
  getTransactionSender,
  SuiTransactionBlockResponse,
} from "@mysten/sui.js";
import { getCreatedObjects } from "./utils";

export const logTransactionDigest = (res: SuiTransactionBlockResponse) => {
  console.log("Transaction digest", getTransactionDigest(res));
};

export const logTransactionSender = (res: SuiTransactionBlockResponse) => {
  console.log("Transaction sender", getTransactionSender(res));
};

export const logPublishedPackageId = (res: SuiTransactionBlockResponse) => {
  const publishEvents = getPublishedObjectChanges(res);
  if (publishEvents.length !== 1) {
    throw new Error(
      "Unexpected number of publish events found:" +
        JSON.stringify(publishEvents, null, 2)
    );
  }

  console.log("Published to", publishEvents[0].packageId);
};

export const logCreatedObjects = (res: SuiTransactionBlockResponse) => {
  console.log(
    "Created objects",
    JSON.stringify(getCreatedObjects(res), null, 2)
  );
};
