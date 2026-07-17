# UI Style Guide

> 紧凑、扁平、高信息密度的桌面控制台设计系统  
> 参考：VS Code、Linear、Raycast

---

## 1. 设计原则

- **Flat & Neutral**：背景使用纯中性灰，无蓝色 tint
- **1px Hairlines**：用发丝线边框定义层级，而非多层阴影
- **Compact Controls**：控件高度 20–32px（默认 SM），适合密集操作
- **Small Type Scale**：紧缩模式基础字号 12px，最小 10px
- **Primary Color Unity**：焦点环、激活态、链接统一使用 `#2563eb`

---

## 2. 颜色系统

### 主色调
```css
--ued-color-primary: #2563eb;        /* 主色 */
--ued-color-primary-hover: #3b6ef0;  /* 悬停 */
--ued-color-primary-active: #1d4ed8; /* 按下 */
--ued-color-primary-soft: #eaf1ff;   /* 柔化背景 */
```

### 中性色阶
```css
--ued-color-bg-page: #f3f4f6;    /* 页面背景 */
--ued-color-bg-shell: #eceef1;   /* 外壳背景 */
--ued-color-bg-card: #ffffff;    /* 卡片背景 */
--ued-color-border: #d4d7dc;     /* 边框 */
--ued-color-border-light: #e4e6ea; /* 浅边框 */
--ued-color-text: #1f2329;       /* 主文字 */
--ued-color-text-secondary: #4b5563; /* 次要文字 */
--ued-color-text-muted: #6b7280;  /* 弱化文字 */
```

### 状态色
```css
--ued-color-success: #16a34a;
--ued-color-success-soft: #ecfaf0;
--ued-color-warning: #c2790a;
--ued-color-warning-soft: #fdf6e8;
--ued-color-destructive: #dc2626;
--ued-color-error-soft: #fdf0ef;
--ued-color-info: #2563eb;
--ued-color-info-soft: #eaf1ff;
```

---

## 3. 字体排版

### 字体族
```css
font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", Arial, sans-serif;
/* 等宽字体 */
font-family: "Cascadia Code", "SFMono-Regular", Consolas, monospace;
```

### 字号层级（紧缩 compact 默认）
| Token | 尺寸 | 用途 |
|-------|------|------|
| `--ued-font-size-xs` | 10px | 辅助信息、状态栏 |
| `--ued-font-size-sm` | 11px | 次要文本、描述 |
| `--ued-font-size-base` | 12px | 正文默认 |
| `--ued-font-size-lg` | 14px | 强调文本 |
| `--ued-font-size-title` | 16px | 页面标题 |

> 宽松（comfortable）模式在此基础上 +1px 左右，由 `uiPrefs` 运行时覆盖。

### 字重
- `font-medium` (500)：按钮、标签
- `font-semibold` (600)：标题、重要文本
- `font-mono`：代码、路径、地址

---

## 4. 布局架构

### 经典桌面布局
```
┌─────────────────────────────────────────────┐
│  Title Bar (28px, dark)                      │
├──────────┬──────────────────────────────────┤
│          │  Top Bar (36px, breadcrumb)       │
│  Sidebar │──────────────────────────────────┤
│  168px   │                                   │
│  (可折叠  │  Main Content (scrollable)        │
│   52px)   │                                   │
│          │                                   │
├──────────┴──────────────────────────────────┤
│  Status Bar (20px)                           │
└─────────────────────────────────────────────┘
```

### 内容区内边距
```css
.app-content {
  padding: var(--ued-space-page); /* compact: 8px · comfortable: 12px */
}
```

### 间距刻度（紧缩 baseline）
| Token | 值 | 语义别名 |
|-------|-----|---------|
| space-1…3 | 2 / 3 / 4px | inline / chip |
| space-4…5 | 6 / 8px | control / panel |
| space-6…8 | 10 / 10 / 12px | section / card |
| page / section / panel | 8px | 页面与块级 |
| table row / header | 30 / 28px | 表格密度 |

---

## 5. 统一布局类

### 5.1 查询表单 `.query-form`
```html
<div class="query-form">
  <UInput class="query-form__field" label="关键词" hide-label />
  <USelect class="query-form__field query-form__field--compact" label="类型" hide-label :options="options" />
  <div class="query-form__actions">
    <UButton variant="secondary" @click="reset">重置</UButton>
    <UButton variant="primary" @click="search">查询</UButton>
  </div>
</div>
```

### 5.2 表单网格 `.form-grid`
```html
<form class="form-grid">
  <UInput v-model="form.name" label="名称" />
  <USelect v-model="form.type" label="类型" :options="options" />
  <UInput v-model="form.url" label="地址" />
  <UInput v-model="form.key" label="密钥" />
</form>
```

