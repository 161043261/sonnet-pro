import { MessageSquarePlus, ChevronDown, User } from "lucide-react";
import { SessionType } from "../types";

interface SidebarProps {
  sessionList: SessionType[];
  currentSessionId: string | null;
  createNewSession: () => void;
  switchSession: (id: string) => void;
  handleLogout: () => void;
}

function Sidebar({
  sessionList,
  currentSessionId,
  createNewSession,
  switchSession,
  handleLogout,
}: SidebarProps) {
  return (
    <div className="bg-base-200/50 border-base-300 flex w-64 shrink-0 flex-col border-r">
      <div className="flex flex-col gap-4 p-4">
        <h1 className="px-2 text-lg font-semibold">lark AI</h1>
        <button
          className="btn btn-ghost w-full justify-start font-normal"
          onClick={createNewSession}
        >
          <MessageSquarePlus size={18} className="mr-2 opacity-70" />
          New chat
        </button>
      </div>

      <div className="mt-2 flex-1 overflow-y-auto px-4">
        {sessionList.length === 0 ? (
          <div className="text-base-content/50 px-2 text-sm italic">
            Your chats will show up here
          </div>
        ) : (
          <ul className="menu menu-sm px-0">
            {sessionList.map((session) => (
              <li key={session.id}>
                <a
                  className={`truncate rounded-lg ${currentSessionId === session.id ? "active bg-base-300/50 text-base-content" : "text-base-content/70 hover:bg-base-300/30"}`}
                  onClick={() => switchSession(session.id)}
                >
                  {session.name || `Session ${session.id}`}
                </a>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* User Profile Area */}
      <div className="border-base-300 mt-auto border-t p-4">
        <div className="dropdown dropdown-top w-full">
          <div
            tabIndex={0}
            role="button"
            className="btn btn-ghost w-full flex-nowrap justify-between px-2"
          >
            <div className="flex items-center gap-2 truncate">
              <div className="bg-base-300 text-base-content flex h-8 w-8 items-center justify-center rounded-full">
                <User size={16} />
              </div>
              <div className="flex flex-col items-start truncate">
                <span className="text-sm leading-tight font-medium">User</span>
              </div>
            </div>
            <ChevronDown size={14} className="shrink-0 opacity-50" />
          </div>
          <ul
            tabIndex={0}
            className="dropdown-content menu bg-base-100 rounded-box border-base-200 z-[1] mb-2 w-full border p-2 shadow-sm"
          >
            <li>
              <a onClick={handleLogout} className="text-error">
                Sign out
              </a>
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
}

export default Sidebar;
