import { fetchInternalApi } from "@/utils/api";
import { useQuery } from "react-query";
import TwoColumnTable from "../ConfigurationTable/TwoColumnTable";
import { useMutation } from "react-query";
const Service = () => {
  const configServiceQuery = useQuery({
    queryKey: ["configService"],
    queryFn: () =>
      fetchInternalApi(
        {
          target: "getConfigService",
          method: "GET",
        },
        []
      ),
    refetchOnWindowFocus: false,

    select: (data) => {
      return data.data;
    },
  });

  const addService = async (newServiceName: string) => {
    const response = await fetch(`http://localhost:3030/api/v1/service`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ name: newServiceName }),
    });

    if (!response.ok) {
      throw new Error("Network response was not ok");
    }
    return response.json();
  };

  const deleteService = async (id: any) => {
    const response = await fetch(`http://localhost:3030/api/v1/service/${id}`, {
      method: "DELETE",
    });

    if (!response.ok) {
      throw new Error("Network response was not ok");
    }

    return response.json();
  };

  const addServiceMutation = useMutation(addService, {
    onSuccess: (data) => {
      console.log("Data after addServiceMutation:", data);
    },
    onError: (error) => {
      console.log("Error with addServiceMutation:", error);
    },
  });
  const deleteServiceMutation = useMutation(deleteService);

  return (
    <>
      <TwoColumnTable
        title="Service List"
        data={configServiceQuery.data}
        buttonProps={{
          text: "Add Service",
          width: "150px",
          justifyContent: "center",
          height: "50px",
          margin: "0 30px 0 auto",
          background: "rgb(114, 250, 147)",
          color: "black",
        }}
        addTitle={
          " Please enter the name of the Service you would like to add."
        }
        deleteTitle={"Would you like to remove the selected Service?"}
        addConfirmText={"Add"}
        deleteConfrimText={"Delete"}
        onDelete={(id) => deleteServiceMutation.mutate(id)}
        onAdd={(newServiceName) => addServiceMutation.mutate(newServiceName)}
      />
    </>
  );
};

export default Service;
