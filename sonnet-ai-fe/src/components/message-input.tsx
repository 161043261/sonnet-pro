import React from "react";
import { Paperclip, Send, Code, BookOpen, PenTool } from "lucide-react";
import { MessageType } from "../types";

interface MessageInputProps {
  inputMessage: string;
  setInputMessage: (msg: string) => void;
  loading: boolean;
  uploading: boolean;
  sendMessage: (e?: React.KeyboardEvent | React.MouseEvent) => void;
  triggerFileUpload: () => void;
  handleFileUpload: (e: React.ChangeEvent<HTMLInputElement>) => void;
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  currentMessages: MessageType[];
  messagesRef: React.RefObject<HTMLDivElement | null>;
}

function MessageInput({
  inputMessage,
  setInputMessage,
  loading,
  uploading,
  sendMessage,
  triggerFileUpload,
  handleFileUpload,
  fileInputRef,
  currentMessages,
  messagesRef,
}: MessageInputProps) {
  return (
    <div
      className={`from-base-100 via-base-100 absolute bottom-0 left-0 w-full bg-gradient-to-t to-transparent px-4 pt-10 pb-6 md:px-12 ${
        currentMessages.length === 0
          ? "top-1/2 bottom-auto -translate-y-1/2 bg-none pt-0"
          : ""
      }`}
    >
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-3">
        <div className="border-base-300 bg-base-100 focus-within:border-base-400 relative flex flex-col overflow-hidden rounded-3xl border shadow-sm transition-all focus-within:shadow-md">
          <div className="flex items-center gap-2 px-2 py-1">
            <button
              className="btn btn-circle btn-sm btn-ghost text-base-content/50 hover:text-base-content hover:bg-base-200"
              onClick={triggerFileUpload}
              disabled={uploading || loading}
              title="Attach file"
            >
              <Paperclip size={18} />
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept=".md,.txt,text/markdown,text/plain"
              className="hidden"
              onChange={handleFileUpload}
            />
            <textarea
              value={inputMessage}
              onChange={(e) => setInputMessage(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  sendMessage(e);
                }
              }}
              placeholder="Type / for skills"
              disabled={loading}
              rows={1}
              className="text-base-content placeholder-base-content/40 flex max-h-48 min-h-[40px] w-full resize-none items-center border-none bg-transparent px-2 py-2 text-[15px] leading-normal outline-none"
              style={{ fieldSizing: "content" } as React.CSSProperties}
            />

            <div className="flex shrink-0 items-center gap-2">
              <div className="text-base-content/50 bg-base-200/50 hidden rounded-md px-2 py-1 text-xs font-medium sm:block">
                lark
              </div>
              <button
                type="button"
                disabled={!inputMessage.trim() || loading}
                onClick={sendMessage}
                className="btn btn-circle btn-sm bg-base-content text-base-100 hover:bg-base-content/80 disabled:bg-base-300 disabled:text-base-content/30 border-none transition-colors"
              >
                <Send size={16} className="ml-0.5" />
              </button>
            </div>
          </div>
        </div>

        {/* Empty State Suggestions */}
        {currentMessages.length === 0 && (
          <div className="mt-2 flex flex-wrap justify-center gap-2">
            <button
              className="btn btn-sm btn-outline border-base-300 text-base-content/70 hover:bg-base-200 hover:border-base-300 hover:text-base-content bg-base-100 rounded-full font-normal"
              onClick={() => {
                setInputMessage("Write a python script that...");
                messagesRef.current?.focus();
              }}
            >
              <Code size={14} className="mr-1" /> Code
            </button>
            <button
              className="btn btn-sm btn-outline border-base-300 text-base-content/70 hover:bg-base-200 hover:border-base-300 hover:text-base-content bg-base-100 rounded-full font-normal"
              onClick={() => {
                setInputMessage("Help me understand...");
                messagesRef.current?.focus();
              }}
            >
              <BookOpen size={14} className="mr-1" /> Learn
            </button>
            <button
              className="btn btn-sm btn-outline border-base-300 text-base-content/70 hover:bg-base-200 hover:border-base-300 hover:text-base-content bg-base-100 rounded-full font-normal"
              onClick={() => {
                setInputMessage("Write an email to...");
                messagesRef.current?.focus();
              }}
            >
              <PenTool size={14} className="mr-1" /> Write
            </button>
          </div>
        )}

        {currentMessages.length > 0 && (
          <div className="text-base-content/40 text-center text-xs">
            AI can make mistakes. Please verify important information.
          </div>
        )}
      </div>
    </div>
  );
}

export default MessageInput;
