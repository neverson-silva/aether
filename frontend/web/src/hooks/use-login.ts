import { useMutation } from "@tanstack/react-query";
import { apiPost, setServer } from "../api/client";
import type { LoginResponse } from "../api/types";

export function useLogin() {
  return useMutation({
    mutationFn: async ({
      email,
      password,
      server,
    }: {
      email: string;
      password: string;
      server: string;
    }) => {
      setServer(server);
      const data = await apiPost<LoginResponse>("/api/v1/auth/login", {
        email,
        password,
      });
      return data;
    },
  });
}
