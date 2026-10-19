import { Hono } from "hono";
import { streamSSE } from "hono/streaming";
import type { Context } from "hono";
import type { Container } from "../../composition";
import type { Graph } from "../../application/graph";

type Body = Record<string, unknown>;

// Request-validation failures carry their own message (safe to echo). Anything
// else — model/API errors, stack-message leakage — is replaced with a generic
// message; the detail is logged server-side only.
class BadRequestError extends Error {}

// Public message for a thrown value. Never sends internal error text to the
// client unless it was raised as a deliberate validation error.
function publicMessage(err: unknown, logger: (msg: string) => void): string {
  if (err instanceof BadRequestError) return err.message;
  logger(err instanceof Error ? (err.stack ?? err.message) : String(err));
  return "internal error";
}
type InputMapper = (body: Body) => unknown;

async function readBody(c: Context): Promise<Body> {
  return (await c.req.json().catch(() => ({}))) as Body;
}

function threadIdOf(body: Body): string {
  return typeof body.thread_id === "string" && body.thread_id ? body.thread_id : crypto.randomUUID();
}

// Agents are chat-shaped: { message, thread_id? } -> a LangGraph messages input.
const agentInput: InputMapper = (body) => {
  if (typeof body.message !== "string" || body.message.trim() === "") {
    throw new BadRequestError("`message` (non-empty string) is required");
  }
  return { messages: [{ role: "user", content: body.message }] };
};

// Workflows take a typed input object: { input, thread_id? }.
const workflowInput: InputMapper = (body) =>
  body.input && typeof body.input === "object" ? body.input : {};

function invokeHandler(registry: Record<string, Graph>, mapInput: InputMapper, logError: (msg: string) => void) {
  return async (c: Context) => {
    const graph = registry[c.req.param("name") ?? ""];
    if (!graph) return c.json({ error: "not found" }, 404);
    const body = await readBody(c);
    const threadId = threadIdOf(body);
    try {
      const result = await graph.invoke(mapInput(body), { threadId });
      return c.json({ thread_id: threadId, result });
    } catch (err) {
      return c.json({ error: publicMessage(err, logError) }, 400);
    }
  };
}

function streamHandler(registry: Record<string, Graph>, mapInput: InputMapper, logError: (msg: string) => void) {
  return (c: Context) => {
    const graph = registry[c.req.param("name") ?? ""];
    if (!graph) return c.json({ error: "not found" }, 404);
    return streamSSE(c, async (stream) => {
      const body = await readBody(c);
      const threadId = threadIdOf(body);
      await stream.writeSSE({ event: "start", data: JSON.stringify({ thread_id: threadId }) });
      try {
        const events = await graph.stream(mapInput(body), { threadId });
        for await (const chunk of events) {
          await stream.writeSSE({ event: "update", data: JSON.stringify(chunk) });
        }
        await stream.writeSSE({ event: "done", data: "{}" });
      } catch (err) {
        await stream.writeSSE({
          event: "error",
          data: JSON.stringify({ error: publicMessage(err, logError) }),
        });
      }
    });
  };
}

// Builds the Hono app exposing every agent and workflow over REST + SSE.
export function buildServer(container: Container): Hono {
  const app = new Hono();
  const logError = (msg: string) => console.error("[ai] request failed:", msg);

  app.get("/health", (c) =>
    c.json({
      status: "ok",
      agents: Object.keys(container.agents),
      workflows: Object.keys(container.workflows),
    }),
  );

  app.post("/agents/:name/invoke", invokeHandler(container.agents, agentInput, logError));
  app.post("/agents/:name/stream", streamHandler(container.agents, agentInput, logError));
  app.post("/workflows/:name/invoke", invokeHandler(container.workflows, workflowInput, logError));
  app.post("/workflows/:name/stream", streamHandler(container.workflows, workflowInput, logError));

  return app;
}
