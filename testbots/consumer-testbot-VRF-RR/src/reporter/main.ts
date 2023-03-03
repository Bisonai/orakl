import { reportRR } from "./request-response";
import { reportVRF } from "./vrf";

async function main() {
  await reportVRF();
  await reportRR();
}
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
