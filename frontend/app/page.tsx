"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { listWorkspaces, listCollections, listUsers, getAdminStats, getHealth } from "@/lib/api";
import type { Workspace, Collection, User, AdminStats, HealthStatus } from "@/lib/api";
import { FolderKanban, Database, MessageSquare, Users, Activity, Cpu } from "lucide-react";
import { cn } from "@/lib/utils";

function SkeletonCard() {
  return (
    <Card>
      <CardHeader>
        <div className="h-4 w-24 animate-pulse rounded bg-muted" />
        <div className="h-3 w-32 animate-pulse rounded bg-muted" />
      </CardHeader>
      <CardContent>
        <div className="h-8 w-16 animate-pulse rounded bg-muted" />
      </CardContent>
    </Card>
  );
}

function StatusDot({ status }: { status: string }) {
  return (
    <span
      className={cn(
        "inline-block h-2 w-2 rounded-full",
        status === "ok" ? "bg-green-500" : status === "down" ? "bg-red-500" : "bg-yellow-500"
      )}
      title={status}
    />
  );
}

export default function DashboardPage() {
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [statsRes, healthRes] = await Promise.allSettled([
        getAdminStats(),
        getHealth(),
      ]);
      if (statsRes.status === "fulfilled") setStats(statsRes.value);
      if (healthRes.status === "fulfilled") setHealth(healthRes.value);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load dashboard");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  if (loading) {
    return (
      <div className="p-6">
        <div className="h-8 w-32 animate-pulse rounded bg-muted mb-6" />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
        </div>
        <div className="mt-4">
          <div className="h-4 w-48 animate-pulse rounded bg-muted mb-3" />
          <div className="h-8 w-full animate-pulse rounded bg-muted" />
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Dashboard</h1>
        <button onClick={load} className="text-sm text-muted-foreground hover:text-foreground transition-colors">
          Refresh
        </button>
      </div>

      {error && (
        <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Users</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.user_count ?? "—"}</div>
            <CardDescription>Registered users</CardDescription>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Workspaces</CardTitle>
            <FolderKanban className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.workspace_count ?? "—"}</div>
            <CardDescription>Active workspaces</CardDescription>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Collections</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.collection_count ?? "—"}</div>
            <CardDescription>
              {stats?.vector_count?.toLocaleString() ?? "0"} vectors
            </CardDescription>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Sessions</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.active_sessions ?? "—"}</div>
            <CardDescription>Current sessions</CardDescription>
          </CardContent>
        </Card>
      </div>

      {/* Service Health */}
      <div className="mt-6">
        <h2 className="text-sm font-medium text-muted-foreground mb-3 flex items-center gap-2">
          <Cpu className="h-4 w-4" /> Service Status
        </h2>
        <div className="grid gap-3 sm:grid-cols-3">
          {health && (
            <>
              <div className="flex items-center gap-2 rounded-md border p-3">
                <StatusDot status={health.database.status} />
                <span className="text-sm font-medium">Database</span>
                <span className="ml-auto text-xs text-muted-foreground capitalize">{health.database.status}</span>
              </div>
              <div className="flex items-center gap-2 rounded-md border p-3">
                <StatusDot status={health.qdrant.status} />
                <span className="text-sm font-medium">Qdrant</span>
                <span className="ml-auto text-xs text-muted-foreground capitalize">{health.qdrant.status}</span>
              </div>
              <div className="flex items-center gap-2 rounded-md border p-3">
                <StatusDot status={health.openai.status} />
                <span className="text-sm font-medium">OpenAI</span>
                <span className="ml-auto text-xs text-muted-foreground capitalize">{health.openai.status}</span>
              </div>
            </>
          )}
          {!health && (
            <div className="col-span-full text-sm text-muted-foreground">Health check unavailable</div>
          )}
        </div>
      </div>

      {/* Quick Links */}
      <div className="mt-6">
        <h2 className="text-sm font-medium text-muted-foreground mb-3">Quick Actions</h2>
        <div className="flex gap-3">
          <Link href="/chat" className="rounded-md border px-4 py-2 text-sm font-medium hover:bg-accent transition-colors">
            <MessageSquare className="h-4 w-4 inline mr-2" />
            Open Chat
          </Link>
          <Link href="/workspaces" className="rounded-md border px-4 py-2 text-sm font-medium hover:bg-accent transition-colors">
            <FolderKanban className="h-4 w-4 inline mr-2" />
            Manage Workspaces
          </Link>
          <Link href="/collections" className="rounded-md border px-4 py-2 text-sm font-medium hover:bg-accent transition-colors">
            <Database className="h-4 w-4 inline mr-2" />
            Browse Collections
          </Link>
        </div>
      </div>
    </div>
  );
}
