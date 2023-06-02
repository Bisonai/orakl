"use client";

import Chain from "./Chain";
import Service from "./Service";

export default function ConfigurationDetailTemplate({
  configType,
}: {
  configType: string;
}) {
  return (
    <>
      {configType === "chain" && <Chain />}
      {configType === "service" && <Service />}
    </>
  );
}
