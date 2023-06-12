import { fetchInternalApi } from "@/utils/api";
import { useQuery } from "react-query";
import TwoColumnTable from "../ConfigurationTable/TwoColumnTable";
import { useMutation } from "react-query";
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

    select: (data) => {
      console.log("Received data from query:", data);
      return data.data;
    },
  });
  console.log(configChainQuery, "configChainQuery", configChainQuery?.data);
  const addChain = async (newChainName: string) => {
    console.log("Adding chain with data:", newChainName); // 데이터 출력
    const response = await fetch(`http://localhost:3030/api/v1/chain`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ name: newChainName }), // Here, we send an object with the property name
    });

    if (!response.ok) {
      throw new Error("Network response was not ok");
    }
    return response.json();
  };
  // Delete request
  const deleteChain = async (id: any) => {
    const response = await fetch(`http://localhost:3030/api/v1/chain/${id}`, {
      method: "DELETE",
    });

    if (!response.ok) {
      throw new Error("Network response was not ok");
    }

    return response.json();
  };

  // Using the defined functions with useMutation
  const addChainMutation = useMutation(addChain, {
    onSuccess: (data) => {
      console.log("Data after addChainMutation:", data);
    },
    onError: (error) => {
      console.log("Error with addChainMutation:", error);
    },
  });
  const deleteChainMutation = useMutation(deleteChain);

  // Call the mutations

  console.log(configChainQuery, "configChainQuery", configChainQuery?.data);
  return (
    <>
      <TwoColumnTable
        title="Chain List"
        data={configChainQuery.data}
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
        onDelete={(id) => deleteChainMutation.mutate(id)}
        onAdd={(newChainName) => addChainMutation.mutate(newChainName)}
      />
    </>
  );
};

export default Chain;
