import TwoColumnTable from "../ConfigurationTable/TwoColumnTable";

const Service = () => {
  return (
    <>
      <TwoColumnTable
        title="Service List"
        data={[
          {
            id: "1",
            name: "VRF",
          },
          {
            id: "2",
            name: "Aggregator",
          },
          {
            id: "3",
            name: "RequestResponse",
          },
        ]}
        buttonProps={{
          text: "Add Chain",
          width: "150px",
          justifyContent: "center",
          height: "50px",
          margin: "0 30px 0 auto",
          background: "rgb(114, 250, 147)",
          color: "black",
        }}
        addTitle={""}
        deleteTitle={""}
        addConfirmText={""}
        deleteConfrimText={""}
      />
    </>
  );
};
export default Service;