### 5.3 统计卡片网格 `.stat-grid`
```html
<div class="stat-grid stat-grid--4">
  <StatCard icon="server" label="总数" :value="count" tone="info" />
  <StatCard icon="check" label="已启用" :value="enabled" tone="success" />
  <StatCard icon="alert" label="错误" :value="errors" tone="danger" />
  <StatCard icon="timer" label="耗时" :value="latency" />
</div>
```

| 修饰符 | 列数 | 响应式断点 |
|--------|------|-----------|
| `stat-grid--2` | 2列 | 760px → 1列 |
| `stat-grid--3` | 3列 | 960px → 2列 |
| `stat-grid--4` | 4列 | 960px → 3列 |
| `stat-grid--5` | 5列 | 1180px → 4列, 960px → 3列 |

### 5.4 Token 单元格 `.token-cell`
```html
<div class="token-cell">
  <div class="token-cell__row">
    <span class="token-cell__label">输入</span>
    <span class="token-cell__value">1,234</span>
  </div>
  <div class="token-cell__row token-cell__row--total">
    <span class="token-cell__label">总计</span>
    <span class="token-cell__value">5,678</span>
  </div>
</div>
```

### 5.5 徽章 Chip `.query-form__chip`
```html
<UTag variant="primary" size="xs" dot>已启用</UTag>
```

---

## 6. 组件使用规范

### 6.1 按钮 UButton
```html
<!-- 主要操作 -->
<UButton variant="primary" @click="save">保存</UButton>

<!-- 次要操作 -->
<UButton variant="secondary" @click="cancel">取消</UButton>

<!-- 危险操作 -->
<UButton variant="error" @click="delete">删除</UButton>

<!-- 尺寸 -->
<UButton size="xs">XS</UButton>
<UButton size="sm">SM</UButton>
<UButton size="md">MD</UButton>
<UButton size="lg">LG</UButton>

<!-- 加载状态 -->
<UButton :loading="saving">保存中...</UButton>
```

### 6.2 表格 UTable
```html
<UTable
  :columns="columns"
  :rows="rows"
  row-key="id"
  fixed
  stripe
  size="sm"
  :min-width="1200"
  pagination
  :page="page"
  :page-size="pageSize"
  :total="total"
  @page-change="onPageChange"
>
  <template #query>
    <div class="query-form">
      <!-- 查询表单 -->
    </div>
  </template>
  <template #cell-status="{ value }">
    <UTag :variant="value === '启用' ? 'success' : 'error'" dot>{{ value }}</UTag>
  </template>
  <template #actions="{ row }">
    <div class="table-actions">
      <UIconButton icon="edit" label="编辑" @click="edit(row)" />
    </div>
  </template>
</UTable>
```

### 6.3 表单组件
```html
<!-- 输入框 -->
<UInput v-model="form.name" label="名称" placeholder="请输入" required />

<!-- 下拉选择 -->
<USelect v-model="form.type" label="类型" :options="options" required />

<!-- 多行文本 -->
<UInput v-model="form.desc" label="描述" textarea placeholder="请输入描述" />

<!-- 开关 -->
<USwitch v-model="form.enabled" label="启用" hint="启用后将立即生效" />
```

### 6.4 反馈组件
```html
<!-- 警告提示 -->
<UAlert type="warning" message="标题" description="详细描述" closable />

<!-- 全局消息 -->
<UTag variant="success" dot>成功</UTag>

<!-- 加载 -->
<ULoading tip="加载中..." />

<!-- 弹窗 -->
<UModal v-model:open="open" title="标题" width="640px">
  <p>内容</p>
  <template #footer>
    <UButton variant="secondary" @click="open = false">取消</UButton>
    <UButton @click="confirm">确认</UButton>
  </template>
</UModal>

<!-- 确认对话框 -->
<UConfirmDialog
  v-model:open="open"
  title="确认删除"
  message="删除后无法恢复"
  confirm-text="确认删除"
  cancel-text="取消"
  danger
  @confirm="onConfirm"
/>
```

---

## 7. 页面结构模板

### 7.1 标准列表页
```html
<section class="page-section">
  <!-- 顶部操作栏 -->
  <Teleport to="#app-topbar-actions">
    <div class="app-topbar-actions__group">
      <UButton variant="primary" @click="create">新建</UButton>
    </div>
  </Teleport>

  <!-- 统计卡片 -->
  <div class="stat-grid stat-grid--4">
    <StatCard icon="server" label="总数" :value="total" tone="info" />
    <StatCard icon="check" label="已启用" :value="enabled" tone="success" />
    <StatCard icon="alert" label="错误" :value="errors" tone="danger" />
    <StatCard icon="timer" label="耗时" :value="latency" />
  </div>

  <!-- 数据表格 -->
  <UTable ...>
    <template #query>
      <div class="query-form">
        <!-- 查询条件 -->
      </div>
    </template>
  </UTable>
</section>
```

