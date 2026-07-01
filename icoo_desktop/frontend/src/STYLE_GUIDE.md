# UI Style Guide

> 紧凑、扁平、高信息密度的桌面控制台设计系统  
> 参考：VS Code、Linear、Raycast

---

## 1. 设计原则

- **Flat & Neutral**：背景使用纯中性灰，无蓝色 tint
- **1px Hairlines**：用发丝线边框定义层级，而非多层阴影
- **Compact Controls**：控件高度 22-34px，适合密集操作
- **Small Type Scale**：基础字号 13px，最小 11px
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

### 字号层级
| Token | 尺寸 | 用途 |
|-------|------|------|
| `--ued-font-size-xs` | 11px | 辅助信息、状态栏 |
| `--ued-font-size-sm` | 12px | 次要文本、描述 |
| `--ued-font-size-base` | 13px | 正文默认 |
| `--ued-font-size-lg` | 16px | 强调文本 |
| `--ued-font-size-title` | 18px | 页面标题 |

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
│          │  Top Bar (38px, breadcrumb)       │
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
  padding: 12px;
}
```

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

### 间距
- 页面内容区内边距：`12px`
- 卡片/面板内边距：`12px` / `16px`
- 元素间距：`8px` / `12px` / `16px`

### 控件高度
| 尺寸 | 高度 | 用途 |
|------|------|------|
| xs | 22px | 紧凑标签 |
| sm | 26px | 表格内操作 |
| md | 28px | 默认按钮 |
| lg | 34px | 强调按钮 |

### 圆角
- 控件：`3px` / `4px`
- 卡片/面板：`6px`
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
5. **保持紧凑风格**：遵循 28px 按钮、13px 正文、6px 圆角的紧凑规范

---

## 13. 相关文件

- `frontend/src/main.css` — 全局样式库
- `frontend/src/components/ued/` — UED 组件库
- `frontend/tailwind.config.js` — Tailwind 配置
