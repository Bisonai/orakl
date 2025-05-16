import { check, sleep } from "k6";
import http from "k6/http";

const API_KEY = ""; // Replace with your actual API key

export const options = {
  stages: [
    { duration: "30s", target: 48 },
    { duration: "1m30s", target: 32 },
    { duration: "20s", target: 0 },
  ],
};

export default function () {
  const url =
    "https://dal.cypress.orakl.network/latest-data-feeds/transpose/all";
  const params = {
    headers: {
      "X-API-KEY": API_KEY,
    },
  };

  const res = http.get(url, params);
  check(res, { "status was 200": (r) => r.status == 200 });
  sleep(1);
}
