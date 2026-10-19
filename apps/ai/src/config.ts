// Environment-driven configuration for the AI service.
//
// Config is validated once at startup (fail fast) and passed explicitly to the
// composition root — nothing reads process.env past this module.

export type CheckpointerConfig =
  | { kind: "memory" }
  | { kind: "postgres"; connectionString: string };

export interface Config {
  /** HTTP port the Hono server listens on. */
  port: number;
  /** Anthropic API key. Optional at boot; required when an agent actually runs. */
  anthropicApiKey: string | undefined;
  model: {
    /** Anthropic model id, e.g. "claude-opus-4-8". */
    name: string;
    maxTokens: number;
  };
  api: {
    /** Base URL of the Go API (apps/api) the tools/workflows call. */
    baseUrl: string;
    /** Optional service bearer token for authenticated API calls. */
    serviceToken: string | null;
  };
  /** Where LangGraph persists thread state (enables resume + interrupts). */
  checkpointer: CheckpointerConfig;
  /** "production" enables fail-fast checks that would be noise in dev. */
  environment: string;
}

function toInt(value: string | undefined, fallback: number): number {
  if (value === undefined || value.trim() === "") return fallback;
  const n = Number(value);
  if (!Number.isFinite(n)) throw new Error(`expected an integer, got "${value}"`);
  return Math.trunc(n);
}

export function loadConfig(env: NodeJS.ProcessEnv = process.env): Config {
  const kind = (env.CHECKPOINTER ?? "memory").toLowerCase();
  if (kind !== "memory" && kind !== "postgres") {
    throw new Error(`CHECKPOINTER must be "memory" or "postgres", got "${kind}"`);
  }

  let checkpointer: CheckpointerConfig;
  if (kind === "postgres") {
    const connectionString = env.DATABASE_URL;
    if (!connectionString) {
      throw new Error("CHECKPOINTER=postgres requires DATABASE_URL to be set");
    }
    checkpointer = { kind, connectionString };
  } else {
    checkpointer = { kind: "memory" };
  }

  const environment = (env.NODE_ENV ?? "development").toLowerCase();
  const serviceToken = env.API_SERVICE_TOKEN || null;
  // In production every agent tool and workflow calls the Go API, so a missing
  // token means every call would 401 at runtime instead of failing at boot.
  if (environment === "production" && !serviceToken) {
    throw new Error("API_SERVICE_TOKEN is required in production");
  }

  return {
    port: toInt(env.PORT, 4111),
    anthropicApiKey: env.ANTHROPIC_API_KEY || undefined,
    model: {
      name: env.AI_MODEL ?? "claude-opus-4-8",
      maxTokens: toInt(env.AI_MAX_TOKENS, 8192),
    },
    api: {
      baseUrl: env.API_BASE_URL ?? "http://localhost:3000",
      serviceToken,
    },
    checkpointer,
    environment,
  };
}
