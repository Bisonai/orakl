import { randomIntBetween } from "https://jslib.k6.io/k6-utils/1.2.0/index.js";
import { check } from "k6";

import ws from "k6/ws";
const sessionDuration = randomIntBetween(120000, 240000);
const msg =
  '{"method":"SUBSCRIBE","params":["submission@ADA-KRW","submission@AKT-KRW","submission@AAVE-KRW","submission@ADA-USDT","submission@ATOM-USDT","submission@APT-KRW","submission@ASTR-KRW","submission@AUCTION-KRW","submission@AVAX-USDT","submission@ARB-KRW","submission@AVAX-KRW","submission@BCH-KRW","submission@BLUR-KRW","submission@AXS-KRW","submission@BLAST-KRW","submission@BNB-USDT","submission@BORA-KRW","submission@BTC-KRW","submission@BTC-USDT","submission@BSV-KRW","submission@CHF-USD","submission@BTG-KRW","submission@BTT-KRW","submission@BONK-KRW","submission@CHZ-KRW","submission@CTC-KRW","submission@DAI-USDT","submission@DOGE-USDT","submission@DOGE-KRW","submission@DOT-KRW","submission@DOT-USDT","submission@ENS-KRW","submission@EOS-KRW","submission@ETH-USDT","submission@ETC-KRW","submission@FET-KRW","submission@FTM-USDT","submission@FLOW-KRW","submission@GAS-KRW","submission@GBP-USD","submission@GLM-KRW","submission@ETH-KRW","submission@GRT-KRW","submission@HPO-KRW","submission@HBAR-KRW","submission@EUR-USD","submission@IQ-KRW","submission@JOY-USDT","submission@IMX-KRW","submission@JPY-USD","submission@KLAY-KRW","submission@KLAY-USDT","submission@KNC-KRW","submission@KRW-USD","submission@LINK-KRW","submission@KSP-KRW","submission@LTC-USDT","submission@MBL-KRW","submission@MATIC-KRW","submission@MBX-KRW","submission@MED-KRW","submission@MATIC-USDT","submission@MLK-KRW","submission@MINA-KRW","submission@MNR-KRW","submission@NEAR-KRW","submission@MTL-KRW","submission@ONDO-KRW","submission@ONG-KRW","submission@PEPE-KRW","submission@SAND-KRW","submission@PYTH-KRW","submission@PAXG-USDT","submission@SEI-KRW","submission@SHIB-KRW","submission@SNT-KRW","submission@PER-KLAY","submission@SHIB-USDT","submission@PEPE-USDT","submission@SOL-USDT","submission@SOL-KRW","submission@STPT-KRW","submission@STG-KRW","submission@STX-KRW","submission@STRK-KRW","submission@TRX-KRW","submission@SUI-KRW","submission@TRX-USDT","submission@USDT-KRW","submission@WEMIX-KRW","submission@USDT-USD","submission@USDC-USDT","submission@WEMIX-USDT","submission@WLD-KRW","submission@WAVES-KRW","submission@XEC-KRW","submission@XLM-KRW","submission@XRP-KRW","submission@XRP-USDT","submission@ZK-KRW","submission@ZETA-KRW","submission@ZRO-KRW","submission@UNI-USDT"]}';
export const options = {
  // A number specifying the number of VUs to run concurrently.
  vus: 10,
  // A string specifying the total duration of the test run.
  iterations: 30,

  // The following section contains configuration options for execution of this
  // test script in Grafana Cloud.
  //
  // See https://grafana.com/docs/grafana-cloud/k6/get-started/run-cloud-tests-from-the-cli/
  // to learn about authoring and running k6 test scripts in Grafana k6 Cloud.
  //
  // cloud: {
  //   // The ID of the project to which the test is assigned in the k6 Cloud UI.
  //   // By default tests are executed in default project.
  //   projectID: "",
  //   // The name of the test in the k6 Cloud UI.
  //   // Test runs with the same name will be grouped.
  //   name: "script.js"
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
export default function () {
  const url = "ws://dal.baobab.orakl.network/ws";
  const params = { headers: { "X-API-Key": "" } };

  const res = ws.connect(url, params, function (socket) {
    socket.on("open", function open() {
      console.log("connected");
      socket.send(msg);
    });

    socket.on("ping", function () {
      console.log("PING!");
    });

    socket.on("message", (data) => console.log("Message received: ", data));
    socket.on("close", () => console.log("disconnected"));

    socket.setTimeout(function () {
      console.log(`Closing the socket forcefully 3s after graceful LEAVE`);
      socket.close();
    }, sessionDuration + 15000);
  });

  check(res, { "status is 101": (r) => r && r.status === 101 });
}
