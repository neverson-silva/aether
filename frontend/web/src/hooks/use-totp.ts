import { apiDelete, apiPost } from "../api/client";

export function useTOTP() {
  return {
    enroll: () => apiPost<{ secret: string; uri: string }>("/api/v1/auth/totp/enroll"),
    verify: (code: string) => apiPost("/api/v1/auth/totp/verify", { code }),
    disable: () => apiDelete("/api/v1/auth/totp"),
  };
}
