# icoo_proxy UI 评审与 Windows PC 亮色风格设计规范

> 文档版本: v1.0  
> 创建日期: 2026-04-15  
> 适用范围: `frontend/src` 现有 Wails + Vue 桌面前端  
> 设计目标: 将当前偏 VS Code 工具面板风格的界面，重构为更接近 Windows PC 亮色桌面应用的 UI 体系

---

## 1. 文档结论

当前项目已经具备较完整的组件基础、主题变量和页面框架，但整体视觉语言仍然偏向“开发者工具 / IDE 面板风格”，与“Windows PC 亮色桌面应用”存在明显差距，主要体现在：

- 标题栏、侧边栏、状态栏采用了强烈的 VS Code 语义和配色
- 页面层级依赖渐变、高饱和强调色和卡片堆叠，不够克制
- 控件圆角、阴影、按钮质感更接近 Web SaaS，而非桌面工具
- 信息组织偏“模块展示”，缺少 Windows 桌面常见的“分组面板 + 明确表单区 + 标准工具栏”秩序
- 主题系统支持多彩换肤，但缺少主品牌色克制、灰阶稳定、系统化状态色约束

建议将项目 UI 改造为以下方向：

- 视觉定位: Windows 11 亮色桌面工具，但避免完全复制系统 UI
- 风格关键词: 清晰、克制、轻办公、系统感、专业配置工具
- 信息架构: 左侧导航 + 顶部上下文工具栏 + 内容工作区 + 底部状态栏
- 控件质感: 低饱和灰白底、细描边、轻阴影、明确焦点、高可读文本
- 交互原则: 稳定优先，减少花哨动画，增强表单、列表、状态反馈的可预期性

---

## 2. 当前 UI 评审

### 2.1 现状优点

- 全局结构完整，已形成 `header + sidebar + main + footer` 的桌面应用骨架，见 [App.vue](E:/codes/icoo_proxy/frontend/src/App.vue:1)
- 已有较丰富的全局 token 和基础样式，可作为重构起点，见 [style.css](E:/codes/icoo_proxy/frontend/src/style.css:1)
- 页面头部、对话框、按钮、输入框等已有复用基础，组件化程度不错
- 页面信息密度适中，适合继续向“配置型桌面工具”演进
- 业务页面边界清晰，网关、供应商、日志、设置已经具备独立模块化结构

### 2.2 主要问题

#### A. 风格不统一

- 根样式使用了 VS Code Light+/Dark+ 命名和视觉语义，如 `--vscode-*`，这会把产品风格锁死在 IDE 语境里，见 [style.css](E:/codes/icoo_proxy/frontend/src/style.css:14)
- 标题栏、侧栏、状态栏、页内卡片分别采用不同的质感表达，缺乏同一套桌面界面语言
- 一部分组件使用 Tailwind 语义类，一部分使用传统 CSS 类，视觉细节容易漂移

#### B. 不够像 Windows 亮色桌面应用

- 当前导航栏是“竖向图标活动栏”思路，更像编辑器侧边栏，不像常见 Windows 配置工具
- 页面表面大量使用渐变、高亮发光和强调色混合背景，显得偏 Web 化
- 圆角偏大，很多卡片和按钮使用 `lg/xl` 圆角，不够系统化
- 底部蓝色状态栏过于 IDE 化，压过了主内容层级

#### C. 信息层级需要重构

- 首页网关页采用多块卡片并列，但重要任务没有形成主次顺序，见 [GatewayView.vue](E:/codes/icoo_proxy/frontend/src/views/GatewayView.vue:1)
- 供应商页卡片化展示适合少量对象，但当供应商增多时，不如“表格 + 详情面板”高效，见 [ProvidersView.vue](E:/codes/icoo_proxy/frontend/src/views/ProvidersView.vue:1)
- 设置页已经有侧栏结构，但视觉上仍偏 Web 后台，缺少 Windows 设置类页面的稳定秩序，见 [SettingsView.vue](E:/codes/icoo_proxy/frontend/src/views/SettingsView.vue:1)

#### D. 桌面端可用性细节不足

- 标题栏图标和主题切换菜单偏“悬浮工具”表现，不够原生桌面工具化
- 缺少统一的数据表格规范、主次按钮规范、危险操作规范
- 缺少空状态、错误状态、加载骨架、禁用状态的统一视觉定义
- 缺少明确的焦点环、键盘路径和高对比度策略说明

### 2.3 评审评分

