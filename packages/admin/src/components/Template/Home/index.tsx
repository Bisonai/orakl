import React, { useEffect, useState } from "react";
import BullMonitorTemplate from "../BullMonitor/main";
import { LoginPage } from "./Login";
import { getCookie } from "@/lib/cookies";
import { IsLoadingBase } from "../BullMonitor/DetailTable/styled";

export default function HomeTemplate() {
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

  return <BullMonitorTemplate />;
}
