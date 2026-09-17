import axios, { AxiosError } from "axios";

const ORG_KEY = "aether_org";
let refreshPromise: Promise<void> | null = null;

function authLog(message: string, details?: unknown) {
  if (details === undefined) {
    console.info(`[aether:auth] ${message}`);
    return;
  }
  console.info(`[aether:auth] ${message} ${JSON.stringify(details)}`);
}

export function getToken(): string {
	return "";
}

export function setToken(_token: string) {}

export function setRefreshToken(_token: string) {}
export function getRefreshToken(): string { return ""; }

export function clearToken() {
	localStorage.removeItem(ORG_KEY);
}

export async function logout() {
  try {
    await http.post("/api/v1/auth/logout");
  } finally {
    clearToken();
  }
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

async function refreshAccessToken(): Promise<void> {
  if (!refreshPromise) {
    refreshPromise = http.post("/api/v1/auth/refresh")
      .then(() => undefined)
      .finally(() => { refreshPromise = null; });
  }
  return refreshPromise as Promise<void>;
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
  async (err: AxiosError<{ error?: string }>) => {
    const config = err.config as (typeof err.config & { _retry?: boolean }) | undefined;
    if (err.response?.status === 401 && config && !config._retry && !config.url?.endsWith("/auth/refresh")) {
      config._retry = true;
      try {
        await refreshAccessToken();
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
    if (path.includes("/auth/") || path === "/api/v1/me") {
      authLog("request succeeded", { method: options.method ?? "GET", path, status: res.status, visibleCookieNames: document.cookie.split(";").map((entry) => entry.trim().split("=")[0]).filter(Boolean) });
    }
    return res.data;
  } catch (err) {
    const axiosErr = err as AxiosError<{ error?: string }>;
    const status = axiosErr.response?.status ?? 0;
    const message =
      axiosErr.response?.data?.error || axiosErr.message || "network error";
    if (path.includes("/auth/") || path === "/api/v1/me") {
      authLog("request failed", { method: options.method ?? "GET", path, status, message, visibleCookieNames: document.cookie.split(";").map((entry) => entry.trim().split("=")[0]).filter(Boolean) });
    }
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
