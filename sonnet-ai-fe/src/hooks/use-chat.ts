import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { z } from "zod";
import api from "../api";
import { ToastType } from "./use-toast";
import { MessageType, SessionType } from "../types";

export function useChat(showToast: (text: string, type: ToastType) => void) {
  const [sessions, setSessions] = useState<Record<string, SessionType>>({});
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);
  const [tempSession, setTempSession] = useState(false);
  const [currentMessages, setCurrentMessages] = useState<MessageType[]>([]);
  const [inputMessage, setInputMessage] = useState("");
  const [loading, setLoading] = useState(false);
  const [selectedModel, setSelectedModel] = useState("ollama");
  const [isStreaming, setIsStreaming] = useState(true);

  useQuery({
    queryKey: ["sessions"],
    queryFn: async () => {
      const response = await api.get<{
        status_code: number;
        status_msg: string;
        sessions: unknown;
      }>("/ai/chat/sessions");
      if (response.data && response.data.status_code === 1000) {
        const sessionMap: Record<string, SessionType> = {};
        const sessionsData = z
          .array(
            z.object({
              session_id: z.union([z.string(), z.number()]),
              name: z.string().optional(),
            }),
          )
          .safeParse(response.data.sessions);
        if (sessionsData.success) {
          sessionsData.data.forEach((s) => {
            const sessionId = String(s.session_id);
            sessionMap[sessionId] = {
              id: sessionId,
              name: s.name || `Session ${sessionId}`,
              messages: [],
            };
          });
        }
        setSessions(sessionMap);
      }
      return response.data;
    },
  });

  const sessionList = useMemo(() => Object.values(sessions), [sessions]);

  const createNewSession = () => {
    setCurrentSessionId("temp");
    setTempSession(true);
    setCurrentMessages([]);
  };

  const switchSession = async (sessionId: string) => {
    if (!sessionId) return;
    setCurrentSessionId(String(sessionId));
    setTempSession(false);

    if (
      !sessions[sessionId]?.messages ||
      sessions[sessionId].messages.length === 0
    ) {
      try {
        const response = await api.post<{
          status_code: number;
          status_msg: string;
          history: unknown;
        }>("/ai/chat/history", {
          session_id: String(sessionId),
        });
        if (response.data && response.data.status_code === 1000) {
          const historyData = z
            .array(z.object({ is_user: z.boolean(), content: z.string() }))
            .safeParse(response.data.history);
          if (historyData.success) {
            const msgs: MessageType[] = historyData.data.map((item) => ({
              role: item.is_user ? "user" : "assistant",
              content: item.content,
            }));
            setSessions((prev) => ({
              ...prev,
              [sessionId]: { ...prev[sessionId], messages: msgs },
            }));
            setCurrentMessages(msgs);
          }
        }
      } catch (err: unknown) {
        console.error("Load history error:", err);
      }
    } else {
      setCurrentMessages([...(sessions[sessionId].messages || [])]);
    }
  };

  const handleStreaming = async (question: string) => {
    const aiMessage: MessageType = {
      role: "assistant",
      content: "",
      meta: { status: "streaming" },
    };
    setCurrentMessages((prev) => [...prev, aiMessage]);

    const url = tempSession
      ? "/api/v1/ai/chat/send-stream-new-session"
      : "/api/v1/ai/chat/send-stream";
    const body = tempSession
      ? { question, model_type: selectedModel }
      : { question, model_type: selectedModel, session_id: currentSessionId };

    try {
      const response = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${localStorage.getItem("token") || ""}`,
        },
        body: JSON.stringify(body),
      });

      if (!response.ok) throw new Error("Network response was not ok");
      const reader = response.body?.getReader();
      if (!reader) throw new Error("No reader");

      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";

        for (const line of lines) {
          if (!line.trim()) continue;

          if (line.startsWith("data:")) {
            let data = line.slice(5);
            if (data.startsWith(" ")) {
              data = data.slice(1); // SSE standard allows one optional space after colon
            }

            if (data === "[DONE]") {
              setCurrentMessages((prev) => {
                const next = [...prev];
                next[next.length - 1].meta = { status: "done" };
                return next;
              });
              setLoading(false);
            } else if (data.startsWith("{")) {
              try {
                const parsed = JSON.parse(data);
                if (parsed.sessionId) {
                  const newSid = String(parsed.sessionId);
                  if (tempSession) {
                    setSessions((prev) => ({
                      ...prev,
                      [newSid]: {
                        id: newSid,
                        name: "New Session",
                        messages: [],
                      },
                    }));
                    setCurrentSessionId(newSid);
                    setTempSession(false);
                  }
                } else if (parsed.content !== undefined) {
                  setCurrentMessages((prev) => {
                    const next = [...prev];
                    next[next.length - 1].content += parsed.content;
                    return next;
                  });
                }
              } catch (e: unknown) {
                console.error("Parse data error:", e);
              }
            } else {
              // Fallback for raw text data (if any)
              setCurrentMessages((prev) => {
                const next = [...prev];
                next[next.length - 1].content += data;
                return next;
              });
            }
          }
        }
      }
      setLoading(false);
    } catch (err: unknown) {
      console.error("Stream error:", err);
      setLoading(false);
      showToast("Stream transfer failed", "error");
    }
  };

  const handleNormal = async (question: string) => {
    if (tempSession) {
      const response = await api.post<{
        status_code: number;
        status_msg: string;
        session_id: string;
        information: string;
      }>("/ai/chat/send-new-session", {
        question,
        model_type: selectedModel,
      });
      if (response.data && response.data.status_code === 1000) {
        const sessionId = String(response.data.session_id);
        const aiMessage: MessageType = {
          role: "assistant",
          content: response.data.information || "",
        };
        setSessions((prev) => ({
          ...prev,
          [sessionId]: {
            id: sessionId,
            name: "New Session",
            messages: [{ role: "user", content: question }, aiMessage],
          },
        }));
        setCurrentSessionId(sessionId);
        setTempSession(false);
        setCurrentMessages([{ role: "user", content: question }, aiMessage]);
      } else {
        showToast(response.data?.status_msg || "Send Failed", "error");
        setCurrentMessages((prev) => prev.slice(0, -1));
      }
    } else {
      const response = await api.post<{
        status_code: number;
        status_msg: string;
        information: string;
      }>("/ai/chat/send", {
        question,
        model_type: selectedModel,
        session_id: currentSessionId,
      });
      if (response.data && response.data.status_code === 1000) {
        const aiMessage: MessageType = {
          role: "assistant",
          content: response.data.information || "",
        };
        setCurrentMessages((prev) => [...prev, aiMessage]);
      } else {
        showToast(response.data?.status_msg || "Send Failed", "error");
        setCurrentMessages((prev) => prev.slice(0, -1));
      }
    }
  };

  const sendMessage = async (e?: React.KeyboardEvent | React.MouseEvent) => {
    if (e) e.preventDefault();
    if (!inputMessage.trim()) {
      showToast("Please input message content", "warning");
      return;
    }

    const currentInput = inputMessage;
    setInputMessage("");
    setCurrentMessages((prev) => [
      ...prev,
      { role: "user", content: currentInput },
    ]);
    setLoading(true);

    if (isStreaming) {
      await handleStreaming(currentInput);
    } else {
      await handleNormal(currentInput);
      setLoading(false);
    }
  };

  return {
    sessions,
    sessionList,
    currentSessionId,
    tempSession,
    currentMessages,
    inputMessage,
    setInputMessage,
    loading,
    selectedModel,
    setSelectedModel,
    isStreaming,
    setIsStreaming,
    createNewSession,
    switchSession,
    sendMessage,
  };
}
