export function LoginPage() {
  function login() {
    window.location.href = "/auth/login";
  }

  return (
    <div className="flex items-center justify-center h-screen bg-gray-50">
      <div className="bg-white rounded-xl shadow p-8 flex flex-col items-center gap-4 w-80">
        <h1 className="text-xl font-semibold">LLM Gateway</h1>
        <p className="text-sm text-gray-500 text-center">企业内部 AI 聊天平台</p>
        <button
          onClick={login}
          className="w-full bg-blue-500 text-white py-2 rounded-lg text-sm hover:bg-blue-600"
        >
          企业账号登录
        </button>
      </div>
    </div>
  );
}
