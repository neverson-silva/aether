import axios, { AxiosError } from "axios";

const TOKEN_KEY = "aether_token";
const SERVER_KEY = "aether_server";
const ORG_KEY = "aether_org";

export function getToken(): string {
  return "";
}

export function setToken(_token: string) {
  localStorage.removeItem(TOKEN_KEY);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(ORG_KEY);
}

export function getServer(): string {
  return localStorage.getItem(SERVER_KEY) || "";
}

export function setServer(server: string) {
  localStorage.setItem(SERVER_KEY, server);
}

export function getOrgId(): string {
  return localStorage.getItem(ORG_KEY) || "";
}

export function setOrgId(orgId: string) {
  localStorage.setItem(ORG_KEY, orgId);
  window.dispatchEvent(new CustomEvent("aether:org", { detail: orgId }));
}

export function isPublicRoute(): boolean {
  const p = window.location.pathname.replace(/\/+$/, "");
  return p === "/login" || p === "/onboarding";
}

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

const http = axios.create({ withCredentials: true });

http.interceptors.request.use((config) => {
  const orgId = getOrgId();
  if (orgId) {
    config.headers["X-Aether-Org"] = orgId;
  }
  return config;
});

http.interceptors.response.use(
  (res) => res,
  (err: AxiosError<{ error?: string }>) => {
    if (err.response?.status === 401 && !isPublicRoute()) {
      window.location.href = "/login";
    }
    return Promise.reject(err);
  }
);

export async function api<T>(path: string, options: { method?: string; body?: unknown } = {}): Promise<T> {
  try {
    const res = await http.request<T>({
      baseURL: getServer(),
      url: path,
      method: options.method ?? "GET",
      data: options.body !== undefined ? JSON.stringify(options.body) : undefined,
      headers: {
        "Content-Type": "application/json",
      },
    });
    return res.data;
  } catch (err) {
    const axiosErr = err as AxiosError<{ error?: string }>;
    const status = axiosErr.response?.status ?? 0;
    const message =
      axiosErr.response?.data?.error || axiosErr.message || "erro de rede";
    throw new ApiError(status, message);
  }
}

export const apiGet = <T>(path: string) => api<T>(path);
export const apiPost = <T>(path: string, body?: unknown) =>
  api<T>(path, { method: "POST", body });
export const apiPut = <T>(path: string, body?: unknown) =>
  api<T>(path, { method: "PUT", body });
export const apiDelete = <T>(path: string) => api<T>(path, { method: "DELETE" });
export const apiPatch = <T>(path: string, body?: unknown) =>
  api<T>(path, { method: "PATCH", body });

export { http };
