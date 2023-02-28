import { reportVRF } from "./vrf-reporter";

async function main() {
  reportVRF();
}
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
