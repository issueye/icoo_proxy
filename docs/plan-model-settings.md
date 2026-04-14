# 供应商模型设置功能实现计划

## 功能概述
为 Wails 版本的供应商管理添加完整的模型设置功能，包括：
- 模型列表管理（添加/删除模型）
- 模型别名设置
- 默认模型配置

## 实现步骤

### 第 1 步：扩展后端数据结构

#### 1.1 修改 `internal/config/types.go`
在 `ProviderConfig` 结构体中添加：
```go
type ProviderConfig struct {
    // ... 现有字段 ...
    LLMs        []ModelEntry `json:"llms" toml:"llms"`           // 模型列表
    DefaultModel string      `json:"defaultModel" toml:"default_model"` // 默认模型
}

// 新增：模型条目
type ModelEntry struct {
    Model string `json:"model" toml:"model"`       // 模型名称
    Alias string `json:"alias" toml:"alias"`       // 模型别名
}
```

#### 1.2 修改 `internal/provider/manager.go`
更新 `ProviderListJSON` 函数，包含模型信息：
```go
type providerInfo struct {
    // ... 现有字段 ...
    LLMs         []config.ModelEntry `json:"llms,omitempty"`
    DefaultModel string              `json:"defaultModel,omitempty"`
}
```

### 第 2 步：扩展 Wails API

#### 2.1 修改 `internal/services/app.go`
添加新的 API 方法：
```go
// 获取供应商的模型列表
func (a *App) GetProviderModels(providerID string) string

// 设置供应商的模型列表
func (a *App) SetProviderModels(providerID string, llms []config.ModelEntry, defaultModel string) error

// 测试模型连接（可选）
func (a *App) TestModel(providerID, modelName string) string
```

#### 2.2 修改 `internal/provider/manager.go`
添加模型管理方法：
```go
func (m *Manager) GetModels(providerID string) ([]config.ModelEntry, string, error)
func (m *Manager) SetModels(providerID string, llms []config.ModelEntry, defaultModel string) error
```

### 第 3 步：扩展 Pinia Store

#### 3.1 修改 `frontend/src/stores/provider.js`
添加模型管理方法：
```javascript
async function getProviderModels(providerId) {
  const result = await window.go.services.App.GetProviderModels(providerId);
  return JSON.parse(result);
}

async function setProviderModels(providerId, llms, defaultModel) {
  await window.go.services.App.SetProviderModels(providerId, llms, defaultModel);
  await fetchProviders();
}
```

### 第 4 步：前端 UI 实现

#### 4.1 修改 `frontend/src/views/ProvidersView.vue`

**添加模型管理按钮**：
- 在供应商卡片的操作按钮区添加"模型"按钮（使用 `Database` 图标）

**添加模型管理对话框**：
```vue
<ModalDialog
  :visible="showModelDialog"
  :title="`模型设置 - ${currentProvider.name}`"
  size="xl"
  @close="showModelDialog = false"
>
  <!-- 模型列表表格 -->
  <div class="model-list">
    <table>
      <thead>
        <tr>
          <th>模型名称</th>
          <th>别名</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(model, index) in modelForm.llms" :key="index">
          <td>
            <input v-model="model.model" placeholder="如: gpt-4" />
          </td>
          <td>
            <input v-model="model.alias" placeholder="可选别名" />
          </td>
          <td>
            <button @click="removeModel(index)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>
    <button @click="addModel">+ 添加模型</button>
  </div>

  <!-- 默认模型设置 -->
  <div class="default-model-section">
    <label>默认模型</label>
    <select v-model="modelForm.defaultModel">
      <option value="">无</option>
      <option v-for="m in modelForm.llms" :key="m.model" :value="m.model">
        {{ m.alias || m.model }}
      </option>
    </select>
  </div>

  <!-- 操作按钮 -->
  <div class="dialog-actions">
    <button @click="showModelDialog = false">取消</button>
    <button @click="saveModels">保存</button>
  </div>
</ModalDialog>
```

**添加响应式状态**：
```javascript
const showModelDialog = ref(false);
const currentProvider = ref({});
const modelForm = ref({
  llms: [],
  defaultModel: ''
});
```

**添加方法**：
```javascript
function openModelDialog(provider) {
  currentProvider.value = provider;
  modelForm.value = {
    llms: provider.llms ? [...provider.llms] : [],
    defaultModel: provider.defaultModel || ''
  };
  showModelDialog.value = true;
}

function addModel() {
  modelForm.value.llms.push({ model: '', alias: '' });
}

function removeModel(index) {
  modelForm.value.llms.splice(index, 1);
}

async function saveModels() {
  // 过滤空模型
  const validLlms = modelForm.value.llms.filter(m => m.model);
  await providerStore.setProviderModels(
    currentProvider.value.id,
    validLlms,
    modelForm.value.defaultModel
  );
  showModelDialog.value = false;
  toast('模型设置已保存', 'success');
}
```

### 第 5 步：样式实现

添加模型管理相关样式：
```css
.model-list {
  margin: 16px 0;
}

.model-list table {
  width: 100%;
  border-collapse: collapse;
}

.model-list th,
.model-list td {
  padding: 8px;
  border: 1px solid var(--color-border);
}

.model-list input {
  width: 100%;
  padding: 6px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
}

.default-model-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border);
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
```

## 测试验证
1. 添加供应商后，点击"模型"按钮打开模型设置
2. 添加多个模型及其别名
3. 设置默认模型
4. 保存后验证数据持久化
5. 重新打开编辑对话框，验证数据正确加载

## 文件修改清单
1. `internal/config/types.go` - 添加 ModelEntry 和扩展 ProviderConfig
2. `internal/provider/manager.go` - 添加模型管理方法和 JSON 序列化
3. `internal/services/app.go` - 添加 Wails API 方法
4. `frontend/src/stores/provider.js` - 添加模型管理方法
5. `frontend/src/views/ProvidersView.vue` - 添加模型管理 UI 和逻辑
