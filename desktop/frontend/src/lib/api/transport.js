import axios from "axios";

export const API_PREFIX = "/api/v1";

export const client = axios.create({
  timeout: 15000,
  headers: { "Content-Type": "application/json" },
});

client.interceptors.request.use((config) => {
  config.baseURL =
    (typeof window !== "undefined" && window.__ICOOSERVER_URL) ||
    "http://127.0.0.1:18181";
  const key =
    (typeof window !== "undefined" && window.__ICOOSERVER_API_KEY) || "";
  if (key) config.headers.Authorization = `Bearer ${key}`;
  return config;
});

client.interceptors.response.use(
  (res) => res.data?.data,
  (err) => {
    const msg = err.response?.data?.error?.message || err.message;
    throw new Error(msg);
  },
);
