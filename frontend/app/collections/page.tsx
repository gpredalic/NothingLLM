"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { listCollections } from "@/lib/api";
import type { Collection } from "@/lib/api";
import { RefreshCw } from "lucide-react";

export default function CollectionsPage() {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true);
    listCollections()
      .then((res) => setCollections(res.collections))
      .finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Collections</h1>
        <Button variant="outline" size="sm" onClick={load}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            Qdrant Collections
            <span className="ml-2 text-sm font-normal text-muted-foreground">
              {collections.length} total
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading...</p>
          ) : collections.length === 0 ? (
            <p className="text-sm text-muted-foreground">No collections found. Ensure Qdrant is running and has data.</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Vectors</TableHead>
                  <TableHead>Dimensions</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Last Sync</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {collections.map((c) => (
                  <TableRow key={c.name}>
                    <TableCell className="font-medium">{c.name}</TableCell>
                    <TableCell>{c.vector_count?.toLocaleString() || 0}</TableCell>
                    <TableCell>{c.dimensions || "—"}</TableCell>
                    <TableCell>
                      <Badge variant={c.status === "online" ? "success" : "warning"}>
                        {c.status || "unknown"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {c.last_synced_at ? new Date(c.last_synced_at).toLocaleString() : "Never"}
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