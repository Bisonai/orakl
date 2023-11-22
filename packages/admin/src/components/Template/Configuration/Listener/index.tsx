import DataTable from "../ConfigurationTable/DataTable";

const Listener = () => {
  return (
    <DataTable
      fetchEndpoint="getListenerConfig"
      deleteEndpoint="modifyListenerConfig"
      apiKey="listenerConfig"
      title="Listener"
      dataLabels={["id", "address", "eventName", "service", "chain"]}
      jsonData={{
        address: "string",
        eventName: "string",
        chain: "string",
        service: "string",
      }}
    />
  );
};

export default Listener;
