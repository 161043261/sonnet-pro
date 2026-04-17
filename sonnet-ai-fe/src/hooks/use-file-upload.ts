import { useState, useRef } from "react";
import api from "../api";
import { ToastType } from "./use-toast";

export function useFileUpload(
  showToast: (text: string, type: ToastType) => void,
) {
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const triggerFileUpload = () => {
    if (fileInputRef.current) {
      fileInputRef.current.click();
    }
  };

  const handleFileUpload = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const fileName = file.name.toLowerCase();
    if (!fileName.endsWith(".md") && !fileName.endsWith(".txt")) {
      showToast("Only .md or .txt files are allowed", "error");
      if (fileInputRef.current) fileInputRef.current.value = "";
      return;
    }

    try {
      setUploading(true);
      const formData = new FormData();
      formData.append("file", file);
      const response = await api.post("/file/upload", formData, {
        headers: { "Content-Type": "multipart/form-data" },
      });
      if (response.data && response.data.status_code === 1000) {
        showToast("File uploaded successfully", "success");
      } else {
        showToast(response.data?.status_msg || "Upload failed", "error");
      }
    } catch (error: unknown) {
      console.error("Upload error:", error);
      showToast("File upload failed", "error");
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  return { uploading, fileInputRef, triggerFileUpload, handleFileUpload };
}
