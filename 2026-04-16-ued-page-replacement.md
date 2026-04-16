# UED 页面替换实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将现有业务页面逐步切换到新的 `UED` 规范组件体系，并保持现有功能不回退。

**Architecture:** 采用“先接入基础样式与基础组件，再逐页替换”的渐进式方案。优先替换结构最明确的页面，避免一次性重写全部页面导致样式和交互回归。新页面优先使用 `UEDPageHeader`、`UEDPageSection`、`UEDTable`、`UEDPageShell` 等组件，旧组件保留兼容。

**Tech Stack:** Vue 3、Wails、Pinia、Tailwind 基础层、项目现有全局 CSS token、自定义 `UED` 组件

---

## 阶段一：基础接入

**文件：**
- 已完成 `frontend/src/main.js`
- 已完成 `frontend/src/styles/ued-spec.css`
- 已完成 `frontend/src/components/layout/UEDPageHeader.vue`
- 已完成 `frontend/src/components/layout/UEDPageShell.vue`
- 已完成 `frontend/src/components/layout/UEDPageSection.vue`
- 已完成 `frontend/src/components/layout/UEDTable.vue`
- 已完成 `frontend/src/components/layout/index.js`

**目标：**
- 确保新规范样式和基础组件可被页面直接引用。

---

## 阶段二：优先替换核心页面

### 任务 1：替换 `GatewayView.vue`

**文件：**
- 修改 `frontend/src/views/GatewayView.vue`

**目标：**
- 将页面头部切到 `UEDPageHeader`
- 将主要区块切到 `UEDPageSection`
- 保留现有业务逻辑、按钮行为、复制行为、状态逻辑

### 任务 2：替换 `LogsView.vue`

**文件：**
- 修改 `frontend/src/views/LogsView.vue`

**目标：**
- 将页面头部切到 `UEDPageHeader`
- 将日志列表切到 `UEDTable`
- 保留现有筛选、日志详情、行点击、状态码展示逻辑

### 任务 3：替换 `DialogRulesView.vue`

**文件：**
- 修改 `frontend/src/views/DialogRulesView.vue`

**目标：**
- 将左右结构切到 `UEDPageShell` 或 `ued-split`
- 将规则编辑区和规则列表面板切到 `UEDPageSection`
- 保留规则编辑与保存行为

### 任务 4：替换 `ProvidersView.vue`

**文件：**
- 修改 `frontend/src/views/ProvidersView.vue`

**目标：**
- 将头部切到 `UEDPageHeader`
- 将列表主区切到 `UEDTable`
- 保留抽屉、测试连接、模型映射等逻辑

---

## 阶段三：收口与验证

### 任务 5：统一导出与样式补口

**文件：**
- 视情况修改 `frontend/src/components/layout/index.js`
- 视情况修改 `frontend/src/styles/ued-spec.css`

**目标：**
- 补齐页面替换过程中发现的通用样式缺口
- 避免在页面 `scoped` 中重复实现基础视觉规则

### 任务 6：验证

**文件：**
- 修改过的页面文件

**目标：**
- 检查编译层面是否有明显导入错误
- 检查插槽、事件、列渲染是否保持兼容
- 检查样式类名是否可被新规范正确命中