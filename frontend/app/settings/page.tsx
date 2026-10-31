"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { useTheme } from "@/components/theme-provider";
import { Moon, Sun, Monitor } from "lucide-react";
import { cn } from "@/lib/utils";

export default function SettingsPage() {
  const { theme, setTheme } = useTheme();

  return (
    <div className="p-6">
      <h1 className="text-2xl font-semibold mb-6">Settings</h1>

      <div className="max-w-lg space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Appearance</CardTitle>
            <CardDescription>Choose your preferred theme.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex gap-2">
              <button
                className={cn(
                  "flex items-center gap-2 rounded-md border p-3 text-sm",
                  theme === "light" ? "bg-accent" : "hover:bg-accent"
                )}
                onClick={() => setTheme("light")}
              >
                <Sun className="h-4 w-4" />
                Light
              </button>
              <button
                className={cn(
                  "flex items-center gap-2 rounded-md border p-3 text-sm",
                  theme === "dark" ? "bg-accent" : "hover:bg-accent"
                )}
                onClick={() => setTheme("dark")}
              >
                <Moon className="h-4 w-4" />
                Dark
              </button>
              <button
                className={cn(
                  "flex items-center gap-2 rounded-md border p-3 text-sm",
                  theme === "system" ? "bg-accent" : "hover:bg-accent"
                )}
                onClick={() => setTheme("system")}
              >
                <Monitor className="h-4 w-4" />
                System
              </button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">About</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm text-muted-foreground">
              <p>NothingLLM v0.1.0</p>
              <p>A minimal AI workspace manager.</p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}