<template>
  <section class="page-section">
    <div class="page-header">
      <h2 class="page-title">UED 组件规范</h2>
      <p class="page-description">
        面向本地代理控制台的组件样例，覆盖按钮、表单、状态反馈、表格和弹窗。
      </p>
    </div>

    <PanelBlock title="设计 Token">
      <div class="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
        <div class="sub-card">
          <p class="text-sm font-medium text-strong">主色</p>
          <div class="mt-3 flex items-center gap-2">
            <span class="h-6 w-10 rounded bg-[var(--ued-color-primary)]"></span>
            <span class="font-mono text-xs text-muted">var(--ued-color-primary)</span>
          </div>
        </div>
        <div class="sub-card">
          <p class="text-sm font-medium text-strong">圆角</p>
          <p class="mt-3 text-sm text-muted">控件 6px，面板 8px，标签 4px。</p>
        </div>
        <div class="sub-card">
          <p class="text-sm font-medium text-strong">控件高度</p>
          <p class="mt-3 text-sm text-muted">XS 24 / SM 28 / MD 32 / LG 40。</p>
        </div>
        <div class="sub-card">
          <p class="text-sm font-medium text-strong">状态色</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <UTag variant="success" dot>Success</UTag>
            <UTag variant="warning" dot>Warning</UTag>
            <UTag variant="error" dot>Error</UTag>
            <UTag variant="info" dot>Info</UTag>
          </div>
        </div>
      </div>
    </PanelBlock>

    <div class="section-grid lg:grid-cols-2">
      <PanelBlock title="按钮">
        <div class="space-y-3">
          <div class="flex flex-wrap gap-2">
            <UButton variant="primary">Primary</UButton>
            <UButton variant="success">Success</UButton>
            <UButton variant="warning">Warning</UButton>
            <UButton variant="error">Error</UButton>
            <UButton variant="info">Info</UButton>
            <UButton variant="secondary">Secondary</UButton>
            <UButton variant="ghost">Ghost</UButton>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <UButton size="xs">XS</UButton>
            <UButton size="sm">SM</UButton>
            <UButton size="md">MD</UButton>
            <UButton size="lg">LG</UButton>
            <UButton loading>加载中</UButton>
          </div>
        </div>
      </PanelBlock>

      <PanelBlock title="开关">
        <div class="space-y-3">
          <USwitch v-model="switchValue" label="启用自动健康检查" hint="用于开关型参数。" />
          <USwitch :model-value="true" label="保留系统默认路由" hint="禁用态示例" disabled />
        </div>
      </PanelBlock>
    </div>

    <div class="section-grid lg:grid-cols-2">
      <PanelBlock title="提示">
        <div class="space-y-3">
          <UAlert type="success" message="保存成功，配置已写入本地代理。" />
          <UAlert type="info" message="提示信息" description="用于承载页面内常驻说明或轻量引导。" />
          <UAlert type="warning" message="代理尚未重载" description="保存端点后需要重载代理。" closable />
          <UAlert type="error" message="连接失败" description="请检查供应商地址、鉴权 Key 和网络连通性。" />
        </div>
      </PanelBlock>

      <PanelBlock title="消息">
        <div class="flex flex-wrap gap-2">
          <UButton variant="success" @click="showMessage('success')">Success</UButton>
          <UButton variant="info" @click="showMessage('info')">Info</UButton>
          <UButton variant="warning" @click="showMessage('warning')">Warning</UButton>
          <UButton variant="error" @click="showMessage('error')">Error</UButton>
          <UButton variant="secondary" @click="showLoadingMessage">Loading</UButton>
        </div>
      </PanelBlock>
    </div>

    <div class="section-grid lg:grid-cols-2">
      <PanelBlock title="加载">
        <div class="space-y-4">
          <div class="flex flex-wrap items-center gap-5">
            <ULoading size="sm" />
            <ULoading />
            <ULoading size="lg" tip="加载中" />
          </div>
          <ULoading tip="正在加载端点数据..." :spinning="true">
            <div class="rounded-md border border-[var(--ued-color-divider)] bg-[var(--ued-color-muted)] p-4">
              <p class="text-sm font-medium text-strong">代理端点</p>
              <p class="mt-2 text-sm leading-6 text-muted">
                区域加载用于表格、详情面板或配置块刷新。
              </p>
            </div>
          </ULoading>
          <UButton variant="secondary" @click="showFullscreenLoading">全屏 Loading</UButton>
        </div>
      </PanelBlock>

      <PanelBlock title="标签">
        <div class="space-y-3">
          <div class="flex flex-wrap gap-2">
            <UTag variant="primary" dot>primary</UTag>
            <UTag variant="success" dot>success</UTag>
            <UTag variant="warning" dot>warning</UTag>
            <UTag variant="error" dot>error</UTag>
            <UTag variant="info" dot>info</UTag>
            <UTag>neutral</UTag>
            <UTag code>openai-responses</UTag>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <UTag size="xs" variant="primary">xs</UTag>
            <UTag size="sm" variant="success">sm</UTag>
            <UTag size="md" variant="warning">md</UTag>
            <UTag size="lg" variant="error">lg</UTag>
            <UTag size="sm" code>/v1/responses</UTag>
          </div>
        </div>
      </PanelBlock>
    </div>

    <div class="section-grid lg:grid-cols-2">
      <PanelBlock title="输入与下拉">
        <div class="space-y-3">
          <UInput v-model="form.name" label="名称" placeholder="请输入供应商名称" hint="表单项采用上 label、下控件布局。" required />
          <USelect v-model="form.protocol" label="协议" :options="protocolOptions" required />
          <UInput v-model="form.description" label="描述" placeholder="请输入用途说明" textarea />
        </div>
      </PanelBlock>

      <PanelBlock title="弹窗">
        <div class="flex flex-wrap gap-2">
          <UButton @click="showModal = true">普通弹窗</UButton>
          <UButton variant="error" @click="showConfirm = true">确认弹窗</UButton>
        </div>
      </PanelBlock>
    </div>

    <PanelBlock title="基础表格">
      <UTable
        :columns="columns"
        :rows="filteredRows"
        row-key="id"
        empty-text="暂无组件示例数据。"
        action-width="72px"
        pagination
        :page-size="3"
      >
        <template #query>
          <div class="table-query-form">
            <UInput
              v-model="basicTableQuery.keyword"
              label="关键词"
              hide-label
              placeholder="搜索名称"
              class="table-query-form__field table-query-form__field--keyword"
            />
            <USelect
              v-model="basicTableQuery.type"
              label="类型"
              hide-label
              :options="basicTypeOptions"
              class="table-query-form__field table-query-form__field--select"
            />
            <USelect
              v-model="basicTableQuery.status"
              label="状态"
              hide-label
              :options="basicStatusOptions"
              class="table-query-form__field table-query-form__field--select"
            />
            <div class="table-query-form__actions">
              <UButton variant="secondary" @click="resetBasicTableQuery">重置</UButton>
            </div>
          </div>
        </template>
        <template #cell-status="{ value }">
          <UTag :variant="value === '启用' ? 'success' : 'error'" dot>{{ value }}</UTag>
        </template>
        <template #actions="{ row }">
          <div class="table-actions">
            <UIconButton icon="edit" :label="`编辑示例 ${row.id}`" />
          </div>
        </template>
      </UTable>
    </PanelBlock>

    <PanelBlock title="固定列表格">
      <UTable
        :columns="advancedColumns"
        :rows="advancedRows"
        row-key="id"
        fixed
        stripe
        action-width="90px"
      >
        <template #cell-status="{ value }">
          <UTag :variant="value === '启用' ? 'success' : 'error'" dot>{{ value }}</UTag>
        </template>
        <template #actions="{ row }">
          <div class="table-actions">
            <UIconButton icon="edit" :label="`编辑 ${row.name}`" />
            <UIconButton icon="detail" :label="`查看 ${row.name}`" />
          </div>
        </template>
      </UTable>
    </PanelBlock>

    <PanelBlock title="交互表格">
      <UTable
        :columns="interactiveColumns"
        :rows="interactiveRows"
        row-key="id"
        selectable
        row-clickable
        :selected-keys="interactiveSelected"
        :loading="interactiveLoading"
        loading-text="正在刷新..."
        @update:selected-keys="interactiveSelected = $event"
        @row-click="onInteractiveRowClick"
        @selection-change="onInteractiveSelectionChange"
      >
        <template #cell-status="{ value }">
          <UTag :variant="value === '启用' ? 'success' : 'error'" dot>{{ value }}</UTag>
        </template>
        <template #actions="{ row }">
          <div class="table-actions">
            <UIconButton icon="detail" :label="`查看 ${row.name}`" />
          </div>
        </template>
      </UTable>
      <div class="mt-3 flex flex-wrap items-center gap-2">
        <UButton variant="secondary" size="sm" @click="toggleInteractiveLoading">
          {{ interactiveLoading ? "停止加载" : "模拟加载" }}
        </UButton>
        <UButton variant="secondary" size="sm" :disabled="!interactiveSelected.length" @click="interactiveSelected = []">
          清除选择（{{ interactiveSelected.length }}）
        </UButton>
      </div>
    </PanelBlock>

    <PanelBlock title="Tooltip">
      <div class="flex flex-wrap items-center gap-4">
        <UTooltip content="这是一个基础提示。">
          <UButton size="sm" variant="secondary">悬停查看提示</UButton>
        </UTooltip>
        <UTooltip content="提示可以包含较长说明，用于补充界面中无法完整展示的信息。">
          <span class="text-sm text-secondary underline decoration-dotted">长文本提示</span>
        </UTooltip>
      </div>
    </PanelBlock>

    <UModal v-model:open="showModal" title="普通弹窗">
      <p class="text-sm leading-6 text-secondary">
        用于承载说明、预览或表单内容。
      </p>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="secondary" @click="showModal = false">关闭</UButton>
          <UButton @click="showModal = false">确认</UButton>
        </div>
      </template>
    </UModal>

    <UConfirmDialog
      v-model:open="showConfirm"
      title="确认删除示例"
      message="删除后将无法恢复该示例数据。"
      description="确认弹窗适用于删除、覆盖、停用等高风险操作。"
      confirm-text="确认删除"
      cancel-text="取消"
      danger
      @confirm="showConfirm = false"
    />

    <ULoading fullscreen tip="正在加载页面..." :spinning="fullscreenLoading" />
  </section>
