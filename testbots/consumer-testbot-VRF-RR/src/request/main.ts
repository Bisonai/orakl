import { boolean } from "mathjs";
import { sendRequestData } from "./request-response";
import { sendRequestDataDirect } from "./request-response-direct";
import { sendRequestRandomWords } from "./vrf-consumer";
import { sendRequestRandomWordsDirect } from "./vrf-consumer-direct";
let running = false;
async function main() {
  setInterval(async () => {
    if (running) return;
    running = true;
    const d = new Date();
    const m = d.toISOString().split("T")[0];
    await sendRequestRandomWords(m);
    await sendRequestRandomWordsDirect(m);
    // await sendRequestData(m);
    // await sendRequestDataDirect(m);
    running = false;
  }, 1000 * 30);
}
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
