import { afterEach, describe, expect, it, vi } from "vitest";
import fixture from "../../../../tests/contracts/foundation.json";
import { api, ApiError } from "./client";

afterEach(() => vi.unstubAllGlobals());
describe("统一请求边界", () => {
  it("发送同源Cookie，不解析204；登出携带CSRF", async () => {
    const fetch = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetch);
    const signal = new AbortController().signal;
    await expect(api.logout("memory-csrf", signal)).resolves.toBeUndefined();
    expect(fetch).toHaveBeenCalledWith("/api/v1/auth/logout", expect.objectContaining({ method: "POST", credentials: "same-origin", headers: { "X-CSRF-Token": "memory-csrf" }, signal }));
  });
  it("401和429保留类型、request id和retry-after，不自动重试", async () => {
    const fetch = vi.fn().mockResolvedValue(Response.json(fixture.unauthenticated, { status: 401 }));
    vi.stubGlobal("fetch", fetch);
    await expect(api.projects(new AbortController().signal)).rejects.toMatchObject({ status: 401, code: "unauthenticated", requestId: "test-request" });
    fetch.mockResolvedValue(Response.json(fixture.invalid_request, { status: 429, headers: { "Retry-After": "8" } }));
    await expect(api.login("not-real-password", new AbortController().signal)).rejects.toMatchObject({ status: 429, retryAfter: 8 });
    expect(fetch).toHaveBeenCalledTimes(2);
  });
  it("网络异常不虚构空列表", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("network")));
    await expect(api.terminals(new AbortController().signal)).rejects.toThrow(TypeError);
    expect(new ApiError(403, "forbidden", "拒绝访问", "id", null).status).toBe(403);
  });
});