</template>

<script setup>
import { computed, reactive, ref } from "vue";
import PanelBlock from "../components/PanelBlock.vue";
import UAlert from "../components/ued/UAlert.vue";
import UButton from "../components/ued/UButton.vue";
import UConfirmDialog from "../components/ued/UConfirmDialog.vue";
import UIconButton from "../components/ued/UIconButton.vue";
import UInput from "../components/ued/UInput.vue";
import ULoading from "../components/ued/ULoading.vue";
import UModal from "../components/ued/UModal.vue";
import USelect from "../components/ued/USelect.vue";
import USwitch from "../components/ued/USwitch.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";
import UTooltip from "../components/ued/UTooltip.vue";
import { message } from "../components/ued/message";

const showModal = ref(false);
const showConfirm = ref(false);
const fullscreenLoading = ref(false);
const switchValue = ref(true);
const interactiveSelected = ref([]);
const interactiveLoading = ref(false);

const form = reactive({
  name: "",
  protocol: "openai-responses",
  description: "",
});

const basicTableQuery = reactive({
  keyword: "",
  type: "all",
  status: "all",
});

const protocolOptions = [
  { label: "anthropic", value: "anthropic" },
  { label: "openai-chat", value: "openai-chat" },
  { label: "openai-responses", value: "openai-responses" },
];

