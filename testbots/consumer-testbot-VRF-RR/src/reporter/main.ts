import { reportRR } from "./request-response";
import { reportVRF } from "./vrf";
import nodeCron from "node-cron";
import { time } from "console";

async function main(date: number) {
  await reportVRF(date);
  await reportRR(date);
}

// [minute] [hour] [day of month] [month] [day of week]
// run at 1h
nodeCron.schedule(
  "0 1 * * *",
  async () => {
    main(1).catch((error) => {
      console.error(error);
      process.exitCode = 1;
    });
  },
  {
    timezone: "UTC",
  }
);
main(1).catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
