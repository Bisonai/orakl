"use client";

import Adapter from "./Adapter";
import Aggregator from "./Aggregator";
import Chain from "./Chain";
import Delegator from "./Delegator";
import Listener from "./Listener";
import Reporter from "./Reporter";
import Service from "./Service";
import VrfKeys from "./VrfKeys";

export default function ConfigurationDetailTemplate({
  configType,
}: {
  configType: string;
}) {
  return (
    <>
      {configType === "chain" && <Chain />}
      {configType === "service" && <Service />}
      {configType === "listener" && <Listener />}
      {configType === "vrf-keys" && <VrfKeys />}
      {configType === "adapter" && <Adapter />}
      {configType === "aggregator" && <Aggregator />}
      {configType === "reporter" && <Reporter />}
      {configType === "delegator" && <Delegator />}
    </>
  );
}
