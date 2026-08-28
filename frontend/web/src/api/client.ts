import axios, { AxiosError } from "axios";

const TOKEN_KEY = "aether_token";
const REFRESH_KEY = "aether_refresh_token";
const ORG_KEY = "aether_org";

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) || "";
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}

export function setRefreshToken(token: string) { localStorage.setItem(REFRESH_KEY, token); }
export function getRefreshToken(): string { return localStorage.getItem(REFRESH_KEY) || ""; }

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_KEY);
  localStorage.removeItem(ORG_KEY);
}

export function getServer(): string {
  return "";
}

export function setServer(_server: string) {}

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
  const token = getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (res) => res,
  async (err: AxiosError<{ error?: string }>) => {
    const config = err.config as (typeof err.config & { _retry?: boolean }) | undefined;
    if (err.response?.status === 401 && config && !config._retry && getRefreshToken() && !config.url?.endsWith("/auth/refresh")) {
      config._retry = true;
      try {
        const response = await http.post<{ token: string; refresh_token: string }>("/api/v1/auth/refresh", { refresh_token: getRefreshToken() });
        setToken(response.data.token);
        setRefreshToken(response.data.refresh_token);
        config.headers.Authorization = `Bearer ${response.data.token}`;
        return http.request(config);
      } catch { clearToken(); }
    }
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
      axiosErr.response?.data?.error || axiosErr.message || "network error";
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

export async function apiUpload<T>(
  path: string,
  file: File,
  onProgress?: (loaded: number, total: number) => void
): Promise<T> {
  const form = new FormData();
  form.append("file", file);
  try {
    const res = await http.post<T>(`${getServer()}${path}`, form, {
      headers: { "X-File-Size": String(file.size) },
      onUploadProgress: (event) => {
        if (onProgress && event.total) {
          onProgress(event.loaded, event.total);
        }
      },
    });
    return res.data;
  } catch (err) {
    const axiosErr = err as AxiosError<{ error?: string }>;
    const status = axiosErr.response?.status ?? 0;
    const message =
      axiosErr.response?.data?.error || axiosErr.message || "network error";
    throw new ApiError(status, message);
  }
}

export { http };
