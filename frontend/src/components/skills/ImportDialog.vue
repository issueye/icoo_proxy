<template>
    <ModalDialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        title="导入技能"
        confirm-text="导入"
        :confirm-disabled="!selectedFile"
        @confirm="handleImport"
    >
        <div class="dialog-body">
            <section class="dialog-section">
                <h4 class="dialog-section-title">文件选择</h4>
                <div
                    @click="!selectedFile && $refs.fileInput.click()"
                    @drop.prevent="handleDrop"
                    @dragover.prevent="isDragging = true"
                    @dragleave.prevent="isDragging = false"
                    :class="[
                        'cursor-pointer rounded-md border-2 border-dashed p-8 text-center transition-all',
                        isDragging
                            ? 'border-primary bg-primary/10 scale-[1.01]'
                            : selectedFile
                                ? 'cursor-default border-green-500/40 bg-green-500/5'
                                : 'border-border hover:border-primary/40 hover:bg-background/70'
                    ]"
                >
                    <input
                        ref="fileInput"
                        type="file"
                        accept=".zip"
                        class="hidden"
                        @change="handleFileSelect"
                    />

                    <!-- 已选择文件 -->
                    <div v-if="selectedFile" class="space-y-2">
                        <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-md bg-green-500/10">
                            <FileTextIcon :size="22" class="text-green-500" />
                        </div>
                        <p class="text-sm font-medium text-green-500">{{ selectedFile.name }}</p>
                        <p class="dialog-helper">{{ formatFileSize(selectedFile.size) }}</p>
                        <button @click.stop="selectedFile = null; $refs.fileInput.value = ''"
                            class="mx-auto mt-2 flex items-center gap-1 text-[11px] text-muted-foreground transition-colors hover:text-red-500">
                            <XIcon :size="11" />
                            移除文件
                        </button>
                    </div>

                    <!-- 未选择 -->
                    <div v-else class="space-y-2">
                        <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-md bg-secondary">
                            <UploadIcon :size="22" class="text-muted-foreground" />
                        </div>
                        <p class="text-sm text-foreground">点击选择文件或拖拽到此处</p>
                        <p class="dialog-helper">支持 .zip 格式的技能包，压缩包内需包含一个或多个 `SKILL.md`</p>
                    </div>
                </div>
            </section>

            <section v-if="selectedFile" class="dialog-section">
                <h4 class="dialog-section-title">导入预览</h4>
                <div class="flex items-center justify-between mb-1">
                    <span class="text-xs font-medium text-muted-foreground">压缩包已就绪</span>
                    <span class="dialog-helper">导入时会自动扫描其中的 `SKILL.md` 文件</span>
                </div>
            </section>

            <section class="dialog-section">
                <h4 class="dialog-section-title">导入选项</h4>

                <label class="flex cursor-pointer items-start gap-3 rounded-md border border-border bg-background/70 p-3">
                    <input
                        v-model="overwrite"
                        type="checkbox"
                        class="mt-0.5 h-4 w-4 rounded border-border bg-background text-primary focus:ring-primary"
                    />
                    <div>
                        <span class="text-sm text-foreground">覆盖已存在的技能</span>
                        <p class="dialog-helper mt-0.5">勾选后，同名技能将被新导入的内容覆盖，否则跳过</p>
                    </div>
                </label>
            </section>

            <section class="rounded-md border border-blue-500/20 bg-blue-500/5 p-3">
                <div class="flex items-start gap-2">
                    <InfoIcon :size="14" class="text-blue-400 mt-0.5 flex-shrink-0" />
                    <div class="space-y-1 text-[11px] text-muted-foreground">
                        <p class="font-medium text-blue-400">导入说明</p>
                        <p>• 支持从 zip 技能包批量导入技能</p>
                        <p>• zip 内每个技能目录都需要包含 `SKILL.md`</p>
                        <p>• 内置技能（builtin）不会被覆盖或删除</p>
                        <p>• 建议导入前备份现有配置</p>
                    </div>
                </div>
            </section>
        </div>
    </ModalDialog>
</template>

<script setup>
import { ref } from "vue";
import {
    Upload as UploadIcon,
    FileText as FileTextIcon,
    X as XIcon,
    Info as InfoIcon,
} from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";

defineProps({ visible: { type: Boolean, default: false } });
const emit = defineEmits(['update:visible', 'import']);

const selectedFile = ref(null);
const overwrite = ref(false);
const isDragging = ref(false);

function formatFileSize(bytes) {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function selectFile(file) {
    if (!file) return;
    selectedFile.value = file;
}

function handleFileSelect(event) {
    const file = event.target.files?.[0];
    if (file && file.name.toLowerCase().endsWith(".zip")) selectFile(file);
}

function handleDrop(event) {
    isDragging.value = false;
    const file = event.dataTransfer.files?.[0];
    if (file && file.name.toLowerCase().endsWith(".zip")) {
        selectFile(file);
    }
}

function handleImport() {
    if (!selectedFile.value) return;
    emit('import', selectedFile.value, overwrite.value);
    selectedFile.value = null;
    overwrite.value = false;
}
</script>
