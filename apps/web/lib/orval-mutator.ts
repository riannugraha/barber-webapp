import { api } from "./api";

// Orval mutator — wraps ky instance for react-query codegen.
// Orval will import { customInstance } from "./lib/orval-mutator" and use it for all requests.
// Keeps ky (not Server Actions) and injects Authorization from localStorage via api hooks.
export const customInstance = async <T>(config: {
  url: string;
  method: string;
  params?: Record<string, unknown>;
  data?: unknown;
  headers?: Record<string, string>;
}): Promise<T> => {
  const { url, method, params, data, headers } = config;

  // Build query string if params exist
  let fullUrl = url;
  if (params && Object.keys(params).length > 0) {
    const sp = new URLSearchParams();
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== null && v !== "") sp.set(k, String(v));
    });
    const qs = sp.toString();
    if (qs) fullUrl = `${url}?${qs}`;
  }

  // api is ky instance with prefixUrl API_URL, so url should be relative
  const response = await api(fullUrl, {
    method: method.toUpperCase(),
    json: data || undefined,
    headers,
  });

  // ky returns Response; parse json if possible
  const text = await response.text();
  if (!text) return undefined as T;
  try {
    return JSON.parse(text) as T;
  } catch {
    return text as unknown as T;
  }
};
