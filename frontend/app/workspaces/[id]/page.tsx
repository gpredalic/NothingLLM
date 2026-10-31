"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { getWorkspace, listWorkspaceCollections, assignCollection, unassignCollection, listCollections } from "@/lib/api";
import type { Workspace, WorkspaceCollection, Collection } from "@/lib/api";
import { Plus, Trash2 } from "lucide-react";

export default function WorkspaceDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const [workspace, setWorkspace] = useState<(Workspace & { members: Array<{ user_id: string; role: string }> }) | null>(null);
  const [collections, setCollections] = useState<WorkspaceCollection[]>([]);
  const [allCollections, setAllCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [selectedCollection, setSelectedCollection] = useState("");

  const load = () => {
    Promise.all([getWorkspace(id), listWorkspaceCollections(id), listCollections()])
      .then(([ws, cols, all]) => {
        setWorkspace(ws);
        setCollections(cols.collections);
        setAllCollections(all.collections);
      })
      .finally(() => setLoading(false));
  };

  useEffect(load, [id]);

  const handleAssign = async () => {
    if (!selectedCollection) return;
    await assignCollection(id, selectedCollection);
    setDialogOpen(false);
    setSelectedCollection("");
    load();
  };

  const handleUnassign = async (name: string) => {
    await unassignCollection(id, name);
    load();
  };

  if (loading) return <div className="p-6"><p className="text-muted-foreground">Loading...</p></div>;
  if (!workspace) return <div className="p-6"><p>Workspace not found.</p></div>;

  return (
    <div className="p-6">
      <div className="mb-8">
        <h1 className="text-2xl font-semibold">{workspace.name}</h1>
        <p className="text-muted-foreground mt-1">{workspace.description || "No description"}</p>
      </div>
      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">Collections</CardTitle>
            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
              <DialogTrigger asChild>
                <Button variant="outline" size="sm">
                  <Plus className="h-3 w-3 mr-1" /> Assign
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Assign Collection</DialogTitle>
                </DialogHeader>
                <div className="space-y-2">
                  {allCollections
                    .filter((c) => !collections.find((wc) => wc.collection.name === c.name))
                    .map((c) => (
                      <button key={c.name} className="w-full text-left rounded-md border p-3 text-sm hover:bg-accent" onClick={() => setSelectedCollection(c.name)}>
                        <div className="font-medium">{c.name}</div>
                        <div className="text-xs text-muted-foreground">{c.vector_count} vectors • {c.dimensions}d</div>
                      </button>
                    ))}
                  <Button onClick={handleAssign} disabled={!selectedCollection} className="w-full">Assign</Button>
                </div>
              </DialogContent>
            </Dialog>
          </CardHeader>
          <CardContent>
            {collections.length === 0 ? (
              <p className="text-sm text-muted-foreground">No collections assigned.</p>
            ) : (
              <div className="space-y-2">
                {collections.map((wc) => (
                  <div key={wc.collection.name} className="flex items-center justify-between rounded-md border p-3">
                    <div>
                      <div className="text-sm font-medium">{wc.collection.name}</div>
                      <div className="text-xs text-muted-foreground">{wc.collection.vector_count} vectors • {wc.collection.dimensions}d</div>
                    </div>
                    <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground" onClick={() => handleUnassign(wc.collection.name)}>
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Members</CardTitle>
          </CardHeader>
          <CardContent>
            {workspace.members.length === 0 ? (
              <p className="text-sm text-muted-foreground">No members.</p>
            ) : (
              <div className="space-y-2">
                {workspace.members.map((m) => (
                  <div key={m.user_id} className="flex items-center gap-3 rounded-md border p-3">
                    <div className="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-xs font-medium">
                      {m.user_id.slice(0, 2).toUpperCase()}
                    </div>
                    <div className="flex-1 text-sm">{m.user_id.slice(0, 8)}...</div>
                    <Badge variant="secondary">{m.role}</Badge>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}