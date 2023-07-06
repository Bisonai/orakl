import DataTable from "../ConfigurationTable/DataTable";

const Adapter = () => {
  return (
    <DataTable
      fetchEndpoint="getAdapterConfig"
      deleteEndpoint="modifyAdapterConfig"
      apiKey="adapterConfig"
      title="Adapters"
      dataLabels={["id", "adapterHash", "name", "decimals"]}
      jsonData={{
        adapterHash: "string",
        name: "string",
        decimals: 0,
        feeds: [
          {
            name: "string",
            definition: {},
          },
        ],
      }}
    />
  );
};

export default Adapter;
