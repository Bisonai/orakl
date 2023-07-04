import React, { useEffect, useState } from "react";
import BullMonitorTemplate from "../BullMonitor/main";
import { LoginPage } from "./Login";
import { getCookie } from "@/lib/cookies";

export default function HomeTemplate() {
  const [isLoggedIn, setIsLoggedIn] = useState(false);

  useEffect(() => {
    const token = getCookie("token");
    setIsLoggedIn(!!token);
  }, []);

  if (isLoggedIn === false) {
    return <LoginPage onLogin={() => setIsLoggedIn(true)} />;
  }

  return <BullMonitorTemplate />;
}
