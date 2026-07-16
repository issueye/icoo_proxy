import { createRouter, createWebHashHistory } from "vue-router";

import OverviewView from "../views/OverviewView.vue";
import ChatView from "../views/ChatView.vue";
import AuthKeysView from "../views/AuthKeysView.vue";
import EndpointsView from "../views/EndpointsView.vue";
import RoutingRulesView from "../views/RoutingRulesView.vue";
import SuppliersView from "../views/SuppliersView.vue";
import SettingsView from "../views/SettingsView.vue";
import TrafficView from "../views/TrafficView.vue";
import UedSpecView from "../views/UedSpecView.vue";

import ModelAliasesView from "../views/ModelAliasesView.vue";
import ModelsView from "../views/ModelsView.vue";
import PluginExtensionView from "../views/PluginExtensionView.vue";
import PluginsView from "../views/PluginsView.vue";

export default createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: "/",
      name: "overview",
      component: OverviewView,
    },
    {
      path: "/chat",
      name: "chat",
      component: ChatView,
    },
    {
      path: "/suppliers",
      name: "suppliers",
      component: SuppliersView,
    },
    {
      path: "/routing-rules",
      name: "routing-rules",
      component: RoutingRulesView,
    },
    {
      path: "/models",
      name: "models",
      component: ModelsView,
    },
    {
      path: "/model-aliases",
      name: "model-aliases",
      component: ModelAliasesView,
    },
    {
      path: "/endpoints",
      name: "endpoints",
      component: EndpointsView,
    },
    {
      path: "/auth-keys",
      name: "auth-keys",
      component: AuthKeysView,
    },
    {
      path: "/traffic",
      name: "traffic",
      component: TrafficView,
    },
    {
      path: "/settings",
      name: "settings",
      component: SettingsView,
    },
    {
      path: "/plugins",
      name: "plugins",
      component: PluginsView,
    },
    {
      path: "/ued",
      name: "ued",
      component: UedSpecView,
    },
    {
      // Extension pages contributed by process plugins (iframe shell).
      path: "/ext/:pluginId/:pageId?",
      name: "plugin-extension",
      component: PluginExtensionView,
      props: true,
    },
  ],
});