| 维度 | 评分 | 说明 |
|------|------|------|
| 视觉一致性 | 6/10 | 有基础，但全局语义混杂 |
| 桌面工具感 | 5/10 | 结构像桌面应用，细节更像 Web 后台 |
| 信息组织 | 7/10 | 模块清晰，但层次可继续压缩优化 |
| 可扩展性 | 8/10 | 组件与变量已经具备可重构基础 |
| Windows 亮色匹配度 | 4/10 | 当前更接近 VS Code Light 风格 |

综合结论: 当前 UI 可用，但品牌感、系统感和桌面工具气质还没有建立起来，适合进行一次“设计语言层”的统一升级，而不是仅做局部美化。

---

## 3. 新设计目标

### 3.1 目标用户感知

用户打开应用后，应当立即感受到：

- 这是一个稳定、可靠、偏本地管理工具的桌面应用
- 配置操作是严谨的，不会因为视觉花哨而打扰判断
- 重要状态、错误和连接信息一眼可读
- 即使是复杂配置，也能沿着清晰分组逐步完成

### 3.2 设计原则

1. 系统感优先  
   视觉和布局优先贴近 Windows 桌面工具，而不是 Web 营销页或 IDE 外壳。

2. 信息优先于装饰  
   所有渐变、阴影、色块只服务层级，不服务“好看但无意义”的视觉堆叠。

3. 表单与状态优先  
   这是配置型产品，核心不是展示，而是录入、查看、验证、切换和诊断。

4. 高密度但不拥挤  
   适配 PC 宽屏，允许更高信息密度，但要依赖网格、留白和分组保持秩序。

5. 亮色主导，彩色克制  
   默认以中性灰白为主，品牌蓝只用于焦点、主按钮、选中态和关键状态。

---

## 4. 目标视觉语言

### 4.1 风格基调

- 基础背景: 冷白偏灰
- 面板背景: 纯白或极浅灰
- 描边: 低对比中性灰
- 文字: 深灰接近黑，但不使用纯黑
- 品牌强调: Windows 风格蓝
- 状态色: 成功绿、警告橙、错误红，饱和度控制在中等

### 4.2 字体建议

优先级:

- 中文: `"Segoe UI Variable"`, `"Segoe UI"`, `"Microsoft YaHei UI"`, `"Microsoft YaHei"`, `"PingFang SC"`
- 英文和数字: 与中文共用系统字体栈
- 等宽: `"Cascadia Code"`, `"Consolas"`, `"JetBrains Mono"`

建议不再以 `Nunito UI` 作为主文本字体。它偏柔和亲和，不够像专业桌面配置工具。

### 4.3 视觉气质关键词

- Professional
- Crisp
- Light Utility
- Calm
- Structured

---

## 5. Design Tokens

以下 token 作为新规范基线，建议替换现有 `--vscode-*` 命名体系，建立项目自己的语义层。

### 5.1 Core Color Tokens

```css
:root {
  --ui-bg-app: #f3f5f8;
  --ui-bg-window: #f7f8fa;
  --ui-bg-surface: #ffffff;
  --ui-bg-surface-muted: #f6f7f9;
  --ui-bg-surface-hover: #eef2f7;
  --ui-bg-surface-active: #e7edf7;

  --ui-border-subtle: #e7ebf0;
  --ui-border-default: #d8dee8;
  --ui-border-strong: #c3ccda;

  --ui-text-primary: #1f2329;
  --ui-text-secondary: #4b5563;
  --ui-text-muted: #6b7280;
  --ui-text-disabled: #9aa4b2;

  --ui-accent: #0a64d8;
  --ui-accent-hover: #005fcb;
  --ui-accent-pressed: #0056b7;
  --ui-accent-soft: #e8f1fe;

  --ui-success: #157347;
  --ui-success-soft: #e8f5ee;
  --ui-warning: #b76e00;
  --ui-warning-soft: #fff4df;
  --ui-danger: #c42b1c;
  --ui-danger-soft: #fdecea;
  --ui-info: #0a64d8;
  --ui-info-soft: #e8f1fe;
}
```

### 5.2 Typography Tokens

```css
:root {
  --font-ui: "Segoe UI Variable", "Segoe UI", "Microsoft YaHei UI", "PingFang SC", sans-serif;
  --font-mono: "Cascadia Code", "Consolas", monospace;

  --text-display: 24px;
  --text-title-1: 20px;
  --text-title-2: 16px;
  --text-body: 13px;
  --text-body-small: 12px;
  --text-caption: 11px;

  --line-display: 32px;
  --line-title-1: 28px;
  --line-title-2: 22px;
  --line-body: 20px;
  --line-body-small: 18px;
  --line-caption: 16px;
}
```

