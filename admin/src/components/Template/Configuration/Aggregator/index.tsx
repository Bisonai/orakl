import DataTable from "../ConfigurationTable/DataTable";

const Aggregator = () => {
  return (
    <DataTable
      fetchEndpoint="getAggregatorConfig"
      deleteEndpoint="modifyAggregatorConfig"
      apiKey="aggregatorConfig"
      title="Aggregators"
      dataLabels={[
        "aggregatorHash",
        "name",
        "address",
        "active",
        "heartbeat",
        "threshold",
        "absoluteThreshold",
        "adapterId",
        "chainId",
      ]}
      jsonData={{
        aggregatorHash: "string",
        active: true,
        name: "string",
        address: "string",
        heartbeat: 0,
        threshold: 0,
        absoluteThreshold: 0,
        adapterHash: "string",
        chain: "string",
      }}
    />
  );
};

export default Aggregator;
