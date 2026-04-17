export interface MessageType {
  role: "user" | "assistant";
  content: string;
  meta?:
    | {
        status: string;
      }
    | undefined;
}

export interface SessionType {
  id: string;
  name: string;
  messages: {
    role: "user" | "assistant";
    content: string;
    meta?:
      | {
          status: string;
        }
      | undefined;
  }[];
}