### 5.3 Spacing Tokens

```css
:root {
  --space-2: 2px;
  --space-4: 4px;
  --space-6: 6px;
  --space-8: 8px;
  --space-10: 10px;
  --space-12: 12px;
  --space-16: 16px;
  --space-20: 20px;
  --space-24: 24px;
  --space-32: 32px;
}
```

### 5.4 Radius Tokens

Windows 亮色桌面工具建议使用更克制的圆角：

```css
:root {
  --radius-xs: 4px;
  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 10px;
}
```

规则:

- 输入框、按钮、标签: `6px`
- 卡片、分组面板、对话框: `8px`
- 大容器和整页面板: `10px`

### 5.5 Shadow Tokens

```css
:root {
  --shadow-rest: 0 1px 2px rgba(16, 24, 40, 0.04);
  --shadow-panel: 0 4px 12px rgba(16, 24, 40, 0.06);
  --shadow-dialog: 0 16px 40px rgba(16, 24, 40, 0.14);
  --shadow-focus: 0 0 0 3px rgba(10, 100, 216, 0.18);
}
```

---

## 6. 桌面布局规范

### 6.1 应用总体结构

推荐结构:

```text
+ Title Bar
+ Navigation Rail / Navigation Pane
+ Context Toolbar
+ Content Workspace
+ Status Bar
```

### 6.2 标题栏

目标风格:

- 高度 40px 到 44px
- 背景使用浅灰白纯色，不使用明显渐变
- 左侧为应用图标 + 应用名称 + 当前模块名
- 中间保留拖拽区
- 右侧只保留必要工具，如刷新、帮助、设置入口
- 窗口控制按钮遵循 Windows 习惯，hover 态清晰，关闭按钮红色 hover

不建议:

- 在标题栏放过多换肤功能
- 使用过强品牌色作为标题栏底色
- 使用浮层式色板切换作为一等操作

### 6.3 导航区

建议将当前“仅图标活动栏”升级为“窄导航栏 + 文本标签明显可读”的桌面导航：

- 宽度: 200px 到 220px
- 每个导航项高度: 36px 到 40px
- 图标 16px
- 文本 13px
- 选中态表现: 浅蓝底 + 左侧 3px 强调条，而不是顶部短横条

推荐导航项:

- 网关总览
- 路由规则
- 供应商
- 日志审计
- 设置

### 6.4 内容工作区

内容页分为三层：

1. 页面标题区  
   显示页面标题、摘要说明、右侧主操作按钮

2. 上下文工具栏  
   显示筛选、搜索、批量操作、二级状态

3. 主工作面板  
   采用一个主面板或“主面板 + 次面板”结构，而不是过多漂浮卡片

### 6.5 状态栏

状态栏保留，但应弱化 IDE 感：

- 高度 24px 到 28px
- 默认浅灰背景，不使用高饱和纯蓝铺满
- 左侧显示网关状态、端口、连接状态
- 右侧显示版本号、运行环境或最后同步时间
- 仅在异常或运行关键提示时局部着色，不要整栏抢视觉焦点

---

## 7. 页面级设计规范

### 7.1 网关总览页

目标定位:

- 这是全局控制台，不是营销式 Dashboard
- 用户最关心的是“能不能用、在哪监听、是否安全、当前模型是否可访问”

推荐结构:

1. 顶部摘要带  
   显示运行状态、监听地址、已启用供应商数、可用模型数

2. 网关控制面板  
   启动、停止、刷新模型、复制 API 地址

3. 安全与鉴权面板  
   访问密钥、显示/隐藏、生成、保存、启用状态

4. 接口调用示例面板  
   采用标签页或分组块，不要连续堆多个大 `pre` 区块

5. 诊断提示区  
   显示“未配置供应商”“网关未启动”“默认模型未设置”等关键提示

改造建议:

- 统计卡从“彩色卡片”改为“紧凑状态摘要条”
- `curl` 示例用代码面板承载，并支持复制
- 鉴权面板采用标准表单布局，左侧说明，右侧输入与操作

### 7.2 供应商管理页

目标定位:

- 这是一个资产管理页，优先强调检索、状态、编辑效率

推荐结构:

1. 顶部工具栏  
   搜索、类型筛选、状态筛选、添加供应商

2. 主内容使用表格  
   列建议: 名称、类型、端点模式、API Base、模型数、优先级、状态、操作

3. 右侧详情抽屉或模态  
   用于编辑供应商基础信息

