import { sleep } from "k6";
import http from "k6/http";

export const options = {
  // A number specifying the number of VUs to run concurrently.
  vus: 10,
  // A string specifying the total duration of the test run.
  duration: "30s",

  // The following section contains configuration options for execution of this
  // test script in Grafana Cloud.
  //
  // See https://grafana.com/docs/grafana-cloud/k6/get-started/run-cloud-tests-from-the-cli/
  // to learn about authoring and running k6 test scripts in Grafana k6 Cloud.
  //
  // ext: {
  //   loadimpact: {
  //     // The ID of the project to which the test is assigned in the k6 Cloud UI.
  //     // By default tests are executed in default project.
  //     projectID: "",
  //     // The name of the test in the k6 Cloud UI.
  //     // Test runs with the same name will be grouped.
  //     name: "script.js"
  //   }
  // },

  // Uncomment this section to enable the use of Browser API in your tests.
  //
  // See https://grafana.com/docs/k6/latest/using-k6-browser/running-browser-tests/ to learn more
  // about using Browser API in your test scripts.
  //
  // scenarios: {
  //   // The scenario name appears in the result summary, tags, and so on.
  //   // You can give the scenario any name, as long as each name in the script is unique.
  //   ui: {
  //     // Executor is a mandatory parameter for browser-based tests.
  //     // Shared iterations in this case tells k6 to reuse VUs to execute iterations.
  //     //
  //     // See https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/ for other executor types.
  //     executor: 'shared-iterations',
  //     options: {
  //       browser: {
  //         // This is a mandatory parameter that instructs k6 to launch and
  //         // connect to a chromium-based browser, and use it to run UI-based
  //         // tests.
  //         type: 'chromium',
  //       },
  //     },
  //   },
  // }
};

// The function that defines VU logic.
//
// See https://grafana.com/docs/k6/latest/examples/get-started-with-k6/ to learn more
// about authoring k6 scripts.
//

const body = {
  from: "0x60d690e4d5db4025f4781c6cf3bff8669500823c",
  to: "0x27e1255f2a0ea596992158a0bc838f43be34b99d",
  input:
    "0x202ee0ed000000000000000000000000000000000000000000000000000000000008470b0000000000000000000000000000000000000000000000000000000c1e6bf880",
  gas: "0x61a80",
  value: "0x0",
  chainId: "0x3e9",
  gasPrice: "0xba43b7400",
  nonce: "0x8470a",
  v: "0x07f5",
  r: "0x340a4255cb8eab4d5fc8aac664bbe6b64034dbadc87369ad3b2dd7a2c0a03361",
  s: "0x64593ba4239c27fad361229013510527944f96b825c10b4136b6677e422dbd85",
  rawTx:
    "0x31f8e28308470a850ba43b740083061a809427e1255f2a0ea596992158a0bc838f43be34b99d809460d690e4d5db4025f4781c6cf3bff8669500823cb844202ee0ed000000000000000000000000000000000000000000000000000000000008470b0000000000000000000000000000000000000000000000000000000c1e6bf880f847f8458207f5a0340a4255cb8eab4d5fc8aac664bbe6b64034dbadc87369ad3b2dd7a2c0a03361a064593ba4239c27fad361229013510527944f96b825c10b4136b6677e422dbd85940000000000000000000000000000000000000000c4c3018080",
};

export default function () {
  http.post("http://localhost:3002/api/v1/sign", body);
  sleep(1);
}
