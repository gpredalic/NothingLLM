"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { listCollections, listWorkspaces, assignCollection, Collection, Workspace } from "@/lib/api";
import { RefreshCw, Search, FolderOpen } from "lucide-react";

export default function CollectionsPage() {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");
  const [assignTarget, setAssignTarget] = useState<Collection | null>(null);
  const [selectedWs, setSelectedWs] = useState("");
  const [assigning, setAssigning] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    Promise.allSettled([listCollections(), listWorkspaces()])
      .then(([cols, ws]) => {
        if (cols.status === "fulfilled") setCollections(cols.value.collections);
        if (ws.status === "fulfilled") setWorkspaces(ws.value.workspaces);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const filtered = useMemo(() => {
    if (!search.trim()) return collections;
    const q = search.toLowerCase();
    return collections.filter((c) => c.name.toLowerCase().includes(q));
  }, [collections, search]);

  const handleAssign = async () => {
    if (!assignTarget || !selectedWs) return;
    setAssigning(true);
    setError("");
    try {
      await assignCollection(selectedWs, assignTarget.name);
      setAssignTarget(null);
      setSelectedWs("");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to assign collection");
    } finally {
      setAssigning(false);
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Collections</h1>
        <Button variant="outline" size="sm" onClick={load}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      {error && (
        <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}

      {/* Search */}
      <div className="mb-4 relative max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Search collections..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-9"
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            Qdrant Collections
            <span className="ml-2 text-sm font-normal text-muted-foreground">
              {filtered.length} of {collections.length}
            </span>
          </CardTitle>
          <CardDescription>
            Vector collections synced from Qdrant. Assign to workspaces to enable RAG search.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-12 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : filtered.length === 0 ? (
            <div className="py-8 text-center">
              <p className="text-sm text-muted-foreground">
                {search ? "No collections match your search." : "No collections found. Ensure Qdrant is running and has data."}
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Vectors</TableHead>
                  <TableHead>Dimensions</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Last Sync</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.map((c) => (
                  <TableRow key={c.name}>
                    <TableCell className="font-medium">{c.name}</TableCell>
                    <TableCell>{c.vector_count?.toLocaleString() || 0}</TableCell>
                    <TableCell>{c.dimensions || "—"}</TableCell>
                    <TableCell>
                      <Badge variant={c.status === "online" ? "success" : "secondary"}>
                        {c.status || "unknown"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {c.last_synced_at ? new Date(c.last_synced_at).toLocaleString() : "Never"}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Button
                          variant="ghost" size="sm" className="h-8 text-xs"
                          onClick={() => setAssignTarget(c)}
                          disabled={workspaces.length === 0}
                        >
                          <FolderOpen className="h-3 w-3 mr-1" />
                          Assign
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Assign Dialog */}
      <Dialog open={!!assignTarget} onOpenChange={(open) => !open && setAssignTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Assign Collection to Workspace</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Assign <strong>{assignTarget?.name}</strong> to a workspace:
            </p>
            <div className="space-y-2">
              {workspaces.map((ws) => (
                <button
                  key={ws.id}
                  className={`w-full text-left rounded-md border p-3 text-sm transition-colors ${
                    selectedWs === ws.id ? "bg-accent border-primary" : "hover:bg-accent"
                  }`}
                  onClick={() => setSelectedWs(ws.id)}
                >
                  <div className="font-medium">{ws.name}</div>
                  <div className="text-xs text-muted-foreground">{ws.description || "No description"}</div>
                </button>
              ))}
            </div>
            <Button onClick={handleAssign} disabled={assigning || !selectedWs} className="w-full">
              {assigning ? "Assigning..." : "Assign"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