const messageText = {
  success: "操作成功，配置已保存。",
  info: "这是一条普通提示信息。",
  warning: "请先重载代理使配置生效。",
  error: "操作失败，请检查输入后重试。",
};

const columns = [
  { key: "name", title: "名称" },
  { key: "type", title: "类型" },
  { key: "status", title: "状态" },
];

const rows = [
  { id: "1", name: "供应商按钮", type: "操作组件", status: "启用" },
  { id: "2", name: "确认弹窗", type: "反馈组件", status: "启用" },
  { id: "3", name: "状态标签", type: "展示组件", status: "启用" },
  { id: "4", name: "分页表格", type: "数据组件", status: "启用" },
  { id: "5", name: "消息提示", type: "反馈组件", status: "停用" },
  { id: "6", name: "下拉筛选", type: "表单组件", status: "启用" },
];

const basicTypeOptions = [
  { label: "全部类型", value: "all" },
  { label: "操作组件", value: "操作组件" },
  { label: "反馈组件", value: "反馈组件" },
  { label: "展示组件", value: "展示组件" },
  { label: "数据组件", value: "数据组件" },
  { label: "表单组件", value: "表单组件" },
];

const basicStatusOptions = [
  { label: "全部状态", value: "all" },
  { label: "启用", value: "启用" },
  { label: "停用", value: "停用" },
];

