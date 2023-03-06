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
    await sendRequestRandomWords();
    await sendRequestRandomWordsDirect();
    await sendRequestData();
    await sendRequestDataDirect();
    running = false;
  }, 1000 * 30);
}
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
