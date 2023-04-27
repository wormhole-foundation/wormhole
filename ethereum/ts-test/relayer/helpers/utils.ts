import { BigNumber, ethers } from "ethers";
import { ChainId, tryNativeToHexString } from "@certusone/wormhole-sdk";
import { WORMHOLE_MESSAGE_EVENT_ABI, GUARDIAN_PRIVATE_KEY } from "./consts";

const SAFE_RELAY_DELAY = 8000;

const characters =
  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

export const generateRandomString = (length: number) => {
  let randomString = "";
  for (let i = 0; i < length; i++) {
    randomString += characters.charAt(
      Math.floor(Math.random() * characters.length)
    );
  }
  return randomString;
};

export async function waitForRelay(quantity?: number) {
  await new Promise((resolve) =>
    setTimeout(resolve, SAFE_RELAY_DELAY * (quantity || 1))
  );
}
