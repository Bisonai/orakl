import { cache } from "react";
import { QueryClient } from "react-query";

const getQueryClient = cache(() => new QueryClient());
export default getQueryClient;
