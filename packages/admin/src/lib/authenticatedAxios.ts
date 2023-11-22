import axios from "axios";
import { getCookie } from "@/lib/cookies";

const authenticatedAxios = axios.create();

authenticatedAxios.interceptors.request.use((config) => {
  const token = getCookie("token");

  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }

  return config;
});

export default authenticatedAxios;
