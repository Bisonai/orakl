import { fetchInternalApi } from "@/utils/api";
import { useQuery } from "react-query";

import BasicButton from "@/components/Common/BasicButton";
import TwoColumnTable from "../ConfigurationTable/TwoColumnTable";

const Chain = () => {
  const configChainQuery = useQuery({
    queryKey: ["configChain"],
    queryFn: () =>
      fetchInternalApi(
        {
          target: "getConfigChain",
          method: "GET",
        },
        []
      ),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });

  console.log(configChainQuery, "configChainQuery", configChainQuery?.data);
  return (
    <>
      <TwoColumnTable
        title="Chain List"
        data={[{ id: "1", name: "baobab" }]}
        buttonProps={{
          text: "Add Chain",
          width: "150px",
          justifyContent: "center",
          height: "50px",
          margin: "0 30px 0 auto",
          background: "rgb(114, 250, 147)",
          color: "black",
        }}
        addTitle={" Please enter the name of the chain you would like to add."}
        deleteTitle={"Would you like to remove the selected chain?"}
        addConfirmText={"Add"}
        deleteConfrimText={"Delete"}
      />
    </>
  );
};

export default Chain;
