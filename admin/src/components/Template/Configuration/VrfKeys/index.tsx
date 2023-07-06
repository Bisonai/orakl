import DataTable from "../ConfigurationTable/DataTable";

const VrfKeys = () => {
  return (
    <DataTable
      fetchEndpoint="getVrfKeysConfig"
      deleteEndpoint="modifyVrfKeysConfig"
      apiKey="vrfKeysConfig"
      title="VRF Keys"
      dataLabels={["id", "sk", "pk", "pkX", "pkY", "keyHash", "chain"]}
      jsonData={{
        sk: "string",
        pk: "string",
        pkX: "string",
        pkY: "string",
        keyHash: "string",
        chain: "string",
      }}
    />
  );
};

export default VrfKeys;
