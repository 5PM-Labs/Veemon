// Thin typed REST client over the Go Fiber routes.
//
// Routes and shapes are derived from contract/user/user.proto (veemon.route
// options) and apps/api/handler/grpc/user/user_fiber.pb.go. The Go server wraps
// every REST response in the pkg/response envelope:
//   success: { success: true, data, meta? }
//   error:   { success: false, error: { code, message } }
// Field names are camelCase protojson. Response/message types are derived from
// the generated protobuf-es schema (exported as `proto` from the package index),
// so a proto change updates the client types instead of silently drifting.
// Outgoing request types use the schema's JSON shape (ints are numbers).

import * as proto from "./gen/user/user_pb.js";

export class ApiError extends Error {
  constructor(
    public readonly code: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// Message types come from the generated protobuf-es schema (contract/*.proto
// is the single source of truth) — never hand-written copies that can drift.
// `Omit<..., "$typeName">` drops the runtime brand from the brandless wire DTO.
// Responses use the schema type minus 64-bit fields, because the Go side emits
// protojson 64-bit integers as JSON *strings* (`"42"`), which the schema types
// as bigint. Receiving them must stay a string. Incoming requests are hand-kept
// to the JSON shapes the Go binders accept (proto3 JSON: ints are numbers,
// field names are json_name).
export type UserProfile = Omit<proto.UserProfile, "$typeName">;
export type Pagination = Omit<proto.Pagination, "$typeName" | "total"> & { total: string };
export type RegisterReq = Partial<Omit<proto.RegisterReq, "$typeName">>;
export type RegisterRes = Omit<proto.RegisterRes, "$typeName">;
export type LoginReq = Omit<proto.LoginReq, "$typeName">;
export type LoginRes = Omit<proto.LoginRes, "$typeName"> & { user?: UserProfile };
export type RefreshTokenRes = Omit<proto.RefreshTokenRes, "$typeName">;
export type LogoutRes = Omit<proto.LogoutRes, "$typeName">;

/** Query parameters for listUsers; the server clamps page/size. */
export type ListUsersQuery = Partial<Omit<proto.ListUsersReq, "$typeName">>;
export interface ListUsersResult {
  users: UserProfile[];
  pagination: Pagination | undefined;
}

interface Envelope<T> {
  success: boolean;
  data?: T;
  meta?: unknown;
  error?: { code: number; message: string };
}

export interface ApiClientOptions {
  /** Base URL of the Go API, e.g. http://localhost:3000 */
  baseUrl: string;
  /** Returns the PASETO bearer token (or null) for authenticated requests. */
  getToken?: () => string | null | Promise<string | null>;
  /** Override fetch (tests, non-browser runtimes). Defaults to globalThis.fetch. */
  fetch?: typeof fetch;
}

export function createApiClient(opts: ApiClientOptions) {
  const doFetch = opts.fetch ?? globalThis.fetch;
  const base = opts.baseUrl.replace(/\/$/, "");

  async function raw<T>(
    method: string,
    path: string,
    body?: unknown,
    auth = true,
  ): Promise<Envelope<T>> {
    const headers: Record<string, string> = {};
    if (body !== undefined) headers["Content-Type"] = "application/json";
    if (auth && opts.getToken) {
      const token = await opts.getToken();
      if (token) headers["Authorization"] = `Bearer ${token}`;
    }
    const res = await doFetch(`${base}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
    let json: Envelope<T>;
    try {
      json = (await res.json()) as Envelope<T>;
    } catch {
      throw new ApiError(res.status, res.statusText || "invalid response body");
    }
    if (!json.success) {
      throw new ApiError(
        json.error?.code ?? res.status,
        json.error?.message ?? res.statusText,
      );
    }
    return json;
  }

  async function request<T>(
    method: string,
    path: string,
    body?: unknown,
    auth = true,
  ): Promise<T> {
    return (await raw<T>(method, path, body, auth)).data as T;
  }

  return {
    register: (body: RegisterReq) =>
      request<RegisterRes>("POST", "/api/v1/auth/register", body, false),

    login: (body: LoginReq) =>
      request<LoginRes>("POST", "/api/v1/auth/login", body, false),

    // The server rotates the *presented* bearer token; RefreshTokenReq.token is
    // accepted for proto/JSON symmetry but not required.
    refreshToken: (token: string) =>
      request<RefreshTokenRes>("POST", "/api/v1/auth/refresh", { token }),

    getMe: () => request<UserProfile>("GET", "/api/v1/auth/me"),

    logout: () => request<LogoutRes>("POST", "/api/v1/auth/logout"),

    listUsers: async (query: ListUsersQuery = {}): Promise<ListUsersResult> => {
      const params = new URLSearchParams();
      if (query.page != null) params.set("page", String(query.page));
      if (query.size != null) params.set("size", String(query.size));
      if (query.search) params.set("search", query.search);
      if (query.sortBy) params.set("sortBy", query.sortBy);
      if (query.sortOrder) params.set("sortOrder", query.sortOrder);
      const qs = params.toString();
      const env = await raw<UserProfile[]>(
        "GET",
        `/api/v1/users${qs ? `?${qs}` : ""}`,
      );
      return {
        users: env.data ?? [],
        pagination: env.meta as Pagination | undefined,
      };
    },
  };
}

export type ApiClient = ReturnType<typeof createApiClient>;
