import TwoColumnTable from "../ConfigurationTable/TwoColumnTable";
import { useApi } from "@/lib/useApi";

const Service = () => {
  const { configQuery, addMutation, deleteMutation } = useApi({
    fetchEndpoint: "getServiceConfig",
    deleteEndpoint: "modifyServiceConfig",
    key: "serviceConfig",
  });
  return (
    <>
      <TwoColumnTable
        title="Service List"
        data={configQuery.data}
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
        deleteConfirmText={"Delete"}
        onDelete={(id) => deleteMutation.mutate(id)}
        onAdd={(newServiceName) => addMutation.mutate(newServiceName)}
      />
    </>
  );
};

export default Service;
