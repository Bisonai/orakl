import { api, fetchInternalApi } from "@/utils/api";
import { useQuery, useMutation } from "react-query";
import TwoColumnTable from "../ConfigurationTable/TwoColumnTable";
import { useApi } from "@/lib/useApi";

const Chain = () => {
  const { configQuery, addMutation, deleteMutation } = useApi({
    name: "getChainConfig",
    name2: "modifyChainConfig",
    key: "chainConfig",
  });
  return (
    <>
      <TwoColumnTable
        title="Chain List"
        data={configQuery.data}
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
        deleteConfirmText={"Delete"}
        onDelete={(id) => deleteMutation.mutate(id)}
        onAdd={(newChainName) => addMutation.mutate(newChainName)}
      />
    </>
  );
};

export default Chain;
