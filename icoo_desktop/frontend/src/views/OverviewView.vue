<template>
  <section class="page-section">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton variant="primary" :loading="store.refreshing" @click="reloadProxy">
          {{ store.refreshing ? "重载中..." : "重载代理" }}
        </UButton>
        <UTag :variant="store.data?.running ? 'success' : 'error'">
          {{ store.data?.running ? "运行中" : "已停止" }}
        </UTag>
      </div>
    </Teleport>

    <div v-if="store.loading" class="empty-state">
      正在加载网关概览...
    </div>

    <template v-else>
      <div class="content-panel overview-status">
        <div class="overview-status__primary">
          <p class="overview-status__label">服务状态</p>
          <p class="overview-status__value">
            <span class="app-status-dot" :class="store.data?.running ? 'app-status-dot--running' : 'app-status-dot--error'" aria-hidden="true"></span>
            {{ store.data?.running ? "网关运行中" : "网关未运行" }}
          </p>
          <p class="overview-status__hint">{{ store.data?.listen_addr || "尚未获取监听地址" }}</p>
        </div>
        <div class="overview-status__item">
          <p class="overview-status__label">访问模式</p>
          <p class="overview-status__value">{{ store.data?.auth_required ? `${store.data?.auth_key_count || 0} 个授权 Key` : "本地信任模式" }}</p>
          <p class="overview-status__hint">{{ store.data?.auth_required ? "外部访问需要凭据" : "按当前配置信任本地访问" }}</p>
        </div>
        <div class="overview-status__item">
          <p class="overview-status__label">供应商健康</p>
          <p class="overview-status__value">{{ store.reachableSupplierCount }} / {{ store.supplierCount }} 可用</p>
          <p class="overview-status__hint">{{ store.warningSupplierCount }} 个需要关注</p>
        </div>
        <div class="overview-status__item">
          <p class="overview-status__label">路由策略</p>
          <p class="overview-status__value">{{ store.activePolicyCount }} 条启用</p>
          <p class="overview-status__hint">{{ store.inactivePolicyCount }} 条停用</p>
        </div>
      </div>

      <div class="section-grid grid-cols-2 lg:grid-cols-4">
        <StatCard icon="wifi" label="监听地址" :value="store.data?.listen_addr || '-'" />
        <StatCard icon="server" label="供应商总数" :value="String(store.supplierCount)" tone="info" />
        <StatCard icon="check" label="健康可达" :value="String(store.reachableSupplierCount)" tone="success" />
        <StatCard icon="alert" label="需要关注" :value="String(store.warningSupplierCount)" :tone="store.warningSupplierCount ? 'danger' : 'neutral'" />
      </div>

      <div class="section-grid lg:grid-cols-2">
        <PanelBlock title="供应商健康汇总">
          <div class="overview-health-list">
            <div class="overview-health-row">
              <UTag variant="success" size="xs">可达</UTag>
              <div>
                <p class="overview-health-row__value">{{ store.reachableSupplierCount }}</p>
                <p class="overview-health-row__desc">最近检查结果正常的供应商数量</p>
              </div>
            </div>
            <div class="overview-health-row">
              <UTag variant="warning" size="xs">关注</UTag>
              <div>
                <p class="overview-health-row__value">{{ store.warningSupplierCount }}</p>
                <p class="overview-health-row__desc">返回 warning 或 unreachable 的供应商数量</p>
              </div>
            </div>
            <div class="overview-health-row">
              <UTag variant="info" size="xs">未检查</UTag>
              <div>
                <p class="overview-health-row__value">{{ Math.max(store.supplierCount - store.checkedSupplierCount, 0) }}</p>
                <p class="overview-health-row__desc">尚未执行健康检查的供应商数量</p>
              </div>
            </div>
          </div>
        </PanelBlock>

        <PanelBlock title="上游就绪状态">
          <div class="divide-y divide-[var(--ued-color-divider)]">
            <div v-for="upstream in store.data?.upstreams || []" :key="upstream.protocol" class="grid gap-2 py-2 grid-cols-[1fr_auto] items-center">
              <div class="min-w-0">
                <p class="text-[13px] font-medium text-strong">{{ upstream.protocol }}</p>
                <p class="mt-0.5 truncate text-xs text-muted">{{ upstream.base_url || "-" }}</p>
              </div>
              <UTag :variant="upstream.configured ? 'success' : 'warning'" size="xs">
                {{ upstream.configured ? "已配置" : "缺少密钥" }}
              </UTag>
            </div>
          </div>
        </PanelBlock>
      </div>

      <div class="section-grid lg:grid-cols-2">
        <PanelBlock title="运行检查">
          <div class="flex flex-wrap gap-1.5">
            <UTag
              v-for="(value, key) in store.checks"
              :key="key"
              :variant="value ? 'success' : 'warning'"
              size="xs"
            >
              {{ key }}: {{ value }}
            </UTag>
          </div>
        </PanelBlock>

        <PanelBlock title="支持的接口路径">
          <div class="flex flex-wrap gap-1.5">
            <UTag v-for="route in store.routes" :key="route" code size="xs">
              {{ route }}
            </UTag>
          </div>
        </PanelBlock>
      </div>
    </template>
  </section>
</template>

<script setup>
import { onMounted } from "vue";
import { useOverviewStore } from "../stores/overview";
import { useStoreError } from "../composables/useStoreError";

import PanelBlock from "../components/PanelBlock.vue";
import StatCard from "../components/StatCard.vue";
import UButton from "../components/ued/UButton.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";

const store = useOverviewStore();
useStoreError(store);

onMounted(() => {
  store.load();
});

async function reloadProxy() {
  await store.reloadProxy();
  if (!store.error) {
    message.success("代理已重载。");
  }
}
</script>
