<template>
  <section class="page-section">
    <div class="page-header">
      <h2 class="page-title">UED 缁勪欢</h2>
      <p class="page-description">
        闈㈠悜鏈湴浠ｇ悊妗岄潰鎺у埗鍙帮細绱у噾淇℃伅瀵嗗害銆佹竻鏅版搷浣滃眰绾с€佸彲鎵弿鐘舵€佸弽棣堬紝浠ュ強绋冲畾鐨勯敭鐩樼劍鐐逛笌寮瑰眰浣撻獙銆?
      </p>
    </div>

    <PanelBlock title="璁捐 Token">
      <div class="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
        <div class="sub-card">
          <p class="text-sm font-medium text-strong">涓绘搷浣?/p>
          <div class="mt-3 flex items-center gap-2">
            <span class="h-6 w-10 rounded bg-[var(--ued-color-primary)]"></span>
            <span class="font-mono text-xs text-muted">var(--ued-color-primary)</span>
          </div>
        </div>
        <div class="sub-card">
          <p class="text-sm font-medium text-strong">鍦嗚</p>
          <p class="mt-3 text-sm text-muted">鎺т欢 6px锛岄潰鏉?8px锛屾爣绛?4px锛屽崱鐗?12px锛岃兌鍥?9999px銆?/p>
        </div>
        <div class="sub-card">
          <p class="text-sm font-medium text-strong">鎺т欢楂樺害</p>
          <p class="mt-3 text-sm text-muted">XS 24 / SM 28 / MD 32 / LG 40銆?/p>
        </div>
        <div class="sub-card">
          <p class="text-sm font-medium text-strong">鐘舵€佽壊</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <UTag variant="success" dot>Success</UTag>
            <UTag variant="warning" dot>Warning</UTag>
            <UTag variant="error" dot>Error</UTag>
            <UTag variant="info" dot>Info</UTag>
          </div>
        </div>
      </div>
    </PanelBlock>

    <div class="section-grid lg:grid-cols-2">
      <PanelBlock title="鎸夐挳">
        <div class="space-y-3">
          <div class="flex flex-wrap gap-2">
            <UButton variant="primary">
              <template #icon>
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5v14" /><path d="M5 12h14" /></svg>
              </template>
              Primary
            </UButton>
            <UButton variant="success">Success</UButton>
            <UButton variant="warning">Warning</UButton>
            <UButton variant="error">Error</UButton>
            <UButton variant="info">Info</UButton>
            <UButton variant="secondary">Secondary</UButton>
            <UButton variant="ghost">Ghost</UButton>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <UButton size="xs">XS</UButton>
            <UButton size="sm">SM</UButton>
            <UButton size="md">MD</UButton>
            <UButton size="lg">LG</UButton>
            <UButton loading>鍔犺浇涓?/UButton>
          </div>
        </div>
      </PanelBlock>

      <PanelBlock title="寮€鍏?>
        <div class="space-y-3">
          <USwitch v-model="switchValue" label="鍚敤鑷姩鍋ュ悍妫€鏌? hint="閰嶇疆绫诲紑鍏崇敤浜庡紑鍏冲瀷鍙傛暟锛屾枃妗堜繚鎸佸姩瀹剧粨鏋勩€? />
          <USwitch :model-value="true" label="淇濈暀绯荤粺榛樿璺敱" hint="绂佺敤鎬佺ず渚? :disabled="true" />
        </div>
      </PanelBlock>
    </div>

    <div class="section-grid lg:grid-cols-2">
      <PanelBlock title="Alert 璀﹀憡鎻愮ず">
        <div class="space-y-3">
          <UAlert type="success" message="淇濆瓨鎴愬姛锛岄厤缃凡鍐欏叆鏈湴浠ｇ悊銆? />
          <UAlert type="info" message="鎻愮ず淇℃伅" description="鐢ㄤ簬鎵胯浇椤甸潰鍐呭父椹荤殑璇存槑銆佺姸鎬佸彉鏇存垨杞婚噺寮曞銆? />
          <UAlert type="warning" message="浠ｇ悊灏氭湭閲嶈浇" description="淇濆瓨绔偣鍚庨渶瑕侀噸杞戒唬鐞嗭紝鏂扮殑璺敱閰嶇疆鎵嶄細鐢熸晥銆? closable />
          <UAlert type="error" message="杩炴帴澶辫触" description="璇锋鏌ヤ緵搴斿晢鍦板潃銆侀壌鏉?Key 鍜岀綉缁滆繛閫氭€с€? />
        </div>
      </PanelBlock>

      <PanelBlock title="Message 鍏ㄥ眬鎻愮ず">
        <div class="flex flex-wrap gap-2">
          <UButton variant="success" @click="showMessage('success')">Success</UButton>
          <UButton variant="info" @click="showMessage('info')">Info</UButton>
          <UButton variant="warning" @click="showMessage('warning')">Warning</UButton>
          <UButton variant="error" @click="showMessage('error')">Error</UButton>
          <UButton variant="secondary" @click="showLoadingMessage">Loading</UButton>
        </div>
      </PanelBlock>
    </div>

    <div class="section-grid lg:grid-cols-2">
      <PanelBlock title="Loading 鍔犺浇">
        <div class="space-y-4">
          <div class="flex flex-wrap items-center gap-5">
            <ULoading size="sm" />
            <ULoading />
            <ULoading size="lg" tip="鍔犺浇涓? />
          </div>
          <ULoading tip="姝ｅ湪鍔犺浇绔偣鏁版嵁..." :spinning="true">
            <div class="rounded-md border border-[var(--ued-color-divider)] bg-[var(--ued-color-muted)] p-4">
              <p class="text-sm font-medium text-strong">浠ｇ悊绔偣</p>
              <p class="mt-2 text-sm leading-6 text-muted">
                鍖哄煙鍔犺浇鐢ㄤ簬琛ㄦ牸銆佽鎯呴潰鏉挎垨閰嶇疆鍧楀埛鏂帮紝涓嶆墦鏂綋鍓嶉〉闈笂涓嬫枃銆?
              </p>
            </div>
          </ULoading>
          <UButton variant="secondary" @click="showFullscreenLoading">鍏ㄥ睆 Loading</UButton>
        </div>
      </PanelBlock>

      <PanelBlock title="鏍囩">
        <div class="space-y-3">
          <div class="flex flex-wrap gap-2">
            <UTag variant="primary" dot>primary</UTag>
            <UTag variant="success" dot>success</UTag>
            <UTag variant="warning" dot>warning</UTag>
            <UTag variant="error" dot>error</UTag>
            <UTag variant="info" dot>info</UTag>
            <UTag>neutral</UTag>
            <UTag code>openai-responses</UTag>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <UTag size="xs" variant="primary">xs</UTag>
            <UTag size="sm" variant="success">sm</UTag>
            <UTag size="md" variant="warning">md</UTag>
            <UTag size="lg" variant="error">lg</UTag>
            <UTag size="sm" code>/v1/responses</UTag>
          </div>
        </div>
      </PanelBlock>
    </div>

    <div class="section-grid lg:grid-cols-2">
      <PanelBlock title="杈撳叆涓庝笅鎷?>
        <div class="space-y-3">
          <UInput v-model="form.name" label="鍚嶇О" placeholder="璇疯緭鍏ヤ緵搴斿晢鍚嶇О" hint="琛ㄥ崟椤归噰鐢ㄤ笂 label銆佷笅鎺т欢甯冨眬銆? required />
          <USelect v-model="form.protocol" label="鍗忚" :options="protocolOptions" required />
          <UInput v-model="form.description" label="鎻忚堪" placeholder="璇疯緭鍏ョ敤閫旇鏄? textarea />
        </div>
      </PanelBlock>

      <PanelBlock title="寮圭獥">
        <div class="flex flex-wrap gap-2">
          <UButton @click="showModal = true">鏅€氬脊绐?/UButton>
          <UButton variant="error" @click="showConfirm = true">纭寮圭獥</UButton>
        </div>
      </PanelBlock>
    </div>

    <PanelBlock title="鍩虹琛ㄦ牸">
      <UTable
        :columns="columns"
        :rows="filteredRows"
        row-key="id"
        empty-text="鏆傛棤缁勪欢绀轰緥鏁版嵁銆?
        action-width="72px"
        pagination
        :page-size="3"
      >
        <template #query>
          <div class="table-query-form">
            <UInput
              v-model="basicTableQuery.keyword"
              label="鍏抽敭璇?
              hide-label
              placeholder="鎼滅储鍚嶇О"
              class="table-query-form__field table-query-form__field--keyword"
            />
            <USelect
              v-model="basicTableQuery.type"
              label="绫诲瀷"
              hide-label
              :options="basicTypeOptions"
              class="table-query-form__field table-query-form__field--select"
            />
            <USelect
              v-model="basicTableQuery.status"
              label="鐘舵€?
              hide-label
              :options="basicStatusOptions"
              class="table-query-form__field table-query-form__field--select"
            />
            <div class="table-query-form__actions">
              <UButton variant="secondary" @click="resetBasicTableQuery">閲嶇疆</UButton>
            </div>
          </div>
        </template>
        <template #cell-status="{ value }">
          <UTag :variant="value === '鍚敤' ? 'success' : 'error'" dot>{{ value }}</UTag>
        </template>
        <template #actions="{ row }">
          <div class="table-actions">
            <UIconButton icon="edit" :label="`缂栬緫绀轰緥 ${row.id}`" />
          </div>
        </template>
      </UTable>
    </PanelBlock>

    <PanelBlock title="鐜颁唬鍖栬〃鏍?>
      <p class="mb-3 text-xs text-muted">
        琛ㄦ牸娌跨敤 Ant Design 鐨勬祬琛ㄥご銆佽 hover銆侀珮瀵嗗害淇℃伅灞曠ず锛屽苟淇濈暀鍥哄畾鍒椼€佹枃瀛楃渷鐣ャ€乀ooltip 鍜屽垪瀵归綈鑳藉姏銆?
      </p>
      <UTable
        :columns="advancedColumns"
        :rows="advancedRows"
        row-key="id"
        fixed
        stripe
        action-width="90px"
      >
        <template #cell-status="{ value }">
          <UTag :variant="value === '鍚敤' ? 'success' : 'error'" dot>{{ value }}</UTag>
        </template>
        <template #actions="{ row }">
          <div class="table-actions">
            <UIconButton icon="edit" :label="`缂栬緫 ${row.name}`" />
            <UIconButton icon="detail" :label="`鏌ョ湅 ${row.name}`" />
          </div>
        </template>
      </UTable>
    </PanelBlock>

    <PanelBlock title="浜や簰琛ㄦ牸锛堥€夋嫨 / 琛岀偣鍑?/ 鍔犺浇锛?>
      <p class="mb-3 text-xs text-muted">
        UTable 鏀寔 selectable 澶氶€夈€乺ow-click 琛岀偣鍑汇€乴oading 鍔犺浇閬僵銆傞€夋嫨鐘舵€侀€氳繃 v-model:selectedKeys 鍙屽悜缁戝畾锛屾敮鎸佽法椤典繚鎸佷笌鍏ㄩ€夊綋鍓嶉〉銆?
      </p>
      <UTable
        :columns="interactiveColumns"
        :rows="interactiveRows"
        row-key="id"
        selectable
        row-clickable
        :selected-keys="interactiveSelected"
        :loading="interactiveLoading"
        loading-text="姝ｅ湪鍒锋柊鈥?
        @update:selected-keys="interactiveSelected = $event"
        @row-click="onInteractiveRowClick"
        @selection-change="onInteractiveSelectionChange"
      >
        <template #cell-status="{ value }">
          <UTag :variant="value === '鍚敤' ? 'success' : 'error'" dot>{{ value }}</UTag>
        </template>
        <template #actions="{ row }">
          <div class="table-actions">
            <UIconButton icon="detail" :label="`鏌ョ湅 ${row.name}`" />
          </div>
        </template>
      </UTable>
      <div class="mt-3 flex flex-wrap items-center gap-2">
        <UButton variant="secondary" size="sm" @click="toggleInteractiveLoading">
          {{ interactiveLoading ? "鍋滄鍔犺浇" : "妯℃嫙鍔犺浇" }}
        </UButton>
        <UButton variant="secondary" size="sm" :disabled="!interactiveSelected.length" @click="interactiveSelected = []">
          娓呴櫎閫夋嫨锛坽{ interactiveSelected.length }}锛?
        </UButton>
      </div>
    </PanelBlock>

    <PanelBlock title="Tooltip 缁勪欢">
      <div class="flex flex-wrap items-center gap-4">
        <UTooltip content="杩欐槸涓€涓熀纭€鐨勬彁绀烘枃鏈?>
          <UButton size="sm" variant="secondary">鎮仠鏌ョ湅鎻愮ず</UButton>
        </UTooltip>
        <UTooltip content="鎻愮ず鍙互鍖呭惈寰堥暱鐨勮鏄庡唴瀹癸紝鐢ㄤ簬琛ュ厖鐣岄潰涓棤娉曞畬鏁村睍绀虹殑淇℃伅銆?>
          <span class="text-sm text-secondary underline decoration-dotted">闀挎枃鏈彁绀?/span>
        </UTooltip>
      </div>
    </PanelBlock>

    <UModal v-model:open="showModal" title="鏅€氬脊绐?>
      <p class="text-sm leading-6 text-secondary">
        鐢ㄤ簬鎵胯浇璇存槑銆侀瑙堟垨琛ㄥ崟鍐呭銆?
      </p>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="secondary" @click="showModal = false">鍏抽棴</UButton>
          <UButton @click="showModal = false">纭</UButton>
        </div>
      </template>
    </UModal>

    <UConfirmDialog
      v-model:open="showConfirm"
      title="纭鍒犻櫎绀轰緥"
      message="鍒犻櫎鍚庡皢鏃犳硶鎭㈠璇ョず渚嬫暟鎹€?
      description="纭寮圭獥閫傜敤浜庡垹闄ゃ€佽鐩栥€佸仠鐢ㄧ瓑楂橀闄╂搷浣溿€?
      confirm-text="纭鍒犻櫎"
      cancel-text="鍙栨秷"
      danger
      @confirm="showConfirm = false"
    />
    <ULoading fullscreen tip="姝ｅ湪鍔犺浇椤甸潰..." :spinning="fullscreenLoading" />
  </section>