4. 模型映射使用专门编辑器  
   建议保留独立全屏对话框

原因:

- 当供应商超过 6 个时，卡片浏览效率明显下降
- 表格更符合 Windows 工具类应用的管理心智
- 详情编辑保留在弹窗或侧栏，更有“配置中心”感

### 7.3 对话规则页

建议定位为“规则列表 + 编辑器”的双栏结构：

- 左侧规则列表
- 右侧规则详情
- 顶部支持新增、启用/禁用、测试规则

避免每条规则都用独立大卡片铺满页面。

### 7.4 日志页

建议采用典型桌面日志视图：

- 顶部筛选条
- 中部日志表格
- 底部或右侧详情查看器

日志列表建议字段:

- 时间
- 请求方向
- 供应商
- 模型
- 状态码
- 延迟
- Token
- 结果

### 7.5 设置页

建议参考 Windows 设置与企业工具配置页的折中形态：

- 左侧固定分类导航
- 右侧单列滚动内容
- 每个设置分组使用清晰区块，不要过度卡片化
- 分组内部优先使用“标题 + 辅助文案 + 控件”的标准设置行

---

## 8. 组件规范

### 8.1 按钮

按钮层级:

- Primary: 主流程提交、保存、启动
- Secondary: 次操作、取消、刷新
- Tertiary/Ghost: 辅助操作
- Danger: 删除、停用、重置

规范:

- 高度: 32px
- 字号: 13px
- 圆角: 6px
- 内边距: `0 12px`
- 图标尺寸: 14px
- 主按钮底色使用品牌蓝纯色，不建议持续使用蓝色渐变

状态:

- Rest
- Hover
- Pressed
- Disabled
- Focus Visible

### 8.2 输入框

规范:

- 高度: 32px
- 背景: 白色
- 描边: `--ui-border-default`
- 焦点: 蓝色边框 + 外圈焦点环
- 错误: 红色边框 + 浅红提示文案
- 密码字段支持显隐按钮

表单标签:

- 标签字号: 12px
- 标签颜色: `--ui-text-secondary`
- 标签与输入框间距: 6px

### 8.3 选择器 / 下拉框

- 与输入框等高
- 打开面板使用白底 + 细边框 + 小阴影
- 列表项 hover 使用浅蓝灰背景
- 当前选中项左侧或右侧提供明确勾选标记

### 8.4 卡片与面板

新规范中“卡片”不是主视觉元素，只是容器手段。

规范:

- 默认白底
- 1px 描边
- 8px 圆角
- 轻阴影或无阴影
- 面板标题与正文间距统一

不要:

- 大面积渐变卡片
- 每个区域都像宣传模块
- 为了“高级感”堆叠阴影和色块

### 8.5 表格

供应商页、日志页、规则页建议统一使用表格规范：

- 行高: 40px
- 表头高度: 36px
- 表头背景: `--ui-bg-surface-muted`
- 行 hover: `--ui-bg-surface-hover`
- 选中行: `--ui-accent-soft`
- 单元格文字优先左对齐
- 长文本显示省略并支持 tooltip

### 8.6 状态徽标

规范:

- Success: 绿字浅绿底
- Warning: 橙字浅橙底
- Error: 红字浅红底
- Neutral: 灰字浅灰底
- 高度 22px
- 字号 11px
- 圆角 999px

### 8.7 对话框

规范:

- 常规宽度: 640px
- 大对话框: 880px 到 1040px
- 标题区、正文区、底部按钮区分层明确
- 底部操作条固定在下方，右对齐主按钮
- 危险确认弹窗必须突出风险文案

---

## 9. 状态设计

### 9.1 Loading

统一策略:

- 页面首次加载: 使用骨架屏
- 局部刷新: 使用按钮 loading 或局部占位
- 表格加载: 使用表格骨架行

避免:

- 大面积转圈占据主视区
- 所有加载都只靠 Toast 提示

### 9.2 Empty

空状态应包含：

- 简短标题
- 一句说明
- 一个明确下一步操作

示例:

- 暂无供应商，去添加第一个供应商
- 暂无日志，启动网关并发送一次请求后会在这里显示
- 暂无模型映射，添加映射后可用于统一路由

### 9.3 Error

错误信息分层:

- 行内错误: 表单字段下方
- 面板错误: 面板顶部 Alert
- 全局错误: Toast 或对话框

错误文案必须包含：

- 出错对象
- 原因摘要
- 建议动作

### 9.4 Disabled

禁用态不能只靠透明度表现，还应包含：

