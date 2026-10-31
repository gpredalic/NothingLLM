"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { listModels, updateModel, setDefaultModel, Model } from "@/lib/api";
import { toast } from "@/components/ui/toast";
import { Cpu, RefreshCw, Star } from "lucide-react";

export default function ModelsPage() {
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [toggling, setToggling] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    listModels()
      .then((res) => setModels(res))
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load models"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleToggle = async (model: Model) => {
    setToggling(model.id);
    setError("");
    try {
      await updateModel(model.id, !model.enabled);
      setModels((prev) =>
        prev.map((m) => (m.id === model.id ? { ...m, enabled: !m.enabled } : m))
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to update model");
    } finally {
      setToggling(null);
    }
  };

  const handleSetDefault = async (model: Model) => {
    setError("");
    try {
      await setDefaultModel(model.id);
      toast("success", `${model.id} set as default model`);
      setModels((prev) =>
        prev.map((m) => ({ ...m, is_default: m.id === model.id }))
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to set default model");
    }
  };

  const enabledCount = models.filter((m) => m.enabled).length;

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold">Models</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {models.length} discovered • {enabledCount} enabled
          </p>
        </div>
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

      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Cpu className="h-4 w-4" />
            Model Registry
          </CardTitle>
          <CardDescription>
            Models are automatically synced from OpenAI. Enable or disable models to control availability.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {[1, 2, 3, 4, 5].map((i) => (
                <div key={i} className="h-12 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : models.length === 0 ? (
            <div className="py-8 text-center">
              <p className="text-sm text-muted-foreground">
                No models found. Models are synced automatically when the OpenAI connection is active.
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Model ID</TableHead>
                  <TableHead>Context Window</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Last Synced</TableHead>
                  <TableHead className="text-right">Default</TableHead>
                  <TableHead className="text-right">Enabled</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {models.map((m) => (
                  <TableRow key={m.id}>
                    <TableCell className="font-mono text-sm">{m.id}</TableCell>
                    <TableCell>
                      {m.context_window ? `${(m.context_window / 1000).toFixed(0)}K tokens` : "—"}
                    </TableCell>
                    <TableCell>
                      <Badge variant={m.enabled ? "success" : "secondary"}>
                        {m.enabled ? "Enabled" : "Disabled"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {m.synced_at ? new Date(m.synced_at).toLocaleDateString() : "—"}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Button
                          variant="ghost" size="sm" className="h-8 w-8 p-0"
                          onClick={() => handleSetDefault(m)}
                          disabled={!m.enabled}
                          title={m.is_default ? "Current default" : "Set as default"}
                        >
                          <Star className={`h-4 w-4 ${m.is_default ? "fill-yellow-400 text-yellow-400" : "text-muted-foreground"}`} />
                        </Button>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Switch
                          checked={m.enabled}
                          onCheckedChange={() => handleToggle(m)}
                          disabled={toggling === m.id}
                        />
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
