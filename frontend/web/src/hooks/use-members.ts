import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Member } from "../api/types";
import { qk } from "./query-keys";

export function useMembers() {
  return useQuery({
    queryKey: qk.members,
    queryFn: () => apiGet<Member[]>("/api/v1/members"),
  });
}
