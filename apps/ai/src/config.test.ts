import { describe, it, expect } from "bun:test";
import { loadConfig } from "./config";

describe("loadConfig", () => {
  it("requires API_SERVICE_TOKEN in production", () => {
    expect(() => loadConfig({ NODE_ENV: "production" } as NodeJS.ProcessEnv)).toThrow(
      "API_SERVICE_TOKEN is required in production",
    );
  });

  it("allows a missing service token outside production", () => {
    expect(loadConfig({} as NodeJS.ProcessEnv).api.serviceToken).toBeNull();
  });
});
