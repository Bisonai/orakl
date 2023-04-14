import { boolean } from "mathjs";
import { sendRequestData } from "./request-response";
import { sendRequestDataDirect } from "./request-response-direct";
import { sendRequestRandomWords } from "./vrf-consumer";
import { sendRequestRandomWordsDirect } from "./vrf-consumer-direct";
let vrfRunning = false;
let rrRunning = false;

async function main() {
  setInterval(async () => {
    if (vrfRunning) return;
    vrfRunning = true;
    const d = new Date();
    const m = d.toISOString().split("T")[0];
    await sendRequestRandomWords(m);
    await sendRequestRandomWordsDirect(m);
    vrfRunning = false;
  }, 1000 * 30);

  setInterval(async () => {
    if (rrRunning) return;
    rrRunning = true;
    const d = new Date();
    const m = d.toISOString().split("T")[0];
    await sendRequestData(m);
    await sendRequestDataDirect(m);
    rrRunning = false;
  }, 1000 * 70);
}
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
