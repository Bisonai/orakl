export enum Route {
  vrf = "vrf",
  "request-response" = "request-response",
  aggregator = "aggregator",
  fetcher = "fetcher",
  settings = "settings",
}

export const routes: {
  [key in Route]: string;
} = {
  [Route.vrf]: "/bullmonitor/vrf",
  [Route["request-response"]]: "bullmonitor/request-response",
  [Route.aggregator]: "/bullmonitor/aggregator",
  [Route.fetcher]: "/bullmonitor/fetcher",
  [Route.settings]: "/bullmonitor/settings",
};
