import { ref, computed, onMounted, onUnmounted } from "vue"
import { getServerConfig, saveServerConfig, wakeServer, serverStatus } from "../lib/apiClient"

const status = ref("disconnected")
const config = ref({ host: "127.0.0.1", port: 18181 })
const waking = ref(false)
const error = ref("")
let timer = null

const serverUrl = computed(() => `http://${config.value.host}:${config.value.port}`)

const statusText = computed(() => {
  switch (status.value) {
    case "connected": return "已连接"
    case "connecting": return "连接中"
    case "disconnected": return "未连接"
    case "error": return "连接异常"
    default: return "未知"
  }
})

const statusDotClass = computed(() => ({
  "app-status-dot--running": status.value === "connected",
  "app-status-dot--stopped": status.value === "disconnected",
  "app-status-dot--error": status.value === "error",
  "app-status-dot--connecting": status.value === "connecting",
}))

async function checkHealth() {
  try {
    const res = await fetch(`${serverUrl.value}/healthz`, {
      signal: AbortSignal.timeout(3000)
    })
    if (res.ok) {
      status.value = "connected"
      error.value = ""
      window.__ICOOSERVER_URL = serverUrl.value
      return true
    }
    status.value = "error"
    return false
  } catch (e) {
    if (status.value !== "connecting") {
      status.value = "disconnected"
    }
    return false
  }
}

async function wake() {
  waking.value = true
  status.value = "connecting"
  error.value = ""

  try {
    await wakeServer()
  } catch (e) {
    status.value = "error"
    error.value = e?.message || "服务启动失败"
    waking.value = false
    return false
  }

  for (let i = 0; i < 20; i++) {
    await new Promise(r => setTimeout(r, 1000))
    const ok = await checkHealth()
    if (ok) {
      waking.value = false
      return true
    }
  }

  status.value = "disconnected"
  error.value = "服务启动超时，请手动启动 icoo_llm_bridge"
  waking.value = false
  return false
}

async function setConfig(host, port) {
  config.value = { host, port }
  await saveServerConfig({ host, port })
}

function startPolling(interval = 10000) {
  stopPolling()
  checkHealth()
  timer = window.setInterval(checkHealth, interval)
}

function stopPolling() {
  if (timer) {
    window.clearInterval(timer)
    timer = null
  }
}

export function useServerConnection() {
  onMounted(async () => {
    try {
      config.value = await getServerConfig()
    } catch (e) {
      // use defaults
    }
    startPolling()
  })

  onUnmounted(() => stopPolling())

  return {
    status,
    statusText,
    statusDotClass,
    config,
    serverUrl,
    waking,
    error,
    wake,
    setConfig,
    checkHealth,
  }
}
