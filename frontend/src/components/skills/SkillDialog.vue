<template>
    <ModalDialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :title="skill ? '编辑技能' : '添加技能'"
        size="lg"
        :scrollable="true"
        :confirm-text="skill ? '保存' : '添加'"
        :confirm-disabled="!isValid"
        @confirm="handleSubmit"
    >
        <div class="dialog-body">
            <section class="dialog-section">
                <h4 class="dialog-section-title">基本信息</h4>

                <Input
                    v-model="formData.name"
                    label="技能名称"
                    placeholder="例如: code_review"
                    :disabled="!!skill"
                    description="技能名称只能包含字母、数字、下划线"
                />

                <Input
                    v-model="formData.description"
                    label="描述"
                    placeholder="技能功能简述..."
                />

                <div class="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
                    <Input
                        v-model="formData.version"
                        label="版本号"
                        placeholder="1.0.0"
                    />
                    <Input
                        :model-value="formData.source || 'workspace'"
                        label="来源"
                        placeholder="workspace"
                        disabled
                    />
                </div>
            </section>

            <section class="dialog-section">
                <h4 class="dialog-section-title">技能内容</h4>

                <Textarea
                    v-model="formData.content"
                    label="技能内容 (Markdown)"
                    placeholder="## 技能名称&#10;&#10;你是一个...&#10;&#10;## 可用工具&#10;- search_web&#10;- calculator"
                    rows="10"
                    class="font-mono text-sm"
                />

                <Textarea
                    v-model="formData.prompt"
                    label="系统提示词"
                    placeholder="额外的系统级提示词，用于增强或覆盖技能行为..."
                    rows="3"
                    class="font-mono text-sm"
                />
            </section>

            <section class="dialog-section">
                <h4 class="dialog-section-title">标签与工具</h4>

                <div>
                    <label class="block text-sm text-muted-foreground mb-2">标签</label>
                    <div class="flex gap-2 mb-2">
                        <Input
                            v-model="newTag"
                            @keydown="handleTagKeydown"
                            placeholder="输入标签后按回车添加"
                            class="flex-1"
                        />
                        <Button @click="addTag" type="button" variant="secondary" size="sm">
                            <PlusIcon :size="13" />
                            添加
                        </Button>
                    </div>
                    <div v-if="formData.tags.length > 0" class="flex flex-wrap gap-1.5">
                        <span
                            v-for="(tag, idx) in formData.tags"
                            :key="tag"
                            class="inline-flex items-center gap-1 rounded-md bg-primary/10 px-2.5 py-1 text-[12px] text-primary"
                        >
                            {{ tag }}
                            <button @click="removeTag(idx)" type="button" class="transition-colors hover:text-red-500">
                                <XIcon :size="11" />
                            </button>
                        </span>
                    </div>
                    <p v-else class="dialog-helper">暂无标签</p>
                </div>

                <div>
                    <label class="block text-sm text-muted-foreground mb-2">关联工具</label>
                    <div class="flex gap-2 mb-2">
                        <Input
                            v-model="newTool"
                            @keydown="handleToolKeydown"
                            placeholder="输入工具名称后按回车添加"
                            class="flex-1"
                        />
                        <Button @click="addTool" type="button" variant="secondary" size="sm">
                            <PlusIcon :size="13" />
                            添加
                        </Button>
                    </div>
                    <div v-if="formData.tools.length > 0" class="flex flex-wrap gap-1.5">
                        <span
                            v-for="(tool, idx) in formData.tools"
                            :key="tool"
                            class="inline-flex items-center gap-1 rounded-md bg-amber-500/10 px-2.5 py-1 text-[12px] text-amber-500"
                        >
                            <WrenchIcon :size="10" />
                            {{ tool }}
                            <button @click="removeTool(idx)" type="button" class="transition-colors hover:text-red-500">
                                <XIcon :size="11" />
                            </button>
                        </span>
                    </div>
                    <p v-else class="dialog-helper">暂无关联工具</p>
                </div>
            </section>

            <section class="dialog-section">
                <h4 class="dialog-section-title">选项</h4>

                <div class="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
                    <label class="flex items-start gap-3 rounded-md border border-border bg-background/70 p-3 cursor-pointer">
                        <input
                            v-model="formData.always_load"
                            type="checkbox"
                            class="mt-0.5 h-4 w-4 rounded border-border bg-background text-primary focus:ring-primary"
                        />
                        <div>
                            <span class="text-sm text-foreground">始终加载</span>
                            <p class="dialog-helper mt-1">启用后此技能将始终处于活跃状态</p>
                        </div>
                    </label>
                    <label class="flex items-start gap-3 rounded-md border border-border bg-background/70 p-3 cursor-pointer">
                        <input
                            v-model="formData.enabled"
                            type="checkbox"
                            class="mt-0.5 h-4 w-4 rounded border-border bg-background text-primary focus:ring-primary"
                        />
                        <div>
                            <span class="text-sm text-foreground">启用技能</span>
                            <p class="dialog-helper mt-1">禁用的技能不会被 AI 调用</p>
                        </div>
                    </label>
                </div>
            </section>
        </div>
    </ModalDialog>
