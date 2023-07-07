import { fetchInternalApi } from "@/utils/api";
import React from "react";
import { useQuery } from "react-query";

const Delegator = () => {
  const organizationQuery = useQuery({
    queryKey: ["organization"],
    queryFn: () =>
      fetchInternalApi({
        target: "getOrganization",
        method: "GET",
      }),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });

  const contractQuery = useQuery({
    queryKey: ["contract"],
    queryFn: () =>
      fetchInternalApi({
        target: "getContract",
        method: "GET",
      }),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });

  const functionQuery = useQuery({
    queryKey: ["function"],
    queryFn: () =>
      fetchInternalApi({
        target: "getFunction",
        method: "GET",
      }),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });

  const reporterQuery = useQuery({
    queryKey: ["reporter"],
    queryFn: () =>
      fetchInternalApi({
        target: "getReporter",
        method: "GET",
      }),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });
  console.log(
    organizationQuery.data,
    contractQuery.data,
    functionQuery.data,
    reporterQuery.data,
    "data"
  );

  return (
    <div>
      <h2 style={{ color: "white" }}>Delegator</h2>
    </div>
  );
};

export default Delegator;