- 文本颜色降低
- 边框和背景降低对比
- 鼠标样式禁用
- 必要时补充禁用原因 tooltip

---

## 10. 交互与动效规范

### 10.1 动效原则

- 动效时长 120ms 到 180ms
- 只用于状态切换、浮层出现、列表反馈
- 不使用大幅位移和浮夸弹跳

### 10.2 建议动效

- 页面切换: 轻微淡入 + 4px 位移
- 弹窗出现: 透明度 + 缩放 0.98 到 1
- 按钮按下: 轻微颜色变深
- 表格行 hover: 背景色渐变过渡

### 10.3 不建议动效

- 强烈发光
- 大面积渐变扫光
- 卡片漂浮式动画
- 高频无意义骨架闪烁

---

## 11. 可访问性规范

最低要求: WCAG AA

### 11.1 对比度

- 主文本对比度不低于 4.5:1
- 次文本不低于 3:1
- 不依赖颜色单独表达状态

### 11.2 键盘可达性

- 所有按钮、输入框、选择器、导航项都必须可 Tab 到达
- 焦点样式必须统一且明显
- 弹窗打开后焦点落在首个可操作项
- 弹窗关闭后焦点返回触发元素

### 11.3 语义化

- 表单项补齐 label
- 图标按钮必须有 `title` 或 `aria-label`
- 状态提示区需使用语义化容器

---

## 12. 与当前代码结构的映射建议

### 12.1 推荐优先改造文件

第一批:

- [style.css](E:/codes/icoo_proxy/frontend/src/style.css:1)
- [App.vue](E:/codes/icoo_proxy/frontend/src/App.vue:1)
- [PageHeader.vue](E:/codes/icoo_proxy/frontend/src/components/layout/PageHeader.vue:1)
- [Button.vue](E:/codes/icoo_proxy/frontend/src/components/ui/Button.vue:1)
- [Input.vue](E:/codes/icoo_proxy/frontend/src/components/ui/Input.vue:1)

第二批:

- [GatewayView.vue](E:/codes/icoo_proxy/frontend/src/views/GatewayView.vue:1)
- [ProvidersView.vue](E:/codes/icoo_proxy/frontend/src/views/ProvidersView.vue:1)
- [SettingsView.vue](E:/codes/icoo_proxy/frontend/src/views/SettingsView.vue:1)

### 12.2 命名建议

建议逐步替换以下命名：

- `--vscode-*` -> `--ui-*`
- `app-page` -> `workspace-page`
- `page-surface` -> `panel-surface`
- `nav-item.active::before top indicator` -> `left rail indicator`

### 12.3 Theme Store 建议

当前主题 store 支持多彩主题，见 [theme.js](E:/codes/icoo_proxy/frontend/src/stores/theme.js:1)。  
对于 Windows 亮色规范，建议收敛为：

- 默认主题: Windows Light
- 可选主题: Slate Light, Windows Dark
- 品牌色仅保留 2 到 3 种受控方案

不建议继续保留大量高饱和换肤色板作为核心体验。

---

## 13. 页面改版优先级

### P0

- 重构全局 token
- 重构标题栏、导航栏、状态栏
- 重构按钮、输入框、面板、状态徽标

### P1

- 网关页改为“摘要 + 控制 + 鉴权 + 调试示例”结构
- 供应商页从卡片视图切换到表格管理视图
- 设置页改为标准设置组布局

### P2

- 日志页表格与详情联动
- 对话规则页双栏编辑器
- 高对比模式与深色模式补充规范

---

## 14. 验收标准

改版完成后，至少应满足以下标准：

- 用户首次看到界面时，明显感知为 Windows 桌面工具，而非 VS Code 风格壳子
- 全局仅有一套统一的颜色、圆角、阴影和控件状态规范
- 页面中 80% 以上表单、按钮、标签、面板采用统一组件样式
- 供应商页和日志页具备更高信息密度和更高操作效率
- 焦点态、错误态、空状态、加载态具备统一视觉定义
- 亮色主题下长时间使用不刺眼，信息可读性优于当前版本

---

## 15. 一句话设计指令

如果后续要基于这份规范继续出界面稿或改代码，可以统一使用这句设计指令：

> 将 `icoo_proxy` 设计为一款面向 Windows PC 的亮色本地 AI 网关管理工具，整体风格参考 Windows 11 桌面应用的清爽秩序感，采用浅灰白背景、克制蓝色强调、细描边、轻阴影、标准化表单和高信息密度管理界面，避免 VS Code 式 IDE 视觉语言和过度 Web 化卡片风格。

