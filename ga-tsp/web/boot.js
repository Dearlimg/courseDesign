async function start() {
  const status = document.getElementById("authLoading");
  try {
    const response = await fetch("/api/auth/me", {
      credentials: "same-origin",
    });
    if (response.status === 401) {
      location.replace("/auth.html");
      return;
    }
    if (!response.ok) throw Error("登录验证服务暂不可用，请刷新重试。");
    const { user } = await response.json();
    document.documentElement.dataset.userId = user.id;
    document.getElementById("accountName").textContent = user.username;
    await import("./dispatch.js");
    await import("./selection.js");
    await import("./route.js");
    await import("./analysis.js");
    document.body.dataset.auth = "ready";
    status.hidden = true;
    document.getElementById("logout").onclick = async () => {
      const button = document.getElementById("logout");
      button.disabled = true;
      try {
        const response = await fetch("/api/auth/logout", {
          method: "POST",
          credentials: "same-origin",
        });
        if (!response.ok) throw Error("退出失败，请稍后重试。");
        location.replace("/auth.html");
      } catch (e) {
        document.getElementById("notice").textContent = e.message;
        button.disabled = false;
      }
    };
  } catch (error) {
    status.textContent = error.message || "工作台加载失败，请刷新重试。";
  }
}
start();
