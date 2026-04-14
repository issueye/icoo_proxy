<template>
    <ManagePage
        title="技能管理"
        description="管理用户技能与内置技能，扩展 AI 助手能力边界。"
        :icon="SparklesIcon"
        :columns="columns"
        :data="filteredSkills"
        :loading="false"
        :metrics="metrics"
        :filters="[]"
        :primary-action="null"
        @search="handleSearch"
        @filter-change="() => {}"
        @action="() => {}"
        @refresh="() => skillStore.fetchSkills()"
    >
        <template #actions>
            <div class="flex items-center gap-2">
                <Button @click="showInstallDialog = true" variant="secondary" size="sm">
                    <PackageIcon :size="14" />
                    安装
                </Button>
                <Button @click="showImportDialog = true" variant="secondary" size="sm">
                    <UploadIcon :size="14" />
                    导入
                </Button>
                <Button @click="handleExport" variant="secondary" size="sm">
                    <DownloadIcon :size="14" />
                    导出
                </Button>
                <Button @click="openAddDialog" variant="default" size="sm">
                    <PlusIcon :size="14" />
                    添加技能
                </Button>
            </div>
        </template>

        <!-- 批量操作栏 -->
        <div
            v-if="selectedSkills.length > 0"
            class="flex items-center justify-between p-2 bg-accent/10 border border-accent/20 rounded-md mt-3"
        >
            <span class="text-sm">
                已选择
                <strong>{{ selectedSkills.length }}</strong> 个技能
            </span>
            <div class="flex gap-2">
                <button
                    @click="batchEnable(true)"
                    class="btn btn-primary text-sm"
                >
                    启用
                </button>
                <button
                    @click="batchEnable(false)"
                    class="btn btn-secondary text-sm"
                >
                    禁用
                </button>
                <button
                    @click="batchDelete"
                    class="btn btn-danger text-sm"
                >
                    删除
                </button>
                <button
                    @click="selectedSkills = []"
                    class="px-3 py-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
                >
                    取消
                </button>
            </div>
        </div>

        <!-- 标签云 -->
        <div v-if="skillStore.tags.length > 0" class="flex flex-wrap gap-2 mt-3">
            <button
                v-for="tag in skillStore.tags"
                :key="tag"
                @click="toggleTagFilter(tag)"
                :class="[
                    'px-3 py-1 text-xs rounded-full transition-colors',
                    selectedTags.includes(tag)
                        ? 'bg-accent text-white'
                        : 'bg-secondary text-muted-foreground hover:bg-secondary',
                ]"
            >
                {{ tag }}
            </button>
        </div>

        <!-- 技能列表 -->
        <div class="overflow-y-auto space-y-3 pr-1 mt-3 flex-1">
            <!-- 用户技能 -->
            <section v-if="filteredUserSkills.length > 0">
                <h3 class="text-sm font-medium text-muted-foreground mb-2">
                    用户技能 ({{ filteredUserSkills.length }})
                </h3>
                <div class="space-y-3">
                    <SkillCard
                        v-for="skill in filteredUserSkills"
                        :key="skill.id"
                        :skill="skill"
                        :selected="selectedSkills.includes(skill.id)"
                        @toggle-select="toggleSelect(skill.id)"
                        @edit="openEditDialog(skill)"
                        @delete="handleDelete(skill)"
                        @toggle-enabled="toggleSkill(skill)"
                    />
                </div>
            </section>

            <!-- 内置技能 -->
            <section v-if="filteredBuiltinSkills.length > 0">
                <h3 class="text-sm font-medium text-muted-foreground mb-2">
                    内置技能 ({{ filteredBuiltinSkills.length }})
                </h3>
                <div class="space-y-3">
                    <SkillCard
                        v-for="skill in filteredBuiltinSkills"
                        :key="skill.id"
                        :skill="skill"
                        :selected="selectedSkills.includes(skill.id)"
                        @toggle-select="toggleSelect(skill.id)"
                        @edit="openEditDialog(skill)"
                        @delete="handleDelete(skill)"
                        @toggle-enabled="toggleSkill(skill)"
                    />
                </div>
            </section>

            <!-- 空状态 -->
            <div v-if="filteredSkills.length === 0" class="text-center py-12">
                <div class="w-12 h-12 mx-auto mb-3 rounded-md bg-secondary flex items-center justify-center">
                    <SparklesIcon :size="24" class="text-accent" />
                </div>
                <h3 class="text-sm font-medium mb-1">
                    {{ searchKeyword ? "没有找到匹配的技能" : "暂无技能" }}
                </h3>
                <p class="text-muted-foreground text-xs">
                    {{ searchKeyword ? "试试其他搜索条件" : "添加自定义技能来扩展 AI 助手的能力" }}
                </p>
            </div>
        </div>

        <!-- 技能对话框 -->
        <SkillDialog
            v-model:visible="dialogVisible"
            :skill="editingSkill"
            @submit="handleSubmit"
        />

        <!-- 导入对话框 -->
        <ImportDialog
            v-model:visible="showImportDialog"
            @import="handleImport"
        />

        <ModalDialog
            v-model:visible="showInstallDialog"
            title="安装技能"
            confirm-text="安装"
            loading-text="安装中..."
            :loading="installing"
            :confirm-disabled="!installForm.slug.trim()"
            @confirm="handleInstall"
        >
            <div class="space-y-4">
                <div>
                    <label class="block text-sm text-muted-foreground mb-2">技能 Slug</label>
                    <Input
                        v-model="installForm.slug"
                        placeholder="例如: github、docker-compose"
                    />
                    <p class="text-xs text-muted-foreground mt-1">填写 registry 中的唯一技能标识。</p>
                </div>
                <div>
                    <label class="block text-sm text-muted-foreground mb-2">版本号</label>
                    <Input
                        v-model="installForm.version"
                        placeholder="留空则安装最新版本"
                    />
                </div>
            </div>
        </ModalDialog>
    </ManagePage>
