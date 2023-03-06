import { reportRR } from "./request-response";
import { reportVRF } from "./vrf";
import nodeCron from "node-cron";

async function main() {
  await reportVRF();
  await reportRR();
}

// run at 0h
nodeCron.schedule("0 0 * * *", async () => {
  main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
});
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
