import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import AppRouter from "./router";
import "./index.css";

const queryClient = new QueryClient();

const rootElement = document.getElementById("app");
if (rootElement) {
  const root = createRoot(rootElement);
  root.render(
    <QueryClientProvider client={queryClient}>
      <AppRouter />
    </QueryClientProvider>,
  );
}
