---
name: proto-routes
description: Add or change a REST/gRPC endpoint in the Go backend by editing the proto contract and regenerating Fiber routes. Use whenever a task needs a new route, a changed path/method, an auth-policy change, or a rate-limit tweak in the Go backend.
---

# Declaring routes in proto (protoc-gen-fiber)

The REST surface is declared **directly on each RPC** in the `.proto`; the Go
Fiber routes are generated. There is **no hand-written route table or auth map**.
Method, path, binding, auth policy, and rate limits all live in the contract.

## Steps

1. **Edit the contract.** In `contract/<svc>/<svc>.proto`, define the RPC +
   messages and annotate the method with a `veemon.route` option:

   ```proto
   import "veemon/annotations.proto";

   service UserApi {
       rpc GetUser(GetUserReq) returns (UserProfile) {
           option (veemon.route) = {
               method: "GET"
               path: "/api/v1/users/{id}"          // {id} binds to request field `id`
               auth: { required: true roles: ["admin", "superadmin"] }
           };
       }

       rpc Register(RegisterReq) returns (RegisterRes) {
           option (veemon.route) = {
               method: "POST"
               path: "/api/v1/auth/register"
               body: true                          // parse JSON body into the request
               response: RESPONSE_STYLE_CREATED    // 201 instead of 200
               rate_limit: { max: 10 window_seconds: 60 }
           };
       }
   }
   ```

2. **Regenerate.** Run `make proto` (or `bun run proto` from the repo root). This
   runs `protoc-gen-go` / `-go-grpc` / `-fiber` via buf and emits
   `internal/handler/grpc/<svc>/<svc>_fiber.pb.go` containing:
   - `Register<Svc>Routes(router, srv, validator)` — wires every route onto Fiber,
     applying auth middleware + rate limiters, binding path params (`{id}` → `:id`),
     query params (for `GET`), and JSON body, then writing the response.
   - `<Svc>AuthConfig` — the gRPC full-method → auth policy map consumed by the
     gRPC auth interceptor, so **gRPC and REST enforce the same rules**.

3. **Implement the method** on the handler (`handler/user_handler.go`).

4. **Wire-up** already happens in `config/bootstrap.go` — no route file to touch.

## Rules

- Never add a Fiber route or edit the auth map by hand — change the proto and
  regenerate (see [architecture](../../rules/architecture.md)).
- Auth is **fail-closed**: a route with no explicit policy panics at startup.
- Options reference (`contract/veemon/annotations.proto`): `method`, `path`,
  `body`, `auth { required, roles }`, `response`
  (`RESPONSE_STYLE_OK|_CREATED|_LIST`), `rate_limit { max, window_seconds }`.
