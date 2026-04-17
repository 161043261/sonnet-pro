import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";
import { Mail, Lock, Eye, EyeOff, Loader2 } from "lucide-react";
import api from "../api";

const RegisterRequestSchema = z
  .object({
    email: z.email("Please input a valid email address"),
    password: z.string().min(6, "Password must be at least 6 characters"),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords don't match",
    path: ["confirmPassword"],
  });

type RegisterRequest = z.infer<typeof RegisterRequestSchema>;

export default function Register() {
  const navigate = useNavigate();
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [formData, setFormData] = useState({
    email: "yukinohangtiancheng@gmail.com",
    password: "Shita0228",
    confirmPassword: "Shita0228",
  });
  const [errors, setErrors] = useState<{
    email?: string;
    password?: string;
    confirmPassword?: string;
  }>({});
  const [toast, setToast] = useState<{
    text: string;
    type: "success" | "error";
  } | null>(null);

  const showToast = (text: string, type: "success" | "error") => {
    setToast({ text, type });
    setTimeout(() => setToast(null), 3000);
  };

  const registerMutation = useMutation({
    mutationFn: async (values: Omit<RegisterRequest, "confirmPassword">) => {
      const response = await api.post("/user/register", {
        email: values.email,
        password: values.password,
      });
      return response.data;
    },
    onSuccess: (data) => {
      if (data.status_code === 1000) {
        showToast("Registration successful, please sign in", "success");
        setTimeout(() => navigate("/login"), 1000);
      } else {
        showToast(data.status_msg || "Registration failed", "error");
      }
    },
    onError: (error: unknown) => {
      console.error("Register error:", error);
      showToast("Registration failed, please try again", "error");
    },
  });

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault();
    const parsed = RegisterRequestSchema.safeParse(formData);
    if (parsed.success) {
      setErrors({});
      registerMutation.mutate({
        email: parsed.data.email,
        password: parsed.data.password,
      });
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
            <Mail size={24} />
          </div>
          <h2 className="text-2xl font-semibold tracking-tight">
            Create an Account
          </h2>
          <p className="text-base-content/60 mt-2 text-sm font-normal">
            Sign up to experience smart AI services
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div className="form-control w-full">
            <label className="floating-label w-full">
              <input
                type="email"
                placeholder="Email Address"
                className={`input input-md bg-base-200/50 focus:border-primary/30 focus:bg-base-100 w-full rounded-2xl border-transparent transition-all ${errors.email ? "border-error/50" : ""}`}
                value={formData.email}
                onChange={(e) =>
                  setFormData({ ...formData, email: e.target.value })
                }
              />
              <span className="flex items-center gap-2">
                <Mail size={14} className="opacity-50" /> Email Address
              </span>
            </label>
            {errors.email && (
              <label className="label px-2 py-1">
                <span className="label-text-alt text-error">
                  {errors.email}
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

          <div className="form-control w-full">
            <label className="floating-label w-full">
              <input
                type={showConfirmPassword ? "text" : "password"}
                placeholder="Confirm Password"
                className={`input input-md bg-base-200/50 focus:border-primary/30 focus:bg-base-100 w-full rounded-2xl border-transparent pr-10 transition-all ${errors.confirmPassword ? "border-error/50" : ""}`}
                value={formData.confirmPassword}
                onChange={(e) =>
                  setFormData({ ...formData, confirmPassword: e.target.value })
                }
              />
              <span className="flex items-center gap-2">
                <Lock size={14} className="opacity-50" /> Confirm Password
              </span>
              <button
                type="button"
                className="text-base-content/40 hover:text-base-content absolute top-1/2 right-3 -translate-y-1/2 transition-colors"
                onClick={() => setShowConfirmPassword(!showConfirmPassword)}
              >
                {showConfirmPassword ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </label>
            {errors.confirmPassword && (
              <label className="label px-2 py-1">
                <span className="label-text-alt text-error">
                  {errors.confirmPassword}
                </span>
              </label>
            )}
          </div>

          <div className="pt-2">
            <button
              type="submit"
              disabled={registerMutation.isPending}
              className="btn btn-primary btn-md text-primary-content h-12 w-full rounded-2xl text-base font-medium shadow-sm transition-all hover:shadow-md"
            >
              {registerMutation.isPending ? (
                <Loader2 size={18} className="animate-spin" />
              ) : (
                "Create Account"
              )}
            </button>
          </div>
        </form>

        <div className="mt-8 text-center">
          <p className="text-base-content/60 text-sm">
            Already have an account?{" "}
            <Link
              to="/login"
              className="link link-primary font-medium no-underline hover:underline"
            >
              Sign In
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
