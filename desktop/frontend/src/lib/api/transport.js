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
    // Preserve HTTP status on the Error so callers (e.g. plugins API) can
    // distinguish 404 / 401 / 502 without parsing message text alone.
    const msg = err.response?.data?.error?.message || err.message;
    const wrapped = new Error(msg);
    wrapped.response = err.response;
    wrapped.status = err.response?.status;
    throw wrapped;
  },
);
