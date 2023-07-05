import { useQuery, useMutation } from "react-query";
import { fetchInternalApi } from "@/utils/api";

export const useApi = (config: any) => {
  const { name, name2, key } = config;

  const configQuery = useQuery({
    queryKey: [key],
    queryFn: () =>
      fetchInternalApi(
        {
          target: name,
          method: "GET",
        },
        []
      ),
    refetchOnWindowFocus: false,
    select: (data) => {
      return data.data;
    },
  });

  const addEntity = async (newEntityName: string) => {
    const response = await fetchInternalApi(
      {
        target: name,
        method: "POST",
        data: { name: newEntityName },
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
        target: name2,
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
