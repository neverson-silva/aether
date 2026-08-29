export function extractReturnTo(value: string | undefined) {
  if (!value) return undefined;
  const match = value.match(/[?&]returnTo=(.*)$/i);
  if (!match) return undefined;
  try {
    return decodeURIComponent(match[1]);
  } catch {
    return undefined;
  }
}

export function normalizeLegacyKind(value: unknown) {
  if (typeof value !== "string") return undefined;
  const kind = value.trim().split(/[?&]/, 1)[0];
  return kind || undefined;
}

export function sanitizeReturnTo(value: string | undefined) {
  if (!value) return undefined;
  try {
    const decoded = decodeURIComponent(value);
    return decoded.startsWith("/") && !decoded.startsWith("//") ? decoded : undefined;
  } catch {
    return undefined;
  }
}

export function normalizeDetailsSearch(value: unknown) {
  if (!value || typeof value !== "object") return value;
  const input = value as { tab?: unknown; kind?: unknown; returnTo?: unknown };
  const rawTab = Array.isArray(input.tab) ? input.tab[0] : input.tab;
  const rawKind = Array.isArray(input.kind) ? input.kind[0] : input.kind;
  const embeddedReturnTo = extractReturnTo(typeof rawKind === "string" ? rawKind : undefined);
  const explicitReturnTo = typeof input.returnTo === "string" ? input.returnTo : undefined;
  let returnTo = explicitReturnTo ?? embeddedReturnTo;
  if (returnTo) {
    try {
      returnTo = decodeURIComponent(returnTo);
    } catch {
      returnTo = undefined;
    }
  }
  return { tab: rawTab, returnTo: sanitizeReturnTo(returnTo) };
}

export function readDetailsReturnTo(fallback: string | undefined) {
  const safeFallback = sanitizeReturnTo(fallback);
  if (safeFallback) return safeFallback;
  if (typeof window === "undefined") return "/apps";
  const rawKind = new URLSearchParams(window.location.search).get("kind") ?? undefined;
  return sanitizeReturnTo(extractReturnTo(rawKind)) ?? "/apps";
}
