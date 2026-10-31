"use client";

import { useState, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { listWorkspaces, chat } from "@/lib/api";
import type { Workspace, ChatMessage } from "@/lib/api";
import { Send, Copy, Check, RefreshCw } from "lucide-react";
import { Markdown } from "@/components/chat/markdown";
import { cn } from "@/lib/utils";

interface Message { role: "user" | "assistant"; content: string; }

export default function ChatPage() {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [activeWs, setActiveWs] = useState("");
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(-1);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    listWorkspaces().then((res) => {
      setWorkspaces(res.workspaces);
      if (res.workspaces.length > 0) setActiveWs(res.workspaces[0].id);
    });
  }, []);

  useEffect(() => { bottomRef.current?.scrollIntoView({ behavior: "smooth" }); }, [messages]);

  const handleSend = async () => {
    if (!input.trim() || !activeWs || loading) return;
    const userMsg: Message = { role: "user", content: input };
    const newMessages = [...messages, userMsg];
    setMessages(newMessages);
    setInput("");
    setLoading(true);
    try {
      const apiMsgs: ChatMessage[] = newMessages.map((m) => ({ role: m.role, content: m.content }));
      const res = await chat(activeWs, apiMsgs);
      setMessages([...newMessages, { role: "assistant", content: res.content }]);
    } catch (err) {
      setMessages([...newMessages, { role: "assistant", content: `Error: ${err instanceof Error ? err.message : "Failed"}` }]);
    } finally { setLoading(false); }
  };

  const handleRegenerate = async () => {
    if (!activeWs || loading || messages.length < 2) return;
    const trimmed = messages.slice(0, -1);
    setMessages(trimmed);
    setLoading(true);
    try {
      const apiMsgs: ChatMessage[] = trimmed.map((m) => ({ role: m.role, content: m.content }));
      const res = await chat(activeWs, apiMsgs);
      setMessages([...trimmed, { role: "assistant", content: res.content }]);
    } catch {
      setMessages([...trimmed, { role: "assistant", content: "Failed to regenerate." }]);
    } finally { setLoading(false); }
  };

  const handleCopy = (content: string, index: number) => {
    navigator.clipboard.writeText(content);
    setCopied(index);
    setTimeout(() => setCopied(-1), 2000);
  };

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-3 border-b p-4">
        <select value={activeWs} onChange={(e) => setActiveWs(e.target.value)}
          className="h-9 rounded-md border border-input bg-background px-3 py-1 text-sm">
          {workspaces.map((ws) => <option key={ws.id} value={ws.id}>{ws.name}</option>)}
        </select>
      </div>
      <div className="flex-1 overflow-y-auto p-4">
        {messages.length === 0 ? (
          <div className="flex h-full items-center justify-center">
            <p className="text-sm text-muted-foreground">Start a conversation</p>
          </div>
        ) : (
          <div className="mx-auto max-w-3xl space-y-6">
            {messages.map((msg, i) => (
              <div key={i} className={cn("flex gap-3", msg.role === "user" && "justify-end")}>
                {msg.role === "assistant" && (
                  <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground">AI</div>
                )}
                <div className={cn("max-w-[85%] space-y-2", msg.role === "user" && "text-right")}>
                  <div className={cn("rounded-lg px-4 py-3 text-sm",
                    msg.role === "user" ? "bg-primary text-primary-foreground" : "bg-card border")}>
                    {msg.role === "assistant" ? <Markdown content={msg.content} /> : <p>{msg.content}</p>}
                  </div>
                  {msg.role === "assistant" && msg.content && (
                    <div className="flex gap-1">
                      <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={() => handleCopy(msg.content, i)}>
                        {copied === i ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
                      </Button>
                      {i === messages.length - 1 && (
                        <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={handleRegenerate}>
                          <RefreshCw className="h-3 w-3" />
                        </Button>
                      )}
                    </div>
                  )}
                </div>
              </div>
            ))}
            {loading && (
              <div className="flex gap-3">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground">AI</div>
                <div className="rounded-lg border bg-card px-4 py-3">
                  <p className="text-sm text-muted-foreground animate-pulse">Thinking...</p>
                </div>
              </div>
            )}
            <div ref={bottomRef} />
          </div>
        )}
      </div>
      <div className="border-t p-4">
        <div className="mx-auto flex max-w-3xl gap-2">
          <Input value={input} onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && handleSend()}
            placeholder="Type your message..." disabled={loading} />
          <Button onClick={handleSend} disabled={loading || !input.trim()}>
            <Send className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
