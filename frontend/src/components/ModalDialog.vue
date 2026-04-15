<template>
    <!-- 弹窗组件 - 通用对话框封装 -->
    <Transition name="modal">
        <div
            v-if="visible"
            class="fixed inset-0 z-50 flex items-start justify-center px-3 py-6"
        >
            <!-- 遮罩层 -->
            <div
                class="absolute inset-0 modal-mask transition-opacity"
                @click="handleMaskClick"
            />

            <!-- 弹窗内容 -->
            <div
                class="modal-shell relative w-full overflow-hidden flex flex-col max-h-[90vh]"
                :class="[sizeClasses[size], { 'modal-shell--overflow-visible': allowOverflow }]"
            >
                <!-- 头部 -->
                <div
                    v-if="showHeader"
                    class="modal-shell__header"
                    :class="{ 'sticky top-0 z-10': scrollable }"
                >
                    <div class="flex items-center gap-2.5">
                        <!-- 图标插槽 -->
                        <slot name="icon">
                            <div v-if="icon" class="modal-shell__icon">
                                <component :is="icon" :size="13" class="text-accent" />
                            </div>
                        </slot>
                        <h2 class="text-sm font-semibold text-foreground tracking-tight">
                            {{ title }}
                        </h2>
                    </div>
                    <!-- 关闭按钮 -->
                    <button
                        v-if="showClose"
                        @click="handleClose"
                        class="btn btn-ghost btn-icon text-muted-foreground"
                    >
                        <XIcon :size="16" />
                    </button>
                </div>

                <!-- 内容区域 -->
                <div
                    class="modal-shell__body"
                    :class="[contentClass, { 'modal-shell__body--overflow-visible': allowOverflow }]"
                >
                    <slot />
                </div>

                <!-- 底部按钮 -->
                <div
                    v-if="showFooter"
                    class="modal-shell__footer"
                    :class="[
                        footerAlignClass,
                        { 'sticky bottom-0 z-10': scrollable },
                    ]"
                >
                    <slot name="footer">
                        <!-- 取消按钮 -->
                        <button
                            v-if="showCancel"
                            @click="handleCancel"
                            class="btn btn-secondary"
                            :disabled="loading"
                        >
                            {{ cancelText }}
                        </button>
                        <!-- 确认按钮 -->
                        <button
                            v-if="showConfirm"
                            @click="handleConfirm"
                            :disabled="loading || confirmDisabled"
                            class="btn btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            <Loader2Icon
                                v-if="loading"
                                :size="14"
                                class="animate-spin"
                            />
                            {{ loading ? loadingText : confirmText }}
                        </button>
                    </slot>
                </div>
            </div>
        </div>
    </Transition>
</template>

<script setup>
import { XIcon, Loader2Icon } from "lucide-vue-next";

/**
 * 通用弹窗组件
 * @description 封装了遮罩、头部、内容区、底部按钮的通用对话框
 */

const props = defineProps({
    /** 是否显示弹窗 */
    visible: {
        type: Boolean,
        default: false,
    },
    /** 弹窗标题 */
    title: {
        type: String,
        default: "",
    },
    /** 弹窗尺寸: sm(小), md(中), lg(大), xl(超大), full(全屏) */
    size: {
        type: String,
        default: "md",
        validator: (value) => ["sm", "md", "lg", "xl", "full"].includes(value),
    },
    /** 是否显示头部 */
    showHeader: {
        type: Boolean,
        default: true,
    },
    /** 是否显示关闭按钮 */
    showClose: {
        type: Boolean,
        default: true,
    },
    /** 是否显示底部 */
    showFooter: {
        type: Boolean,
        default: true,
    },
    /** 是否显示取消按钮 */
    showCancel: {
        type: Boolean,
        default: true,
    },
    /** 是否显示确认按钮 */
    showConfirm: {
        type: Boolean,
        default: true,
    },
    /** 取消按钮文本 */
    cancelText: {
        type: String,
        default: "取消",
    },
    /** 确认按钮文本 */
    confirmText: {
        type: String,
        default: "确认",
    },
    /** 加载中文本 */
    loadingText: {
        type: String,
        default: "保存中...",
    },
    /** 是否加载中 */
    loading: {
        type: Boolean,
        default: false,
    },
    /** 确认按钮是否禁用 */
    confirmDisabled: {
        type: Boolean,
        default: false,
    },
    /** 点击遮罩是否关闭 */
    maskClosable: {
        type: Boolean,
        default: true,
    },
    /** 内容区域是否可滚动 */
    scrollable: {
        type: Boolean,
        default: false,
    },
    /** 是否允许内容浮层溢出容器 */
    allowOverflow: {
        type: Boolean,
        default: false,
    },
    /** 底部按钮对齐方式: left, center, right */
    footerAlign: {
        type: String,
        default: "right",
        validator: (value) => ["left", "center", "right"].includes(value),
    },
    /** 头部图标组件 */
    icon: {
        type: Object,
        default: null,
    },
    /** 内容区域自定义类名 */
    contentClass: {
        type: String,
        default: "p-4",
    },
});

const emit = defineEmits(["close", "cancel", "confirm", "update:visible"]);

// 尺寸对应的类名
const sizeClasses = {
    sm: "max-w-sm",
    md: "max-w-2xl",
    lg: "max-w-4xl",
    xl: "max-w-5xl",
    full: "max-w-6xl",
};

// 底部对齐类名
const footerAlignClass = {
    left: "justify-start",
    center: "justify-center",
    right: "justify-end",
}[props.footerAlign];

/**
 * 处理关闭事件
 */
function handleClose() {
    emit("close");
    emit("update:visible", false);
}

/**
 * 处理遮罩点击
 */
function handleMaskClick() {
    if (props.maskClosable) {
        handleClose();
    }
}

/**
 * 处理取消事件
 */
function handleCancel() {
    emit("cancel");
    handleClose();
}

/**
 * 处理确认事件
 */
function handleConfirm() {
    emit("confirm");
}
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
    transition: opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
    opacity: 0;
}

.modal-shell {
    border-radius: var(--radius-dialog);
    border: 1px solid var(--ui-border-default);
    background: var(--ui-bg-surface);
    box-shadow: var(--shadow-dialog);
}

.modal-shell--overflow-visible {
    overflow: visible;
}

.modal-mask {
    background: rgba(17, 24, 39, 0.26);
    backdrop-filter: blur(4px);
}

.modal-shell__header,
.modal-shell__footer {
    background: var(--ui-bg-surface-muted);
    border-color: var(--ui-border-default);
}

.modal-shell__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: var(--dialog-header-padding);
    border-bottom: 1px solid var(--ui-border-default);
}

.modal-shell__footer {
    display: flex;
    gap: var(--control-gap);
    padding: var(--dialog-footer-padding);
    border-top: 1px solid var(--ui-border-default);
}

.modal-shell__icon {
    width: 26px;
    height: 26px;
    border-radius: var(--radius-sm);
    background: var(--color-accent-soft);
    border: 1px solid color-mix(in srgb, var(--color-accent) 18%, var(--ui-border-default));
    display: flex;
    align-items: center;
    justify-content: center;
}

.modal-shell__body {
    background: var(--ui-bg-surface);
    flex: 1;
    overflow-y: auto;
    padding: var(--dialog-body-padding);
}

.modal-shell__body--overflow-visible {
    overflow: visible;
}
</style>