</template>

<script setup>
import { ref, computed, watch, onMounted } from "vue";
import {
    Plus as PlusIcon,
    Search as SearchIcon,
    Upload as UploadIcon,
    Download as DownloadIcon,
    Sparkles as SparklesIcon,
    CheckCircle as CheckCircleIcon,
    Package as PackageIcon,
    Tags as TagsIcon,
} from "lucide-vue-next";
import { useSkillStore } from "@/stores/skill";
import SkillCard from "@/components/skills/SkillCard.vue";
import SkillDialog from "@/components/skills/SkillDialog.vue";
import ImportDialog from "@/components/skills/ImportDialog.vue";
import ModalDialog from "@/components/ModalDialog.vue";
import { ManagePage } from "@/components/layout";
import { Button, IconButton, Badge, Input, SearchInput } from "@/components/ui";
import { useToast } from "@/composables/useToast.js";
import { useConfirm } from "@/composables/useConfirm.js";

const skillStore = useSkillStore();
const { toast } = useToast();
const { confirm } = useConfirm();

const searchKeyword = ref("");
const selectedTags = ref([]);
const selectedSkills = ref([]);
const dialogVisible = ref(false);
const showImportDialog = ref(false);
const showInstallDialog = ref(false);
const editingSkill = ref(null);
const installing = ref(false);
const installForm = ref({
    slug: "",
    version: "",
});

const columns = [];

const metrics = computed(() => [
    {
        icon: SparklesIcon,
        iconColor: "text-accent",
        iconBg: "bg-accent/10",
        value: skillStore.skills.length,
        label: "技能总数",
    },
    {
        icon: CheckCircleIcon,
        iconColor: "text-green-500",
        iconBg: "bg-green-500/10",
        value: enabledSkillCount.value,
        label: "已启用技能",
    },
    {
        icon: PackageIcon,
        iconColor: "text-sky-500",
        iconBg: "bg-sky-500/10",
        value: filteredUserSkills.value.length,
        label: "用户技能",
    },
    {
        icon: TagsIcon,
        iconColor: "text-amber-500",
        iconBg: "bg-amber-500/10",
        value: skillStore.tags.length,
        label: "标签总数",
    },
]);

function handleSearch(value) {
    searchKeyword.value = value;
}

const enabledSkillCount = computed(() =>
    skillStore.skills.filter((skill) => skill.enabled !== false).length,
);

