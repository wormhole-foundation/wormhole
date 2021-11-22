import { exit } from "process";
import * as examples from "../lib/examples";

function logWrapper(promise) {
  return promise.catch((e) => {
    console.log(e);
    return Promise.resolve();
  });
}

export async function runAll() {
  console.log("Attestation example");
  await logWrapper(examples.attestationExample());
  console.log("Attestation complete.");
  console.log();

  console.log("Transfer example");
  await logWrapper(examples.transferWithRelayHandoff());
  console.log("Transfer example complete.");
  console.log();

  console.log("Complete");

  return Promise.resolve();
}

let done = false;
runAll().then(
  () => (done = true),
  () => (done = true)
);
function wait() {
  if (!done) {
    setTimeout(wait, 1000);
  } else {
    exit(0);
  }
}
wait();