### 7.2 表单页
```html
<section class="page-section">
  <div class="page-header">
    <h2 class="page-title">标题</h2>
    <p class="page-description">描述文本</p>
  </div>

  <div class="content-panel">
    <form class="form-grid" @submit.prevent="submit">
      <UInput v-model="form.name" label="名称" required />
      <USelect v-model="form.type" label="类型" :options="options" />
      <UInput v-model="form.url" label="地址" />
      <UInput v-model="form.key" label="密钥" />
    </form>
  </div>
</section>
```

---

## 8. 间距与尺寸

### 8.0 密度模式 Density

界面提供两种全局密度，由 `uiPrefs.density` 写入 `document.documentElement[data-density]`，并覆盖间距 / 表格行高 / 字号基线：

| 模式 | 值 | 特征 | 默认控件尺寸建议 |
|------|-----|------|------------------|
| **紧缩** | `compact` | 页面 8、表行 30、stack 6；全局 UED 收紧基线 | `buttonSize=sm` |
| **宽松** | `comfortable` | 页面 12、表行 36、stack 8；相对更易读 | `buttonSize=md` |

应用顺序：`theme` → `density` → `buttonSize`（控件高度在密度基线之上微调）。

```js
// stores/uiPrefs.js
uiPrefs.setDensity("comfortable"); // or "compact"
```

设置页「外观 → 界面密度」可切换；持久化字段 `density`（`/api/v1/ui-prefs`）。

### 8.1 间距刻度（Spacing Scale）

以 **4px 为基准** 的紧凑刻度，统一使用 CSS 变量，禁止在业务样式中硬编码随机间距。

| Token | 值 | 典型用途 |
|-------|-----|----------|
| `--ued-space-0` | 0 | 重置 |
| `--ued-space-1` | 2px | 微偏移 |
| `--ued-space-2` | 3px | 紧凑 inline |
| `--ued-space-3` | 4px | 图标+文字、chip 行 |
| `--ued-space-4` | 6px | **默认元素间距**、按钮组 |
| `--ued-space-5` | 8px | **页面/面板默认** |
| `--ued-space-6` | 10px | 区块 |
| `--ued-space-7` | 10px | 表单元格（兼容档） |
| `--ued-space-8` | 12px | 卡片内容 |
| `--ued-space-10` | 14px | 空状态 |
| `--ued-space-12` | 16px | 大区块 |
| `--ued-space-16` | 24px | 页面级（少用） |

### 8.2 语义别名（Semantic）

| Token | 映射 | 场景 |
|-------|------|------|
| `--ued-space-page` | 8px | `.app-content` 内边距（紧缩） |
| `--ued-space-section` | 8px | 页面块间距 |
| `--ued-space-panel` | 8px | 面板 body |
| `--ued-space-panel-sm` | 6px | 面板 header |
| `--ued-space-stack` | 6px | 表单字段纵排 |
| `--ued-space-inline` | 4px | 图标+文案、标签横排 |
| `--ued-space-control` | 6px | 按钮组间距 |
| `--ued-space-table-x` | 6px | 表格外壳左右外边距 |
| `--ued-space-table-cell-x` | 8px | 表头/单元格水平内边距 |

### 8.3 工具类

**内边距**

```html
<div class="ued-p-6">四边 10px</div>
<div class="ued-px-4 ued-py-2">左右 6 / 上下 3</div>
<div class="ued-pt-4 ued-pb-6">上 6 / 下 10</div>
```

支持：`ued-p-{0,1,2,3,4,5,6,7,8,10,12,16}`，以及 `px` / `py` / `pt` / `pb` / `pl` / `pr` 同档位子集。

**外边距**

```html
<div class="ued-m-4">四边 6px</div>
<div class="ued-mx-auto">水平居中</div>
<div class="ued-mt-6 ued-mb-4">上 10 / 下 6</div>
```

支持：`ued-m-*`、`ued-mx-*`、`ued-my-*`、`ued-mt-*`、`ued-mb-*`、`ued-ml-*`、`ued-mr-*`（含 `auto`）。

**间隙 gap**

```html
<div class="ued-inline ued-gap-3">横排</div>
<div class="ued-stack ued-gap-4">纵排</div>
```

