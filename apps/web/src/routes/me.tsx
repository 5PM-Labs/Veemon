import { useEffect, useState } from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { ApiError } from "@veemon/api-client";
import { api } from "../lib/api";
import { clearToken, getToken } from "../lib/auth";

export const Route = createFileRoute("/me")({
  component: MePage,
});

function MePage() {
  const navigate = useNavigate();
  // Token reads are async (Tauri store / sessionStorage), so start "unknown"
  // until the first read settles — never render "signed in" then flip.
  const [hasToken, setHasToken] = useState<boolean | null>(null);

  useEffect(() => {
    let cancelled = false;
    void getToken().then((t) => {
      if (!cancelled) setHasToken(t !== null);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const me = useQuery({
    queryKey: ["me"],
    queryFn: () => api.getMe(),
    enabled: hasToken === true,
    retry: false,
  });

  const users = useQuery({
    queryKey: ["users"],
    queryFn: () => api.listUsers({ page: 1, size: 10 }),
    enabled: hasToken === true,
    retry: false,
  });

  async function logout() {
    await api.logout().catch(() => undefined);
    await clearToken();
    await navigate({ to: "/login" });
  }

  if (hasToken === null) return <p>Loading…</p>;

  if (!hasToken) {
    return (
      <div>
        <p>You are not signed in.</p>
        <button onClick={() => void navigate({ to: "/login" })}>
          Go to login
        </button>
      </div>
    );
  }

  if (me.isLoading) return <p>Loading…</p>;

  if (me.error) {
    const msg =
      me.error instanceof ApiError ? me.error.message : "Failed to load profile";
    return (
      <div>
        <p className="error">{msg}</p>
        <button onClick={() => void logout()}>Back to login</button>
      </div>
    );
  }

  return (
    <div>
      <h2>Signed in</h2>
      <pre>{JSON.stringify(me.data, null, 2)}</pre>

      <h3>Users</h3>
      {users.error ? (
        <p className="error">Cannot list users (requires admin role).</p>
      ) : (
        <ul>
          {users.data?.users.map((u) => (
            <li key={u.id}>
              {u.email} — {u.status}
            </li>
          ))}
        </ul>
      )}

      <button onClick={() => void logout()}>Log out</button>
    </div>
  );
}
