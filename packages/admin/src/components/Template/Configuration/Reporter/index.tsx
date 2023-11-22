import DataTable from "../ConfigurationTable/DataTable";

const Reporter = () => {
  return (
    <DataTable
      fetchEndpoint="getReporterConfig"
      deleteEndpoint="modifyReporterConfig"
      apiKey="reporterConfig"
      title="Reporter"
      dataLabels={[
        "id",
        "address",
        "privateKey",
        "oracleAddress",
        "service",
        "chain",
      ]}
      jsonData={{
        address: "string",
        privateKey: "string",
        oracleAddress: "string",
        chain: "string",
        service: "string",
      }}
    />
  );
};

export default Reporter;
