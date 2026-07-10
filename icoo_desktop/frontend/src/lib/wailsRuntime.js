function callRuntime(method) {
  if (typeof window === "undefined") {
    return;
  }
  const runtimeMethod = window.runtime?.[method];
  if (typeof runtimeMethod === "function") {
    return runtimeMethod();
  }
}

export function WindowHide() {
  return callRuntime("WindowHide");
}

export function WindowMinimise() {
  return callRuntime("WindowMinimise");
}
