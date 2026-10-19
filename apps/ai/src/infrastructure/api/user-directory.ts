import type { ApiClient } from "@veemon/api-client";
import type { UserDirectory } from "../../domain/users";

// Adapter: implements the domain UserDirectory port over @veemon/api-client.
// The mapping to the lean UserSummary keeps only the fields the agent needs,
// which also limits how much PII enters the model context (see the tool's
// security note).
export function createApiUserDirectory(api: ApiClient): UserDirectory {
  return {
    async listUsers({ page, size }) {
      const res = await api.listUsers({ page, size });
      return {
        users: res.users.map((u) => ({
          id: u.id,
          email: u.email,
          name: u.name,
          status: u.status,
        })),
        // meta.total is protojson int64: a JSON string. The domain port uses a
        // number, so parse at the adapter boundary.
        total: res.pagination ? Number(res.pagination.total) : undefined,
      };
    },
  };
}
