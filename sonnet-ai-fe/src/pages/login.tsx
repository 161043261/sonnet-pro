import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";
import { User, Lock, Eye, EyeOff, Loader2 } from "lucide-react";
import api from "../api";

const LoginRequestSchema = z.object({
  username: z.string().min(1, "Username is required"),
  password: z.string().min(6, "Password must be at least 6 characters"),
});
type LoginRequest = z.infer<typeof LoginRequestSchema>;

export default function Login() {
  const navigate = useNavigate();
  const [showPassword, setShowPassword] = useState(false);
  const [formData, setFormData] = useState<LoginRequest>({
    username: "yukinohangtiancheng@gmail.com",
    password: "Shita0228",
  });
  const [errors, setErrors] = useState<{
    username?: string;
    password?: string;
  }>({});
  const [toast, setToast] = useState<{
    text: string;
    type: "success" | "error";
  } | null>(null);

  const showToast = (text: string, type: "success" | "error") => {
    setToast({ text, type });
    setTimeout(() => setToast(null), 3000);
  };

  const loginMutation = useMutation({
    mutationFn: async (values: LoginRequest) => {
      const response = await api.post("/user/login", values);
      return response.data;
    },
    onSuccess: (data) => {
      if (data.status_code === 1000) {
        localStorage.setItem("token", data.token);
        showToast("Login Successful", "success");
        setTimeout(() => navigate("/chat"), 1000);
      } else {
        showToast(data.status_msg || "Login Failed", "error");
      }
    },
    onError: (error: unknown) => {
      console.error("Login error:", error);
      showToast("Login Failed, Please try again", "error");
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const parsed = LoginRequestSchema.safeParse(formData);
    if (parsed.success) {
      setErrors({});
      loginMutation.mutate(parsed.data);
    } else {
      const fieldErrors: Record<string, string> = {};
      parsed.error.issues.forEach((err) => {
        if (err.path[0]) fieldErrors[String(err.path[0])] = err.message;
      });
      setErrors(fieldErrors);
    }
  };

  return (
    <div className="bg-base-200/30 text-base-content flex min-h-screen items-center justify-center p-4">
      <div className="border-base-300 bg-base-100 w-full max-w-100 rounded-3xl border p-8 shadow-sm">
        <div className="mb-10 text-center">
          <div className="bg-primary/10 text-primary mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-2xl">
            <Lock size={24} />
          </div>
          <h2 className="text-2xl font-semibold tracking-tight">
            Welcome Back
          </h2>
          <p className="text-base-content/60 mt-2 text-sm font-normal">
            Please sign in to continue
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div className="form-control w-full">
            <label className="floating-label w-full">
              <input
                type="text"
                placeholder="Username"
                className={`input input-md bg-base-200/50 focus:border-primary/30 focus:bg-base-100 w-full rounded-2xl border-transparent transition-all ${errors.username ? "border-error/50" : ""}`}
                value={formData.username}
                onChange={(e) =>
                  setFormData({ ...formData, username: e.target.value })
                }
              />
              <span className="flex items-center gap-2">
                <User size={14} className="opacity-50" /> Username
              </span>
            </label>
            {errors.username && (
              <label className="label px-2 py-1">
                <span className="label-text-alt text-error">
                  {errors.username}
                </span>
              </label>
            )}
          </div>

          <div className="form-control w-full">
            <label className="floating-label w-full">
              <input
                type={showPassword ? "text" : "password"}
                placeholder="Password"
                className={`input input-md bg-base-200/50 focus:border-primary/30 focus:bg-base-100 w-full rounded-2xl border-transparent pr-10 transition-all ${errors.password ? "border-error/50" : ""}`}
                value={formData.password}
                onChange={(e) =>
                  setFormData({ ...formData, password: e.target.value })
                }
              />
              <span className="flex items-center gap-2">
                <Lock size={14} className="opacity-50" /> Password
              </span>
              <button
                type="button"
                className="text-base-content/40 hover:text-base-content absolute top-1/2 right-3 -translate-y-1/2 transition-colors"
                onClick={() => setShowPassword(!showPassword)}
              >
                {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </label>
            {errors.password && (
              <label className="label px-2 py-1">
                <span className="label-text-alt text-error">
                  {errors.password}
                </span>
              </label>
            )}
          </div>

          <div className="pt-2">
            <button
              type="submit"
              disabled={loginMutation.isPending}
              className="btn btn-primary btn-md text-primary-content h-12 w-full rounded-2xl text-base font-medium shadow-sm transition-all hover:shadow-md"
            >
              {loginMutation.isPending ? (
                <Loader2 size={18} className="animate-spin" />
              ) : (
                "Sign In"
              )}
            </button>
          </div>
        </form>

        <div className="mt-8 text-center">
          <p className="text-base-content/60 text-sm">
            Don't have an account?{" "}
            <Link
              to="/register"
              className="link link-primary font-medium no-underline hover:underline"
            >
              Create Account
            </Link>
          </p>
        </div>
      </div>

      {/* Toast Container */}
      {toast && (
        <div className="toast toast-top toast-center z-50">
          <div
            className={`alert ${toast.type === "success" ? "alert-success text-success-content" : "alert-error text-error-content"} rounded-xl px-4 py-2 shadow-lg`}
          >
            <span>{toast.text}</span>
          </div>
        </div>
      )}
    </div>
  );
}
