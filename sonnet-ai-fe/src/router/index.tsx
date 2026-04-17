import { lazy, Suspense } from "react";
import {
  createBrowserRouter,
  RouterProvider,
  Navigate,
} from "react-router-dom";

const Login = lazy(() => import("../pages/login"));
const Register = lazy(() => import("../pages/register"));
const Chat = lazy(() => import("../pages/chat"));

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const token = localStorage.getItem("token");
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  return <>{children}</>;
};

const LoadingFallback = () => (
  <div className="bg-base-100 flex h-screen w-full items-center justify-center">
    <span className="loading loading-dots loading-md text-base-content/30"></span>
  </div>
);

const router = createBrowserRouter([
  {
    path: "/",
    element: <Navigate to="/chat" replace />,
  },
  {
    path: "/login",
    element: (
      <Suspense fallback={<LoadingFallback />}>
        <Login />
      </Suspense>
    ),
  },
  {
    path: "/register",
    element: (
      <Suspense fallback={<LoadingFallback />}>
        <Register />
      </Suspense>
    ),
  },
  {
    path: "/chat",
    element: (
      <ProtectedRoute>
        <Suspense fallback={<LoadingFallback />}>
          <Chat />
        </Suspense>
      </ProtectedRoute>
    ),
  },
]);

export default function AppRouter() {
  return <RouterProvider router={router} />;
}