const filteredRows = computed(() => {
  const keyword = basicTableQuery.keyword.trim().toLowerCase();

  return rows.filter((row) => {
    const matchesKeyword = !keyword || row.name.toLowerCase().includes(keyword);
    const matchesType = basicTableQuery.type === "all" || row.type === basicTableQuery.type;
    const matchesStatus = basicTableQuery.status === "all" || row.status === basicTableQuery.status;
    return matchesKeyword && matchesType && matchesStatus;
  });
});

const advancedColumns = [
  { key: "id", title: "ID", width: "80px", fixed: "left", align: "center" },
  {
    key: "name",
    title: "名称",
    width: "220px",
    fixed: "left",
    ellipsis: true,
    tooltip: true,
  },
  {
    key: "description",
    title: "说明",
    width: "320px",
    ellipsis: true,
    tooltip: true,
  },
  { key: "type", title: "类型", width: "120px" },
  { key: "status", title: "状态", width: "90px", align: "center" },
  { key: "count", title: "计数", width: "100px", align: "right" },
];

const advancedRows = [
  {
    id: "101",
    name: "这是一个非常长的组件名称，用于测试固定列与省略号",
    description: "这是一段较长的说明文本，用于演示当单元格内容超过列宽时如何省略并通过 Tooltip 展示完整内容。",
    type: "操作组件",
    status: "启用",
    count: 128,
  },
  {
    id: "102",
    name: "确认弹窗",
    description: "用于二次确认的弹窗组件，适用于删除、覆盖等高风险操作场景。",
    type: "反馈组件",
    status: "停用",
    count: 56,
  },
  {
    id: "103",
    name: "数据表格",
    description: "支持固定列、表头固定、斑马纹、文字省略、Tooltip 提示等特性。",
    type: "数据展示",
    status: "启用",
    count: 2048,
  },
];

const interactiveColumns = [
  { key: "name", title: "名称" },
  { key: "type", title: "类型" },
  { key: "status", title: "状态", align: "center" },
];

const interactiveRows = [
  { id: "i1", name: "选择示例 A", type: "展示组件", status: "启用" },
  { id: "i2", name: "选择示例 B", type: "反馈组件", status: "停用" },
  { id: "i3", name: "选择示例 C", type: "数据组件", status: "启用" },
];

function showMessage(type) {
  message[type](messageText[type]);
}

function showLoadingMessage() {
  const key = "ued-loading-demo";
  message.loading({ key, content: "正在同步配置..." });
  window.setTimeout(() => {
    message.success({ key, content: "配置同步完成。" });
  }, 1200);
}

function showFullscreenLoading() {
  fullscreenLoading.value = true;
  window.setTimeout(() => {
    fullscreenLoading.value = false;
  }, 1200);
}

function resetBasicTableQuery() {
  basicTableQuery.keyword = "";
  basicTableQuery.type = "all";
  basicTableQuery.status = "all";
}

function toggleInteractiveLoading() {
  interactiveLoading.value = !interactiveLoading.value;
}

function onInteractiveRowClick({ row }) {
  message.info(`点击了行：${row.name}`);
}

function onInteractiveSelectionChange(keys) {
  if (keys.length) {
    message.info(`已选择 ${keys.length} 项`);
  }
}
</script>