</template>

<script setup>
import { computed, reactive, ref } from "vue";
import PanelBlock from "../components/PanelBlock.vue";
import UAlert from "../components/ued/UAlert.vue";
import UButton from "../components/ued/UButton.vue";
import UConfirmDialog from "../components/ued/UConfirmDialog.vue";
import UIconButton from "../components/ued/UIconButton.vue";
import UInput from "../components/ued/UInput.vue";
import ULoading from "../components/ued/ULoading.vue";
import UModal from "../components/ued/UModal.vue";
import USelect from "../components/ued/USelect.vue";
import USwitch from "../components/ued/USwitch.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";
import UTooltip from "../components/ued/UTooltip.vue";
import { message } from "../components/ued/message";

const showModal = ref(false);
const showConfirm = ref(false);
const fullscreenLoading = ref(false);
const switchValue = ref(true);

// Interactive table demo: selection / row-click / loading.
const interactiveSelected = ref([]);
const interactiveLoading = ref(false);

const interactiveColumns = [
  { key: "name", title: "鍚嶇О" },
  { key: "type", title: "绫诲瀷" },
  { key: "status", title: "鐘舵€?, align: "center" },
];

const interactiveRows = [
  { id: "i1", name: "閫夋嫨绀轰緥 A", type: "灞曠ず缁勪欢", status: "鍚敤" },
  { id: "i2", name: "閫夋嫨绀轰緥 B", type: "鍙嶉缁勪欢", status: "鍋滅敤" },
  { id: "i3", name: "閫夋嫨绀轰緥 C", type: "鏁版嵁缁勪欢", status: "鍚敤" },
];

function toggleInteractiveLoading() {
  interactiveLoading.value = !interactiveLoading.value;
}

function onInteractiveRowClick({ row }) {
  message.info(`鐐瑰嚮浜嗚锛?{row.name}`);
}

function onInteractiveSelectionChange(keys) {
  if (keys.length) {
    message.info(`宸查€夋嫨 ${keys.length} 椤筦);
  }
}

const form = reactive({
  name: "",
  protocol: "openai-responses",
  description: "",
});

const basicTableQuery = reactive({
  keyword: "",
  type: "all",
  status: "all",
});

const protocolOptions = [
  { label: "anthropic", value: "anthropic" },
  { label: "openai-chat", value: "openai-chat" },
  { label: "openai-responses", value: "openai-responses" },
];

const messageText = {
  success: "鎿嶄綔鎴愬姛锛岄厤缃凡淇濆瓨銆?,
  info: "杩欐槸涓€鏉℃櫘閫氭彁绀轰俊鎭€?,
  warning: "璇峰厛閲嶈浇浠ｇ悊浣块厤缃敓鏁堛€?,
  error: "鎿嶄綔澶辫触锛岃妫€鏌ヨ緭鍏ュ悗閲嶈瘯銆?,
};

function showMessage(type) {
  message[type](messageText[type]);
}

function showLoadingMessage() {
  const key = "ued-loading-demo";
  message.loading({ key, content: "姝ｅ湪鍚屾閰嶇疆..." });
  window.setTimeout(() => {
    message.success({ key, content: "閰嶇疆鍚屾瀹屾垚銆? });
  }, 1200);
}

function showFullscreenLoading() {
  fullscreenLoading.value = true;
  window.setTimeout(() => {
    fullscreenLoading.value = false;
  }, 1200);
}

const columns = [
  { key: "name", title: "鍚嶇О" },
  { key: "type", title: "绫诲瀷" },
  { key: "status", title: "鐘舵€? },
];

const rows = [
  { id: "1", name: "渚涘簲鍟嗘寜閽?, type: "鎿嶄綔缁勪欢", status: "鍚敤" },
  { id: "2", name: "纭寮圭獥", type: "鍙嶉缁勪欢", status: "鍚敤" },
  { id: "3", name: "鐘舵€佹爣绛?, type: "灞曠ず缁勪欢", status: "鍚敤" },
  { id: "4", name: "鍒嗛〉琛ㄦ牸", type: "鏁版嵁缁勪欢", status: "鍚敤" },
  { id: "5", name: "娑堟伅鎻愮ず", type: "鍙嶉缁勪欢", status: "鍋滅敤" },
  { id: "6", name: "涓嬫媺绛涢€?, type: "琛ㄥ崟缁勪欢", status: "鍚敤" },
];

const basicTypeOptions = [
  { label: "鍏ㄩ儴绫诲瀷", value: "all" },
  { label: "鎿嶄綔缁勪欢", value: "鎿嶄綔缁勪欢" },
  { label: "鍙嶉缁勪欢", value: "鍙嶉缁勪欢" },
  { label: "灞曠ず缁勪欢", value: "灞曠ず缁勪欢" },
  { label: "鏁版嵁缁勪欢", value: "鏁版嵁缁勪欢" },
  { label: "琛ㄥ崟缁勪欢", value: "琛ㄥ崟缁勪欢" },
];

const basicStatusOptions = [
  { label: "鍏ㄩ儴鐘舵€?, value: "all" },
  { label: "鍚敤", value: "鍚敤" },
  { label: "鍋滅敤", value: "鍋滅敤" },
];

const filteredRows = computed(() => {
  const keyword = basicTableQuery.keyword.trim().toLowerCase();

  return rows.filter((row) => {
    const matchesKeyword = !keyword || row.name.toLowerCase().includes(keyword);
    const matchesType = basicTableQuery.type === "all" || row.type === basicTableQuery.type;
    const matchesStatus = basicTableQuery.status === "all" || row.status === basicTableQuery.status;

    return matchesKeyword && matchesType && matchesStatus;
  });
});

function resetBasicTableQuery() {
  basicTableQuery.keyword = "";
  basicTableQuery.type = "all";
  basicTableQuery.status = "all";
}

const advancedColumns = [
  { key: "id", title: "ID", width: "80px", fixed: "left", align: "center" },
  {
    key: "name",
    title: "鍚嶇О锛堝浐瀹氬乏渚?+ 鐪佺暐 + Tooltip锛?,
    width: "200px",
    fixed: "left",
    ellipsis: true,
    tooltip: true,
  },
  {
    key: "description",
    title: "璇存槑锛堟孩鍑虹渷鐣?+ Tooltip锛?,
    width: "280px",
    ellipsis: true,
    tooltip: true,
  },
  { key: "type", title: "绫诲瀷", width: "120px" },
  { key: "status", title: "鐘舵€?, width: "90px", align: "center" },
  { key: "count", title: "璁℃暟", width: "100px", align: "right" },
];

const advancedRows = [
  {
    id: "101",
    name: "杩欐槸涓€涓潪甯搁暱鐨勭粍浠跺悕绉帮紝鐢ㄤ簬娴嬭瘯鏂囧瓧婧㈠嚭鐪佺暐鍜?Tooltip 鍔熻兘",
    description:
      "杩欐槸瓒呴暱璇存槑鏂囨湰锛岀敤浜庢紨绀哄綋鍗曞厓鏍煎唴瀹硅秴鍑哄垪瀹芥椂锛屽浣曢€氳繃鐪佺暐鍙烽殣钘忓苟鍦ㄦ偓鍋滄椂閫氳繃 Tooltip 灞曠ず瀹屾暣鍐呭銆?,
    type: "鎿嶄綔缁勪欢",
    status: "鍚敤",
    count: 128,
  },
  {
    id: "102",
    name: "纭寮圭獥",
    description: "鐢ㄤ簬浜屾纭鐨勫脊绐楃粍浠讹紝閫傜敤浜庡垹闄ゃ€佽鐩栫瓑楂橀闄╂搷浣滃満鏅€?,
    type: "鍙嶉缁勪欢",
    status: "鍋滅敤",
    count: 56,
  },
  {
    id: "103",
    name: "鏁版嵁琛ㄦ牸",
    description:
      "鐜颁唬鍖栬〃鏍肩粍浠讹紝鏀寔鍥哄畾鍒椼€佽〃澶村浐瀹氥€佹枒椹汗銆佹枃瀛楁孩鍑虹渷鐣ャ€乀ooltip 鎻愮ず绛変赴瀵岀壒鎬с€?,
    type: "鏁版嵁灞曠ず",
    status: "鍚敤",
    count: 2048,
  },
  {
    id: "104",
    name: "涓嬫媺閫夋嫨鍣?,
    description: "鏀寔鍗曢€夈€佹悳绱㈠拰鍒嗙粍鐨勬爣鍑嗕笅鎷夐€夋嫨缁勪欢銆?,
    type: "琛ㄥ崟缁勪欢",
    status: "鍚敤",
    count: 12,
  },
  {
    id: "105",
    name: "寮€鍏虫寜閽?,
    description: "鐢ㄤ簬鍒囨崲閰嶇疆椤圭殑甯冨皵鐘舵€侊紝鏀寔绂佺敤鎬佸拰鎻愮ず鏂囨湰銆?,
    type: "琛ㄥ崟缁勪欢",
    status: "鍚敤",
    count: 99,
  },
];
</script>

</style>
