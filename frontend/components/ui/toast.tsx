"use client";

import * as React from "react";
import { CheckCircle, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";

interface Toast {
  id: number;
  type: "success" | "error";
  message: string;
}

let toastId = 0;
const listeners: Array<(t: Toast) => void> = [];

export function toast(type: "success" | "error", message: string) {
  const t = { id: ++toastId, type, message };
  listeners.forEach((fn) => fn(t));
}

export function Toaster() {
  const [toasts, setToasts] = React.useState<Toast[]>([]);

  React.useEffect(() => {
    const fn = (t: Toast) => {
      setToasts((prev) => [...prev, t]);
      setTimeout(() => setToasts((prev) => prev.filter((x) => x.id !== t.id)), 3000);
    };
    listeners.push(fn);
    return () => {
      const i = listeners.indexOf(fn);
      if (i >= 0) listeners.splice(i, 1);
    };
  }, []);

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
      {toasts.map((t) => (
        <div
          key={t.id}
          className={cn(
            "flex items-center gap-2 rounded-md border px-4 py-3 text-sm shadow-lg animate-in slide-in-from-bottom-2",
            t.type === "success"
              ? "border-green-500/30 bg-green-50 text-green-800 dark:bg-green-950 dark:text-green-300"
              : "border-destructive/30 bg-red-50 text-red-800 dark:bg-red-950 dark:text-red-300"
          )}
        >
          {t.type === "success" ? (
            <CheckCircle className="h-4 w-4 shrink-0" />
          ) : (
            <XCircle className="h-4 w-4 shrink-0" />
          )}
          {t.message}
        </div>
      ))}
    </div>
  );
}