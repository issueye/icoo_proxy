import { checkHealth } from "./common-api";
import * as sessionApi from "./sessions-api";
import * as messageApi from "./messages-api";
import * as providerApi from "./providers-api";
import * as memoryApi from "./memories-api";
import * as configApi from "./config-api";
import * as paramsApi from "./params-api";
import * as chatApi from "./chat-api";
import * as workspaceApi from "./workspace-api";
import {
  getApiBaseUrl,
  setApiBaseUrl,
  request,
} from "./http";

export * from "./http";
export * from "./common-api";
export * from "./sessions-api";
export * from "./messages-api";
export * from "./providers-api";
export * from "./memories-api";
export * from "./config-api";
export * from "./params-api";
export * from "./chat-api";
export * from "./workspace-api";

export default {
  getApiBaseUrl,
  setApiBaseUrl,
  checkHealth,
  request,
  ...sessionApi,
  ...messageApi,
  ...providerApi,
  ...memoryApi,
  ...configApi,
  ...paramsApi,
  ...chatApi,
  ...workspaceApi,
};