</template>

<script setup>
import { ref, computed, watch } from "vue";
import { Plus as PlusIcon, X as XIcon, Wrench as WrenchIcon } from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";
import { Input, Textarea, Button } from "@/components/ui";

const props = defineProps({
    visible: { type: Boolean, default: false },
    skill: { type: Object, default: null },
});

const emit = defineEmits(["update:visible", "submit"]);

const formData = ref({
    name: "",
    description: "",
    content: "",
    prompt: "",
    tags: [],
    tools: [],
    always_load: false,
    enabled: true,
    version: "1.0.0",
    source: "workspace",
});

const newTag = ref("");
const newTool = ref("");

const isValid = computed(() => {
    return formData.value.name.trim() && formData.value.content.trim();
});

function addTag() {
    const tag = newTag.value.trim();
    if (tag && !formData.value.tags.includes(tag)) {
        formData.value.tags.push(tag);
    }
    newTag.value = "";
}

function removeTag(index) {
    formData.value.tags.splice(index, 1);
}

function handleTagKeydown(event) {
    if (event.key === "Enter") {
        event.preventDefault();
        addTag();
    }
}

function addTool() {
    const tool = newTool.value.trim();
    if (tool && !formData.value.tools.includes(tool)) {
        formData.value.tools.push(tool);
    }
    newTool.value = "";
}

function removeTool(index) {
    formData.value.tools.splice(index, 1);
}

function handleToolKeydown(event) {
    if (event.key === "Enter") {
        event.preventDefault();
        addTool();
    }
}

function handleSubmit() {
    if (!isValid.value) return;
    emit("submit", {
        name: formData.value.name.trim(),
        description: formData.value.description.trim(),
        content: formData.value.content.trim(),
        prompt: formData.value.prompt.trim(),
        tags: formData.value.tags,
        tools: formData.value.tools,
        always_load: formData.value.always_load,
        enabled: formData.value.enabled,
        version: formData.value.version,
        source: formData.value.source,
    });
}

function resetForm() {
    formData.value = {
        name: "", description: "", content: "", prompt: "",
        tags: [], tools: [], always_load: false, enabled: true,
        version: "1.0.0", source: "workspace",
    };
    newTag.value = "";
    newTool.value = "";
}

function fillForm(skill) {
    if (skill) {
        formData.value = {
            name: skill.name || "",
            description: skill.description || "",
            content: skill.content || "",
            prompt: skill.prompt || "",
            tags: skill.tags || [],
            tools: skill.tools || [],
            always_load: skill.always_load || false,
            enabled: skill.enabled !== false,
            version: skill.version || "1.0.0",
            source: skill.source || "workspace",
        };
    } else {
        resetForm();
    }
}

watch(() => props.skill, (newSkill) => { fillForm(newSkill); }, { immediate: true });
watch(() => props.visible, (visible) => { if (!visible) resetForm(); });
</script>
