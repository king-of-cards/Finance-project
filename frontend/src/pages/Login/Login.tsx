import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { User, Lock, Eye, EyeOff, BarChart3 } from "lucide-react";
import { loginUser } from "../../api/authApi";

function Login() {
  const [userId, setUserId] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const result = await loginUser(userId, password);

      // Persist the token + user so future requests can authenticate
      // and so the session survives a page refresh.
      localStorage.setItem("token", result.token);
      localStorage.setItem("user", JSON.stringify(result.user));

      navigate("/dashboard");
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("Something went wrong");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex">
      {/* Left panel */}
      <div className="hidden md:flex w-[40%] bg-black flex-col justify-between p-10 relative overflow-hidden">
        <div className="flex items-center gap-3 z-10">
          <div className="w-10 h-10 border border-white/40 rounded-lg flex items-center justify-center">
            <BarChart3 className="text-white" size={20} />
          </div>
          <div>
            <p className="text-white font-bold text-lg leading-tight">FINANCE</p>
            <p className="text-white/60 text-xs tracking-widest">DASHBOARD</p>
          </div>
        </div>

        {/* Chart illustration */}
        <svg viewBox="0 0 300 200" className="absolute bottom-0 left-0 w-full h-2/3 opacity-90">
          <rect x="10" y="140" width="20" height="60" fill="#333" />
          <rect x="45" y="120" width="20" height="80" fill="#3a3a3a" />
          <rect x="80" y="150" width="20" height="50" fill="#333" />
          <rect x="115" y="100" width="20" height="100" fill="#3a3a3a" />
          <rect x="150" y="90" width="20" height="110" fill="#333" />
          <rect x="185" y="60" width="20" height="140" fill="#3a3a3a" />
          <rect x="220" y="40" width="20" height="160" fill="#333" />
          <rect x="255" y="20" width="20" height="180" fill="#3a3a3a" />

          <polyline
            points="20,150 55,130 90,160 125,110 160,100 195,70 230,50 265,30"
            fill="none"
            stroke="white"
            strokeWidth="2"
          />
          {[
            [20, 150], [55, 130], [90, 160], [125, 110],
            [160, 100], [195, 70], [230, 50], [265, 30],
          ].map(([cx, cy], i) => (
            <circle key={i} cx={cx} cy={cy} r="3" fill="white" />
          ))}
        </svg>
      </div>

      {/* Right panel */}
      <div className="flex-1 flex items-center justify-center bg-white p-6">
        <div className="w-full max-w-sm bg-white rounded-2xl border border-gray-100 shadow-lg p-8">
          <h1 className="text-black text-2xl font-bold text-center">Welcome Back</h1>
          <p className="text-gray-500 text-sm text-center mt-1 mb-6">
            Sign in to access your finance dashboard
          </p>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex items-center gap-2 border border-gray-200 rounded-lg px-3 py-2 focus-within:border-black">
              <User size={18} className="text-gray-400" />
              <input
                type="text"
                placeholder="User ID"
                value={userId}
                onChange={(e) => setUserId(e.target.value)}
                className="flex-1 outline-none text-black bg-transparent"
                required
              />
            </div>

            <div className="flex items-center gap-2 border border-gray-200 rounded-lg px-3 py-2 focus-within:border-black">
              <Lock size={18} className="text-gray-400" />
              <input
                type={showPassword ? "text" : "password"}
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="flex-1 outline-none text-black bg-transparent"
                required
              />
              <button
                type="button"
                onClick={() => setShowPassword((prev) => !prev)}
                className="text-gray-400"
                aria-label={showPassword ? "Hide password" : "Show password"}
              >
                {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
              </button>
            </div>

            {error && <p className="text-red-600 text-sm">{error}</p>}

            <button
              type="submit"
              disabled={loading}
              className="bg-black text-white py-2 rounded-lg mt-2 disabled:opacity-50"
            >
              {loading ? "Logging in..." : "Login"}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}

export default Login;