- `ued-gap-{0,1,2,3,4,5,6,8,10,12}`
- `ued-stack` / `ued-stack--sm|md|lg`：纵向 flex + 默认 stack 间距
- `ued-inline` / `ued-inline--sm|md|lg`：横向 flex-wrap + 默认 inline 间距

### 8.4 布局落点（现网约定 · 紧缩）

| 区域 | 推荐 |
|------|------|
| 主内容区 `.app-content` | `padding: var(--ued-space-page)` → 8px |
| 页面块间距 `.page-section` | `gap: var(--ued-space-section)` → 8px |
| 面板 header / body | 6 / 8px（`--ued-space-panel-sm` / `--ued-space-panel`） |
| 表单字段纵向 | `gap: var(--ued-space-stack)` → 6px |
| 顶栏按钮组 | `gap: var(--ued-space-control)` → 6px |
| 表格行 / 表头 | `--ued-table-row-height` 30px / header 28px |
| 表格外壳左右 | `margin-inline: var(--ued-space-table-x)` → 8px |
| 表格单元格水平 | `padding-inline: var(--ued-space-table-cell-x)` → 14px |

### 8.5 控件高度（紧缩基线 / buttonSize=sm）
| 尺寸 | 高度 | 用途 |
|------|------|------|
| xs | 20px | 紧凑标签 |
| sm | 24px | 表格内操作 / 默认预设 |
| md | 26px | 常规按钮 |
| lg | 30px | 强调按钮 |

### 8.6 圆角
- 控件：`3px` / `4px`（`--ued-radius-sm` / `md`）
- 卡片/面板：`6px`（`--ued-radius-card`）
- 标签：`4px`
- 胶囊：`9999px`

---

## 9. 交互状态

### 按钮
- **默认**：边框 + 背景色
- **悬停**：颜色加深，无位移/无阴影提升
- **按下**：颜色再加深
- **禁用**：`opacity: 60%`，`cursor: not-allowed`

### 输入框
- **默认**：边框 `#c4c8cf`
- **悬停**：边框 `#6b7280`
- **聚焦**：边框 `#2563eb` + 焦点环 `2px solid color-mix(primary 14%, transparent)`

### 表格行
- **悬停**：背景 `#f4f5f7`
- **选中**：背景 `#eef4ff`

---

## 10. 命名规范

### CSS 类名
- 使用 BEM 风格：`.block__element--modifier`
- 全局类：小写 + 连字符
- 视图特定类：已统一迁移到全局布局类

### Vue 组件
- 文件名：`PascalCase.vue`
- 组件名：`PascalCase`
- Props：`camelCase`
- 事件：`kebab-case` (`@update:model-value`)

### 设计 Token
- 前缀：`--ued-`
- 颜色：`--ued-color-*`
- 尺寸：`--ued-size-*`
- 间距：`--ued-space-*`（刻度）/ `--ued-space-page` 等（语义）
- 字体：`--ued-font-size-*`
- 圆角：`--ued-radius-*`
- 阴影：`--ued-shadow-*`

---

## 11. 已完成的统一工作

### 迁移的样式
- [x] `TrafficView.vue`：移除 150+ 行 scoped 样式，迁移到全局 `.stat-grid`、`.query-form`、`.token-cell`
- [x] `SuppliersView.vue`：移除 `.provider-form-grid`，迁移到全局 `.form-grid`
- [x] `RoutingRulesView.vue`：移除空 style 块
- [x] `UedSpecView.vue`：移除空 style 块

### 新增的全局类
- [x] `.query-form` / `.query-form__field` / `.query-form__actions`
- [x] `.form-grid`（响应式两列布局）
- [x] `.stat-grid` / `.stat-grid--2` ~ `.stat-grid--5`
- [x] `.token-cell` / `.token-cell__row` / `.token-cell__value`

---

## 12. 后续开发规范

1. **优先使用全局类**：新增视图时优先使用 `.query-form`、`.form-grid`、`.stat-grid` 等全局类
2. **避免 scoped 样式**：视图层应避免 `<style scoped>`，全局样式统一放在 `main.css`
3. **使用 UED 组件**：按钮、输入、表格等必须使用 `ued/` 组件库
4. **遵循设计 Token**：颜色、间距、字号使用 CSS 变量，不硬编码
5. **间距只用刻度**：内边距/外边距/gap 使用 `--ued-space-*` 或 `ued-p-*` / `ued-m-*` / `ued-gap-*`，禁止随意 `13px`、`15px`
6. **保持紧凑风格**：默认 `buttonSize=sm`（约 24–26px 控件）、12px 正文、6px 圆角

---

## 13. 相关文件

- `frontend/src/main.css` — 全局样式库
- `frontend/src/components/ued/` — UED 组件库
- `frontend/tailwind.config.js` — Tailwind 配置
