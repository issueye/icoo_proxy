# 供应商模型映射功能实现计划

## 功能概述
为供应商添加模型路由映射功能，支持在单个供应商级别配置模型名称映射规则。
例如：客户端请求 `gpt-4` 时，实际调用供应商的 `claude-3-sonnet`。

## 使用场景
1. **模型替换**：用某个模型替代另一个模型（如用 Claude 替代 GPT）
2. **模型统一**：将不同供应商的模型名称统一为标准名称
3. **版本管理**：将旧版模型名称映射到新版（如 gpt-4 -> gpt-4o）

## 数据设计

### 扩展 ModelEntry 结构
```go
type ModelEntry struct {
    Model  string `json:"model" toml:"model"`   // 对外暴露的模型名称
    Target string `json:"target" toml:"target"` // 实际调用的目标模型
}
```

**配置示例** (TOML):
```toml
[[providers]]
id = "anthropic-1"
name = "Anthropic"
type = "anthropic"

[[providers.llms]]
model = "gpt-4"        # 客户端请求这个
target = "claude-3-sonnet-20240229"  # 实际调用这个

[[providers.llms]]
model = "gpt-3.5-turbo"
target = "claude-3-haiku-20240307"
```

## 实现步骤

### 第 1 步：修改数据结构

#### 1.1 修改 `internal/config/types.go`
将 `ModelEntry` 的 `Alias` 字段改为 `Target`：
```go
type ModelEntry struct {
    Model  string `json:"model" toml:"model"`   // 对外暴露的模型名称
    Target string `json:"target" toml:"target"` // 实际调用的目标模型
}
```

#### 1.2 更新前端表单
- 将"别名"输入框改为"目标模型"输入框
- 更新 placeholder 提示文本

### 第 2 步：实现模型映射解析

#### 2.1 在 `Manager` 添加映射解析方法

```go
// ResolveModel maps a requested model name to the actual target model.
func (m *Manager) ResolveModel(providerID, requestedModel string) string {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    p, exists := m.providers[providerID]
    if !exists {
        return requestedModel // 供应商不存在，返回原始名称
    }
    
    // 遍历 LLMs 查找匹配
    for _, entry := range p.Config.LLMs {
        if entry.Model == requestedModel {
            // 如果配置了 target，返回 target；否则返回 model
            if entry.Target != "" {
                return entry.Target
            }
            return entry.Model
        }
    }
    
    // 未找到映射，返回原始请求
    return requestedModel
}
```

### 第 3 步：集成到请求处理流程

#### 3.1 修改 Gateway Handler

在 `handler.go` 中，解析请求后、发送请求前进行模型映射：

```go
// 在 ChatCompletions 方法中
func (h *Handler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
    // ... 现有代码：解析请求体 ...
    // ... 现有代码：ResolveProvider(model) ...
    
    if p == nil {
        // ... 错误处理 ...
    }
    
    // === 新增：模型映射解析 ===
    actualModel := provider.GetManager().ResolveModel(p.Config.ID, model)
    
    // ... 修改 internalReq.Model = actualModel ...
    
    // 继续后续处理
}
```

**关键点**：
- 在 `ResolveProvider` 之后调用 `ResolveModel`
- 将映射后的模型名赋值给 `internalReq.Model`
- 后续协议转换和目标 Provider 请求都使用映射后的模型名

### 第 4 步：前端 UI 更新

#### 4.1 修改 `frontend/src/views/ProvidersView.vue`

**更新模型列表表格**：
```vue
<table class="model-table">
  <thead>
    <tr>
      <th>请求模型名称</th>
      <th>目标模型（实际调用）</th>
      <th class="action-col">操作</th>
    </tr>
  </thead>
  <tbody>
    <tr v-for="(model, index) in modelForm.llms" :key="index">
      <td>
        <input
          v-model="model.model"
          class="model-input"
          placeholder="如: gpt-4"
        />
      </td>
      <td>
        <input
          v-model="model.target"
          class="model-input"
          placeholder="如: claude-3-sonnet"
        />
      </td>
      <td class="action-col">
        <button @click="removeModel(index)">
          <Trash2 :size="14" />
        </button>
      </td>
    </tr>
  </tbody>
</table>
```

**更新提示文本**：
```vue
<div class="model-dialog-header">
  <span class="hint-text">
    配置模型映射规则：当客户端请求左侧的模型名称时，实际调用右侧的目标模型
  </span>
</div>
```

**更新默认模型选择器**：
```vue
<select v-model="modelForm.defaultModel" class="form-input">
  <option value="">无</option>
  <option
    v-for="m in modelForm.llms.filter(m => m.model)"
    :key="m.model"
    :value="m.model"
  >
    {{ m.model }} → {{ m.target || m.model }}
  </option>
</select>
```

### 第 5 步：向后兼容

#### 5.1 兼容旧的 `alias` 字段
在加载配置时，如果存在 `alias` 但没有 `target`，自动迁移：

```go
// 在 LoadFromConfig 中添加迁移逻辑
for i, entry := range p.Config.LLMs {
    if entry.Alias != "" && entry.Target == "" {
        p.Config.LLMs[i].Target = entry.Alias
        p.Config.LLMs[i].Alias = "" // 清空旧字段
    }
}
```

### 第 6 步：测试验证

1. 添加供应商并配置模型映射
2. 请求映射的模型名称
3. 验证实际调用的是目标模型
4. 验证协议转换使用正确的模型名
5. 测试未配置映射的模型正常透传

## 文件修改清单

### 后端
1. `internal/config/types.go` - 修改 ModelEntry 结构
2. `internal/provider/manager.go` - 添加 ResolveModel 方法
3. `internal/gateway/handler.go` - 集成模型映射到请求流程

### 前端
4. `frontend/src/views/ProvidersView.vue` - 更新模型管理 UI

## 注意事项

1. **性能**：ResolveModel 使用 RLock，不影响并发性能
2. **容错**：未配置映射的模型正常透传
3. **优先级**：模型映射在 ResolveProvider 之后执行，不影响供应商选择
4. **可观测性**：建议在日志中记录模型映射行为（可选）

## 扩展功能（未来）

1. **全局映射规则**：跨供应商的全局模型映射
2. **映射日志**：记录模型映射的调用日志
3. **映射验证**：检查目标模型在供应商中是否存在
4. **批量导入**：从 CSV/JSON 批量导入映射规则
