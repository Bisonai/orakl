export enum Route {
  vrf = "vrf",
  "request-response" = "request-response",
  aggregator = "aggregator",
  fetcher = "fetcher",
  settings = "settings",
}

export enum ConfigRoute {
  chain = "chain",
  service = "service",
  listener = "listener",
  vrfKeys = "vrfKeys",
  adapter = "adapter",
  aggregator = "aggregator",
  reporter = "reporter",
  fetcher = "fetcher",
  delegator = "delegator",
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

export const configRoutes: {
  [key in ConfigRoute]: string;
} = {
  [ConfigRoute.chain]: "/configuration/chain",
  [ConfigRoute.service]: "/configuration/service",
  [ConfigRoute.listener]: "/configuration/listener",
  [ConfigRoute.vrfKeys]: "/configuration/vrf-keys",
  [ConfigRoute.adapter]: "/configuration/adapter",
  [ConfigRoute.aggregator]: "/configuration/aggregator",
  [ConfigRoute.reporter]: "/configuration/reporter",
  [ConfigRoute.fetcher]: "/configuration/fetcher",
  [ConfigRoute.delegator]: "/configuration/delegator",
};
