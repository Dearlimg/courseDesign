const $ = (id) => document.getElementById(id);
let mode = "login",
  busy = false;
function switchMode(next) {
  if (busy) return;
  mode = next;
  const register = mode === "register";
  $("loginTab").setAttribute("aria-selected", String(!register));
  $("registerTab").setAttribute("aria-selected", String(register));
  $("authTitle").textContent = register ? "开启你的调度工作台" : "欢迎回来";
  $("authSubtitle").textContent = register
    ? "创建账号，把下一趟配送安排好。"
    : "登录账号，开始今天的配送规划。";
  $("confirmField").hidden = !register;
  $("confirmPassword").required = register;
  $("password").autocomplete = register ? "new-password" : "current-password";
  $("authSubmit").textContent = register ? "创建账号 →" : "登录工作台 →";
  $("authMessage").textContent = "";
  $("authMessage").className = "";
}
$("loginTab").onclick = () => switchMode("login");
$("registerTab").onclick = () => switchMode("register");
$("showPassword").onclick = () => {
  const show = $("password").type === "password";
  $("password").type = show ? "text" : "password";
  $("showPassword").textContent = show ? "隐藏" : "显示";
  $("showPassword").setAttribute("aria-label", show ? "隐藏密码" : "显示密码");
};
$("authForm").onsubmit = async (e) => {
  e.preventDefault();
  if (busy) return;
  const password = $("password").value;
  if (new TextEncoder().encode(password).length > 72) {
    $("authMessage").textContent = "密码 UTF-8 编码不能超过 72 字节";
    return;
  }
  if (mode === "register" && password !== $("confirmPassword").value) {
    $("authMessage").textContent = "两次输入的密码不一致";
    return;
  }
  busy = true;
  $("authSubmit").disabled = true;
  $("authMessage").className = "";
  $("authMessage").textContent =
    mode === "login" ? "正在登录…" : "正在创建账号…";
  try {
    const response = await fetch("/api/auth/" + mode, {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username: $("username").value.trim(), password }),
    });
    const result = await response.json();
    if (!response.ok) throw Error(result.error || "操作失败，请重试");
    if (mode === "login") {
      location.replace("/");
      return;
    }
    busy = false;
    switchMode("login");
    $("password").value = "";
    $("confirmPassword").value = "";
    $("authMessage").textContent = "注册成功，请使用新账号登录。";
    $("authMessage").className = "success";
    $("password").focus();
  } catch (error) {
    $("authMessage").textContent =
      error.message === "Failed to fetch"
        ? "无法连接服务，请稍后重试。"
        : error.message;
  } finally {
    busy = false;
    $("authSubmit").disabled = false;
  }
};
fetch("/api/auth/me", { credentials: "same-origin" })
  .then((r) => {
    if (r.ok) location.replace("/");
  })
  .catch(() => {});
