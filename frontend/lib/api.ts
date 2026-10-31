const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    credentials: "include",
    ...options,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || `Request failed: ${res.status}`);
  }
  return res.json();
}

// Auth
export async function register(email: string, password: string, name?: string) {
  return request<{ id: string; email: string }>("/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password, name }),
  });
}

export async function login(email: string, password: string) {
  return request<{ ok: boolean }>("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
}

export async function logout() {
  return request<{ ok: boolean }>("/api/auth/logout", { method: "POST" });
}

// Users
export async function listUsers() {
  return request<{ users: Array<{ id: string; email: string; name: string; role: string; is_active: boolean; created_at: string }> }>("/api/users");
}

export async function getUser(id: string) {
  return request<{ id: string; email: string; name: string; role: string; is_active: boolean; created_at: string }>(`/api/users/${id}`);
}

// Workspaces
export interface Workspace {
  id: string;
  name: string;
  description: string;
  created_at: string;
}

export interface WorkspaceMember {
  user_id: string;
  role: string;
}

export async function listWorkspaces() {
  return request<{ workspaces: Workspace[] }>("/api/workspaces");
}

export async function getWorkspace(id: string) {
  return request<Workspace & { members: WorkspaceMember[] }>(`/api/workspaces/${id}`);
}

export async function createWorkspace(name: string, description?: string) {
  return request<{ id: string; name: string }>("/api/workspaces", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, description }),
  });
}

export async function deleteWorkspace(id: string) {
  return request<{ ok: boolean }>(`/api/workspaces/${id}`, { method: "DELETE" });
}

// Collections
export interface Collection {
  name: string;
  vector_count: number;
  dimensions: number;
  status: string;
  last_synced_at: string;
}

export interface WorkspaceCollection {
  collection: Collection;
  role: string;
}

export async function listCollections() {
  return request<{ collections: Collection[] }>("/api/collections");
}

export async function listWorkspaceCollections(wsId: string) {
  return request<{ collections: WorkspaceCollection[] }>(`/api/workspaces/${wsId}/collections`);
}

export async function assignCollection(wsId: string, name: string) {
  return request<{ ok: boolean }>(`/api/workspaces/${wsId}/collections/${name}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });
}

export async function unassignCollection(wsId: string, name: string) {
  return request<{ ok: boolean }>(`/api/workspaces/${wsId}/collections/${name}`, {
    method: "DELETE",
  });
}

// RAG
export interface ChatMessage {
  role: string;
  content: string;
}

export async function search(wsId: string, query: string, topK?: number) {
  return request<{ id: string; score: number; payload: Record<string, unknown> }[]>(
    `/api/workspaces/${wsId}/search`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ query, top_k: topK || 5 }),
    }
  );
}

export async function chat(
  wsId: string,
  messages: ChatMessage[],
  modelId?: string
): Promise<{ content: string; model: string }> {
  return request<{ content: string; model: string }>(`/api/workspaces/${wsId}/chat`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ messages, model_id: modelId }),
  });
}