import { useState, useEffect } from "react";
import Adapter from "./Adapter";
import Aggregator from "./Aggregator";
import Chain from "./Chain";
import Delegator from "./Delegator";
import Listener from "./Listener";
import Reporter from "./Reporter";
import Service from "./Service";
import VrfKeys from "./VrfKeys";

import { getCookie } from "@/lib/cookies";
import { LoginPage } from "../Home/Login";
import { IsLoadingBase } from "../BullMonitor/DetailTable/styled";

export default function ConfigurationDetailTemplate({
  configType,
}: {
  configType: string;
}) {
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [tokenChecked, setTokenChecked] = useState(false);

  useEffect(() => {
    const token = getCookie("token");
    setIsLoggedIn(!!token);
    setTokenChecked(true);
  }, []);

  if (!tokenChecked) {
    return <IsLoadingBase>Loading...</IsLoadingBase>;
  }
  if (isLoggedIn === false) {
    return <LoginPage onLogin={() => setIsLoggedIn(true)} />;
  }

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
