import { describe, it, expect } from "bun:test";
import { createApiClient, ApiError } from "./client.js";

function mockFetch(
  status: number,
  payload: unknown,
  capture?: (init: RequestInit) => void,
): typeof fetch {
  return (async (_url: string, init: RequestInit = {}) => {
    capture?.(init);
    return new Response(JSON.stringify(payload), {
      status,
      headers: { "Content-Type": "application/json" },
    });
  }) as unknown as typeof fetch;
}

const profile = {
  id: "1",
  email: "a@b.c",
  name: "A",
  phone: "",
  status: "active",
  createdAt: "",
};

describe("api-client", () => {
  it("returns data on a success envelope (getMe)", async () => {
    const c = createApiClient({
      baseUrl: "http://x",
      getToken: () => "t",
      fetch: mockFetch(200, { success: true, data: profile }),
    });
    const me = await c.getMe();
    expect(me.email).toBe("a@b.c");
  });

  it("throws ApiError with code+message on a failure envelope", async () => {
    const c = createApiClient({
      baseUrl: "http://x",
      getToken: () => "t",
      fetch: mockFetch(400, {
        success: false,
        error: { code: 40001, message: "bad" },
      }),
    });
    const err = (await c.getMe().catch((e) => e)) as ApiError;
    expect(err).toBeInstanceOf(ApiError);
    expect(err.code).toBe(40001);
    expect(err.message).toBe("bad");
  });

  it("attaches the bearer token when getToken is provided", async () => {
    let seen = "";
    const c = createApiClient({
      baseUrl: "http://x",
      getToken: () => "tok",
      fetch: mockFetch(200, { success: true, data: profile }, (init) => {
        seen = (init.headers as Record<string, string>)?.["Authorization"] ?? "";
      }),
    });
    await c.getMe();
    expect(seen).toBe("Bearer tok");
  });

  it("does not send Authorization for public endpoints (login)", async () => {
    let seen = "none";
    const c = createApiClient({
      baseUrl: "http://x",
      getToken: () => "tok",
      fetch: mockFetch(
        200,
        { success: true, data: { token: "jwt", user: profile } },
        (init) => {
          seen = (init.headers as Record<string, string>)?.["Authorization"] ?? "";
        },
      ),
    });
    const res = await c.login({ email: "a@b.c", password: "x" });
    expect(res.token).toBe("jwt");
    expect(seen).toBe("");
  });

  it("sends refresh as { token } (RefreshTokenReq.token)", async () => {
    let body: unknown;
    const c = createApiClient({
      baseUrl: "http://x",
      getToken: () => "t",
      fetch: mockFetch(200, { success: true, data: { token: "new" } }, (init) => {
        body = JSON.parse(String(init.body));
      }),
    });
    await c.refreshToken("old");
    expect(body).toEqual({ token: "old" });
  });

  it("splits a list response into users + pagination", async () => {
    const c = createApiClient({
      baseUrl: "http://x",
      getToken: () => "t",
      fetch: mockFetch(200, {
        success: true,
        data: [profile],
        // protojson emits int64 as a JSON string.
        meta: { page: 1, size: 10, total: "1", totalPages: 1 },
      }),
    });
    const res = await c.listUsers({ page: 1 });
    expect(res.users).toHaveLength(1);
    expect(res.pagination?.total).toBe("1");
  });
});