const filteredSkills = computed(() => {
    let result = skillStore.skills;
    if (searchKeyword.value) {
        const keyword = searchKeyword.value.toLowerCase();
        result = result.filter(
            (s) =>
                s.name?.toLowerCase().includes(keyword) ||
                s.description?.toLowerCase().includes(keyword) ||
                s.tags?.some((t) => t.toLowerCase().includes(keyword)),
        );
    }
    if (selectedTags.value.length > 0) {
        result = result.filter((s) =>
            selectedTags.value.some((tag) => s.tags?.includes(tag)),
        );
    }
    return result;
});

const filteredUserSkills = computed(() =>
    filteredSkills.value.filter((s) => s.source !== "builtin"),
);

const filteredBuiltinSkills = computed(() =>
    filteredSkills.value.filter((s) => s.source === "builtin"),
);

function openAddDialog() {
    editingSkill.value = null;
    dialogVisible.value = true;
}

function openEditDialog(skill) {
    editingSkill.value = skill;
    dialogVisible.value = true;
}

function closeDialog() {
    dialogVisible.value = false;
    editingSkill.value = null;
}

async function handleSubmit(formData) {
    try {
        if (editingSkill.value) {
            await skillStore.updateSkill({ id: editingSkill.value.id, ...formData });
        } else {
            await skillStore.createSkill(formData);
        }
        closeDialog();
    } catch (error) {
        console.error("保存技能失败:", error);
        toast("保存技能失败: " + error.message, "error");
    }
}

async function handleDelete(skill) {
    const ok = await confirm(`确定要删除技能 "${skill.name}" 吗？`);
    if (!ok) return;
    try {
        await skillStore.deleteSkill(skill.id);
    } catch (error) {
        console.error("删除技能失败:", error);
        toast("删除技能失败: " + error.message, "error");
    }
}

async function toggleSkill(skill) {
    try {
        await skillStore.toggleSkill(skill.id);
    } catch (error) {
        console.error("更新技能失败:", error);
    }
}

function toggleSelect(id) {
    const idx = selectedSkills.value.indexOf(id);
    if (idx > -1) selectedSkills.value.splice(idx, 1);
    else selectedSkills.value.push(id);
}

function toggleTagFilter(tag) {
    const idx = selectedTags.value.indexOf(tag);
    if (idx > -1) selectedTags.value.splice(idx, 1);
    else selectedTags.value.push(tag);
}

async function batchEnable(enabled) {
    try {
        await skillStore.batchUpdateEnabled(selectedSkills.value, enabled);
        selectedSkills.value = [];
    } catch (error) {
        console.error("批量操作失败:", error);
        toast("批量操作失败: " + error.message, "error");
    }
}

async function batchDelete() {
    const ok = await confirm(`确定要删除选中的 ${selectedSkills.value.length} 个技能吗？`);
    if (!ok) return;
    try {
        await skillStore.batchDeleteSkills(selectedSkills.value);
        selectedSkills.value = [];
    } catch (error) {
        console.error("批量删除失败:", error);
        toast("批量删除失败: " + error.message, "error");
    }
}

async function handleExport() {
    try {
        await skillStore.exportSkills();
    } catch (error) {
        console.error("导出失败:", error);
        toast("导出失败: " + error.message, "error");
    }
}

async function handleImport(file, overwrite) {
    try {
        const result = await skillStore.importSkills(file, overwrite);
        showImportDialog.value = false;
        toast(`导入成功: ${result.success} 个, 跳过: ${result.skip} 个`, "success");
    } catch (error) {
        console.error("导入失败:", error);
        toast("导入失败: " + error.message, "error");
    }
}

async function handleInstall() {
    if (!installForm.value.slug.trim()) {
        toast("请填写技能 slug", "warning");
        return;
    }

    installing.value = true;
    try {
        await skillStore.installSkill(
            installForm.value.slug.trim(),
            installForm.value.version.trim(),
        );
        showInstallDialog.value = false;
        installForm.value = { slug: "", version: "" };
        toast("技能安装成功", "success");
    } catch (error) {
        console.error("安装技能失败:", error);
        toast("安装技能失败: " + error.message, "error");
    } finally {
        installing.value = false;
    }
}

watch(searchKeyword, () => { selectedSkills.value = []; });

onMounted(() => {
    skillStore.fetchSkills();
});
</script>

