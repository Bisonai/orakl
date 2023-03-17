import { reportRR } from "./request-response";
import { reportVRF } from "./vrf";
import nodeCron from "node-cron";

async function main() {
  await reportVRF();
  await reportRR();
}

// [minute] [hour] [day of month] [month] [day of week]
// run at 1h
nodeCron.schedule("0 1 * * *", async () => {
  main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
});
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
