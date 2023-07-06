import { useQuery, useMutation } from "react-query";
import { fetchInternalApi } from "@/utils/api";

export const useApi = (config: any) => {
  const { fetchEndpoint, deleteEndpoint, key } = config;

  const configQuery = useQuery({
    queryKey: [key],
    queryFn: () =>
      fetchInternalApi(
        {
          target: fetchEndpoint,
          method: "GET",
        },
        []
      ),
    refetchOnWindowFocus: false,
    select: (data) => {
      return data.data;
    },
  });
  const addEntity = async (newData: { [key: string]: any }): Promise<any> => {
    const response = await fetchInternalApi(
      {
        target: fetchEndpoint,
        method: "POST",
        data: newData,
      },
      []
    );

    if (response.status !== 200) {
      throw new Error("Network response was not ok");
    }

    return response.data;
  };

  const deleteEntity = async (id: any) => {
    const response = await fetchInternalApi(
      {
        target: deleteEndpoint,
        method: "DELETE",
      },
      [id]
    );

    if (response.status !== 200) {
      throw new Error("Network response was not ok");
    }

    return response.data;
  };

  const addMutation = useMutation(addEntity);
  const deleteMutation = useMutation(deleteEntity);

  return { configQuery, addMutation, deleteMutation };
};
