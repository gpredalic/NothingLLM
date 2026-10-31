"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  getWorkspace, listWorkspaceCollections, assignCollection, unassignCollection,
  listCollections, listUsers, listWorkspaceMembers, addMember, removeMember,
  Workspace, Collection, User, WorkspaceMember,
} from "@/lib/api";
import { toast } from "@/components/ui/toast";
import { Plus, Trash2, UserPlus, UserMinus } from "lucide-react";

export default function WorkspaceDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [members, setMembers] = useState<WorkspaceMember[]>([]);
  const [wsCollections, setWsCollections] = useState<string[]>([]);
  const [allCollections, setAllCollections] = useState<Collection[]>([]);
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [assignDialogOpen, setAssignDialogOpen] = useState(false);
  const [selectedCollection, setSelectedCollection] = useState("");
  const [memberDialogOpen, setMemberDialogOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState("");
  const [memberRole, setMemberRole] = useState("member");

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    Promise.allSettled([
      getWorkspace(id),
      listWorkspaceCollections(id),
      listCollections(),
      listUsers(),
      listWorkspaceMembers(id),
    ]).then(([ws, cols, all, users, mems]) => {
      if (ws.status === "fulfilled") setWorkspace(ws.value);
      if (cols.status === "fulfilled") setWsCollections(cols.value);
      if (all.status === "fulfilled") setAllCollections(all.value);
      if (users.status === "fulfilled") setAllUsers(users.value);
      if (mems.status === "fulfilled") setMembers(mems.value);
    }).finally(() => setLoading(false));
  }, [id]);

  useEffect(() => { load(); }, [load]);

  const handleAssign = async () => {
    if (!selectedCollection) return;
    setError("");
    try {
      await assignCollection(id, selectedCollection);
      toast("success", "Collection assigned");
      setAssignDialogOpen(false);
      setSelectedCollection("");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to assign collection");
    }
  };

  const handleUnassign = async (name: string) => {
    setError("");
    try {
      await unassignCollection(id, name);
      toast("success", "Collection removed");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to unassign collection");
    }
  };

  const handleAddMember = async () => {
    if (!selectedUser) return;
    setError("");
    try {
      await addMember(id, selectedUser, memberRole);
      setMemberDialogOpen(false);
      setSelectedUser("");
      setMemberRole("member");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to add member");
    }
  };

  const handleRemoveMember = async (userId: string) => {
    setError("");
    try {
      await removeMember(id, userId);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to remove member");
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="h-8 w-48 animate-pulse rounded bg-muted mb-4" />
        <div className="grid gap-6 lg:grid-cols-2">
          <div className="h-64 animate-pulse rounded-lg bg-muted" />
          <div className="h-64 animate-pulse rounded-lg bg-muted" />
        </div>
      </div>
    );
  }

  if (!workspace) {
    return <div className="p-6"><p className="text-destructive">Workspace not found.</p></div>;
  }

  const availableCollections = allCollections.filter(
    (c) => !wsCollections.includes(c.name)
  );
  const availableUsers = allUsers.filter(
    (u) => !members.find((m) => m.id === u.id)
  );

  return (
    <div className="p-6">
      <div className="mb-8">
        <h1 className="text-2xl font-semibold">{workspace.name}</h1>
        <p className="text-muted-foreground mt-1">{workspace.description || "No description"}</p>
      </div>

      {error && (
        <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Collections */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">Collections</CardTitle>
            <Button variant="outline" size="sm" onClick={() => setAssignDialogOpen(true)}>
              <Plus className="h-3 w-3 mr-1" /> Assign
            </Button>
          </CardHeader>
          <CardContent>
            {wsCollections.length === 0 ? (
              <p className="text-sm text-muted-foreground">No collections assigned.</p>
            ) : (
              <div className="space-y-2">
                {wsCollections.map((name) => {
                  const col = allCollections.find((c) => c.name === name);
                  return (
                    <div key={name} className="flex items-center justify-between rounded-md border p-3">
                      <div>
                        <div className="text-sm font-medium">{name}</div>
                        {col && (
                          <div className="text-xs text-muted-foreground">
                            {col.vector_count?.toLocaleString()} vectors • {col.dimensions}d
                          </div>
                        )}
                      </div>
                      <Button
                        variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground"
                        onClick={() => handleUnassign(name)}
                        title="Remove collection"
                      >
                        <Trash2 className="h-3 w-3" />
                      </Button>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Members */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">Members</CardTitle>
            <Button variant="outline" size="sm" onClick={() => setMemberDialogOpen(true)}>
              <UserPlus className="h-3 w-3 mr-1" /> Add
            </Button>
          </CardHeader>
          <CardContent>
            {members.length === 0 ? (
              <p className="text-sm text-muted-foreground">No members displayed.</p>
            ) : (
              <div className="space-y-2">
                {members.map((m) => (
                  <div key={m.id} className="flex items-center gap-3 rounded-md border p-3">
                    <div className="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-xs font-medium">
                      {m.email.slice(0, 2).toUpperCase()}
                    </div>
                    <div className="flex-1">
                      <div className="text-sm">{m.email}</div>
                    </div>
                    <Badge variant="secondary">{m.role}</Badge>
                    <Button
                      variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground"
                      onClick={() => handleRemoveMember(m.id)}
                      title="Remove member"
                    >
                      <UserMinus className="h-3 w-3" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Assign Collection Dialog */}
      <Dialog open={assignDialogOpen} onOpenChange={setAssignDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Assign Collection</DialogTitle>
          </DialogHeader>
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {availableCollections.length === 0 ? (
              <p className="text-sm text-muted-foreground">No available collections.</p>
            ) : (
              availableCollections.map((c) => (
                <button
                  key={c.name}
                  className={`w-full text-left rounded-md border p-3 text-sm transition-colors ${
                    selectedCollection === c.name ? "bg-accent border-primary" : "hover:bg-accent"
                  }`}
                  onClick={() => setSelectedCollection(c.name)}
                >
                  <div className="font-medium">{c.name}</div>
                  <div className="text-xs text-muted-foreground">
                    {c.vector_count?.toLocaleString()} vectors • {c.dimensions}d
                  </div>
                </button>
              ))
            )}
            <Button onClick={handleAssign} disabled={!selectedCollection} className="w-full">
              Assign
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Add Member Dialog */}
      <Dialog open={memberDialogOpen} onOpenChange={setMemberDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Member</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2 max-h-48 overflow-y-auto">
              {availableUsers.length === 0 ? (
                <p className="text-sm text-muted-foreground">No available users.</p>
              ) : (
                availableUsers.map((u) => (
                  <button
                    key={u.id}
                    className={`w-full text-left rounded-md border p-3 text-sm transition-colors ${
                      selectedUser === u.id ? "bg-accent border-primary" : "hover:bg-accent"
                    }`}
                    onClick={() => setSelectedUser(u.id)}
                  >
                    <div className="font-medium">{u.email}</div>
                    <div className="text-xs text-muted-foreground">{u.role}</div>
                  </button>
                ))
              )}
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Role</label>
              <select
                value={memberRole}
                onChange={(e) => setMemberRole(e.target.value)}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="member">Member</option>
                <option value="owner">Owner</option>
              </select>
            </div>
            <Button onClick={handleAddMember} disabled={!selectedUser} className="w-full">
              Add Member
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
