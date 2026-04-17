import { useRef, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useToast } from "../hooks/use-toast";
import { useChat } from "../hooks/use-chat";
import { useFileUpload } from "../hooks/use-file-upload";
import Sidebar from "../components/sidebar";
import ChatArea from "../components/chat-area";
import MessageInput from "../components/message-input";

export default function Chat() {
  const navigate = useNavigate();
  const { toast, showToast } = useToast();
  const {
    sessionList,
    currentSessionId,
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
  } = useChat(showToast);

  const { uploading, fileInputRef, triggerFileUpload, handleFileUpload } =
    useFileUpload(showToast);

  const messagesRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    if (messagesRef.current) {
      messagesRef.current.scrollTop = messagesRef.current.scrollHeight;
    }
  };

  useEffect(() => {
    scrollToBottom();
  }, [currentMessages]);

  const handleLogout = () => {
    localStorage.removeItem("token");
    navigate("/login");
  };

  return (
    <div className="bg-base-100 text-base-content flex h-screen w-full font-sans">
      <Sidebar
        sessionList={sessionList}
        currentSessionId={currentSessionId}
        createNewSession={createNewSession}
        switchSession={switchSession}
        handleLogout={handleLogout}
      />

      {/* Main Chat Area */}
      <div className="relative flex h-full flex-1 flex-col">
        {/* Top Bar for Config (Mobile toggle, Model Select, etc) */}
        <div className="absolute top-0 right-0 z-10 flex items-center justify-end gap-2 p-3">
          <select
            className="select select-sm select-ghost bg-base-200/50 focus:bg-base-200 rounded-full focus:outline-none"
            value={selectedModel}
            onChange={(e) => setSelectedModel(e.target.value)}
          >
            <option value="ollama">Ollama</option>
            <option value="ollama-rag">Ollama + RAG</option>
            <option value="ollama-mcp">Ollama + MCP</option>
          </select>
          <label className="bg-base-200/50 label cursor-pointer gap-2 rounded-full px-3 py-1">
            <span className="label-text text-xs font-medium">Stream</span>
            <input
              type="checkbox"
              className="toggle toggle-xs toggle-primary"
              checked={isStreaming}
              onChange={(e) => setIsStreaming(e.target.checked)}
            />
          </label>
        </div>

        <ChatArea currentMessages={currentMessages} messagesRef={messagesRef} />

        <MessageInput
          inputMessage={inputMessage}
          setInputMessage={setInputMessage}
          loading={loading}
          uploading={uploading}
          sendMessage={sendMessage}
          triggerFileUpload={triggerFileUpload}
          handleFileUpload={handleFileUpload}
          fileInputRef={fileInputRef}
          currentMessages={currentMessages}
          messagesRef={messagesRef}
        />
      </div>

      {/* Toast Container */}
      {toast && (
        <div className="toast toast-top toast-center z-50">
          <div
            className={`alert ${
              toast.type === "success"
                ? "alert-success text-success-content"
                : toast.type === "error"
                  ? "alert-error text-error-content"
                  : toast.type === "warning"
                    ? "alert-warning text-warning-content"
                    : "alert-info text-info-content"
            } rounded-xl px-4 py-2 shadow-lg`}
          >
            <span>{toast.text}</span>
          </div>
        </div>
      )}
    </div>
  );
}
