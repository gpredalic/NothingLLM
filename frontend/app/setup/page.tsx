"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Loader2 } from "lucide-react";

/** Safely extract an error message from a Response without assuming JSON. */
async function safeError(res: Response): Promise<string> {
  try {
    const data = await res.json();
    return data.error || `Request failed (${res.status})`;
  } catch {
    return `Request failed (${res.status})`;
  }
}

export default function SetupPage() {
  const router = useRouter();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [adminEmail, setAdminEmail] = useState("");
  const [adminPassword, setAdminPassword] = useState("");

  // Non-blocking: if an admin already exists, redirect to /login.
  useEffect(() => {
    let cancelled = false;
    fetch("/api/setup/status", { cache: "no-store" })
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        if (!cancelled && data && data.has_admin) {
          router.replace("/login");
        }
      })
      .catch(() => {});
    return () => { cancelled = true; };
  }, [router]);

  const handleComplete = async () => {
    setError("");

    if (!adminEmail || !adminPassword) {
      setError("Please fill in all fields.");
      return;
    }
    if (adminPassword.length < 8) {
      setError("Password must be at least 8 characters long.");
      return;
    }

    setLoading(true);
    try {
      const adminRes = await fetch("/api/setup/admin", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: adminEmail, password: adminPassword }),
      });
      if (!adminRes.ok) {
        const msg = await safeError(adminRes);
        throw new Error(msg);
      }

      const completeRes = await fetch("/api/setup/complete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (!completeRes.ok) {
        const msg = await safeError(completeRes);
        throw new Error(msg);
      }

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
              />
            </div>
            <Button
              className="w-full"
              onClick={handleComplete}
              disabled={loading}
            >
              {loading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              {loading ? "Creating..." : "Create Admin Account"}
            </Button>
          </CardContent>
        </Card>

        {error && <p className="text-sm text-destructive text-center mt-3">{error}</p>}
      </div>
    </div>
  );
}