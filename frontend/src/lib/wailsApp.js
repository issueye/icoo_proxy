function getAppBinding() {
  const binding = window?.go?.main?.App;
  if (!binding) {
    throw new Error("Wails runtime is not available in this session.");
  }
  return binding;
}

function invoke(method, ...args) {
  const app = getAppBinding();
  const fn = app[method];
  if (typeof fn !== "function") {
    throw new Error(`Wails method "${method}" is not available.`);
  }
  return fn(...args);
}

export function CheckSupplier(arg1) {
  return invoke("CheckSupplier", arg1);
}

export function ClearTrafficRequests() {
  return invoke("ClearTrafficRequests");
}

export function DeleteAuthKey(arg1) {
  return invoke("DeleteAuthKey", arg1);
}

export function DeleteEndpoint(arg1) {
  return invoke("DeleteEndpoint", arg1);
}

export function DeleteModelAlias(arg1) {
  return invoke("DeleteModelAlias", arg1);
}

export function DeleteSupplier(arg1) {
  return invoke("DeleteSupplier", arg1);
}

export function GetAuthKeySecret(arg1) {
  return invoke("GetAuthKeySecret", arg1);
}

export function GetAuthKeysPage(arg1, arg2, arg3, arg4) {
  return invoke("GetAuthKeysPage", arg1, arg2, arg3, arg4);
}

export function GetEndpointsPage(arg1, arg2, arg3, arg4) {
  return invoke("GetEndpointsPage", arg1, arg2, arg3, arg4);
}

export function GetOverview() {
  return invoke("GetOverview");
}

export function GetTrafficPage(arg1, arg2, arg3) {
  return invoke("GetTrafficPage", arg1, arg2, arg3);
}

export function GetProjectSettings() {
  return invoke("GetProjectSettings");
}

export function GetSuppliersPage(arg1, arg2, arg3, arg4) {
  return invoke("GetSuppliersPage", arg1, arg2, arg3, arg4);
}

export function GetUiPrefs() {
  return invoke("GetUiPrefs");
}

export function ListAuthKeys() {
  return invoke("ListAuthKeys");
}

export function ListEndpoints() {
  return invoke("ListEndpoints");
}

export function ListModelAliases() {
  return invoke("ListModelAliases");
}

export function ListRoutePolicies() {
  return invoke("ListRoutePolicies");
}

export function ListSupplierHealth() {
  return invoke("ListSupplierHealth");
}

export function ListSuppliers() {
  return invoke("ListSuppliers");
}

export function ReloadProxy() {
  return invoke("ReloadProxy");
}

export function SaveAuthKey(arg1) {
  return invoke("SaveAuthKey", arg1);
}

export function SaveEndpoint(arg1) {
  return invoke("SaveEndpoint", arg1);
}

export function SaveModelAlias(arg1) {
  return invoke("SaveModelAlias", arg1);
}

export function SaveProjectSettings(arg1) {
  return invoke("SaveProjectSettings", arg1);
}

export function SaveRoutePolicy(arg1) {
  return invoke("SaveRoutePolicy", arg1);
}

export function SaveSupplier(arg1) {
  return invoke("SaveSupplier", arg1);
}

export function SaveUiPrefs(arg1) {
  return invoke("SaveUiPrefs", arg1);
}

export function State() {
  return invoke("State");
}
