import { useState, useEffect } from "react";
import { Sparkles } from "lucide-react";
import { MessageType } from "../types";
import MarkdownMessage from "./markdown-message";

interface ChatAreaProps {
  currentMessages: MessageType[];
  messagesRef: React.RefObject<HTMLDivElement | null>;
}

function getGreeting() {
  const hour = new Date().getHours();
  if (hour >= 5 && hour < 12) return "Good morning";
  if (hour >= 12 && hour < 18) return "Good afternoon";
  return "Good evening";
}

function ChatArea({ currentMessages, messagesRef }: ChatAreaProps) {
  const [greeting, setGreeting] = useState(getGreeting());

  useEffect(() => {
    // Optional: Update greeting if left open for a long time
    const interval = setInterval(() => {
      setGreeting(getGreeting());
    }, 60000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="flex flex-1 flex-col overflow-y-auto" ref={messagesRef}>
      {currentMessages.length === 0 ? (
        <div className="mx-auto flex w-full max-w-3xl flex-1 flex-col items-center justify-center px-6 pb-[20vh]">
          <div className="flex items-center justify-center gap-3">
            <Sparkles className="text-orange-500" size={32} />
            <h2 className="text-base-content/90 text-3xl">
              {greeting}, Master
            </h2>
          </div>
        </div>
      ) : (
        <div className="mx-auto flex w-full max-w-4xl flex-col gap-6 px-4 py-12 pb-36 md:px-12">
          {currentMessages.map((msg, index) => (
            <div
              key={index}
              className={`flex max-w-[85%] flex-col ${msg.role === "user" ? "ml-auto items-end" : "mr-auto items-start"}`}
            >
              <div className="mb-1 flex items-center gap-2 px-1">
                <span className="text-base-content/50 text-xs font-medium">
                  {msg.role === "user" ? "You" : "lark"}
                </span>
              </div>
              <div
                className={`rounded-2xl px-5 py-3.5 text-[15px] leading-relaxed ${
                  msg.role === "user"
                    ? "bg-base-300 text-base-content rounded-tr-sm"
                    : "bg-base-100 border-base-200 text-base-content rounded-tl-sm border shadow-sm"
                }`}
              >
                <MarkdownMessage content={msg.content} />
              </div>
              {msg.meta?.status === "streaming" && (
                <div className="text-base-content/40 mt-2 flex items-center gap-1 text-xs italic">
                  <span className="loading loading-dots loading-xs"></span>
                  Typing
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default ChatArea;
