import { sendRequestData } from "./request-response";
import { sendRequestDataDirect } from "./request-response-direct";
import { sendRequestRandomWords } from "./vrf-consumer";
import { sendRequestRandomWordsDirect } from "./vrf-consumer-direct";

async function main() {
  setInterval(async () => {
    await sendRequestRandomWords();
    await sendRequestRandomWordsDirect();
    await sendRequestData();
    await sendRequestDataDirect();
  }, 1000 * 30);
}
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
