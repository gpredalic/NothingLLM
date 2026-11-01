"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Loader2 } from "lucide-react";

export default function SetupPage() {
  const router = useRouter();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [adminEmail, setAdminEmail] = useState("");
  const [adminPassword, setAdminPassword] = useState("");

  // If setup is already complete, redirect to login
  useEffect(() => {
    fetch("/api/setup/status")
      .then((r) => {
        if (r.status === 404) {
          // Server is in full mode — setup routes don't exist
          router.replace("/login");
          return null;
        }
        return r.json();
      })
      .then((data) => {
        if (data && data.complete) {
          router.replace("/login");
        }
      })
      .catch(() => {});
  }, [router]);

  const handleComplete = async () => {
    setError("");
    setLoading(true);
    try {
      // Step 1: Create admin
      const adminRes = await fetch("/api/setup/admin", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: adminEmail, password: adminPassword }),
      });
      const adminData = await adminRes.json();
      if (!adminRes.ok) {
        throw new Error(adminData.error || "Failed to create admin");
      }

      // Step 2: Complete setup
      const completeRes = await fetch("/api/setup/complete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      const completeData = await completeRes.json();
      if (!completeRes.ok) {
        throw new Error(completeData.error || "Failed to complete setup");
      }

      // Redirect to login
      router.push("/login");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Setup failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-6">
          <h1 className="text-2xl font-semibold">NothingLLM</h1>
          <p className="text-sm text-muted-foreground mt-1">Initial Setup</p>
        </div>

        <Card>
          <CardHeader className="text-center">
            <CardTitle>Create Admin Account</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Email</label>
              <Input
                type="email"
                placeholder="admin@example.com"
                value={adminEmail}
                onChange={(e) => setAdminEmail(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Password</label>
              <Input
                type="password"
                placeholder="min 8 characters"
                value={adminPassword}
                onChange={(e) => setAdminPassword(e.target.value)}
                minLength={8}
              />
            </div>
            <Button
              className="w-full"
              onClick={handleComplete}
              disabled={loading || !adminEmail || !adminPassword}
            >
              {loading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              {loading ? "Setting up..." : "Complete Setup"}
            </Button>
          </CardContent>
        </Card>

        {error && <p className="text-sm text-destructive text-center mt-3">{error}</p>}
      </div>
    </div>
  );
}
