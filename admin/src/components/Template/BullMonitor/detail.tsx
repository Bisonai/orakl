"use client";

import { fetchInternalApi } from "@/utils/api";
import { useQuery } from "react-query";
import DetailHeader from "./DetailHeader";
import DetailTab from "./DetailTab";
import { getCookie } from "@/lib/cookies";
import { useState, useEffect } from "react";
import { LoginPage } from "../Home/Login";
import { IsLoadingBase } from "./DetailTable/styled";

export default function BullMonitorDetailTemplate({
  serviceId,
}: {
  serviceId: string;
}): JSX.Element {
  const serviceQuery = useQuery({
    queryKey: ["service", serviceId],
    queryFn: () =>
      fetchInternalApi(
        {
          target: "service",
          method: "GET",
        },
        [serviceId]
      ),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });
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
      <div
        style={{
          background: "#222831",
          padding: "40px 0px",
          margin: "0px 40px",
          maxWidth: "1400px",
        }}
      >
        <DetailHeader serviceId={serviceId} data={serviceQuery?.data} />
        <DetailTab serviceId={serviceId} data={serviceQuery?.data} />
      </div>
    </>
  );
}
