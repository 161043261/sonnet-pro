import { useState } from "react";

export type ToastType = "info" | "success" | "warning" | "error";

export function useToast() {
  const [toast, setToast] = useState<{ text: string; type: ToastType } | null>(
    null,
  );

  const showToast = (text: string, type: ToastType = "info") => {
    setToast({ text, type });
    setTimeout(() => setToast(null), 3000);
  };

  return { toast, showToast };
}
