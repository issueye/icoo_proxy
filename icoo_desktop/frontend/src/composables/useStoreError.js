import { watch } from "vue";
import { message } from "../components/ued/message";

/**
 * Surface a store's `error` string through the global `message` component
 * instead of an inline `UAlert`, so API failures no longer push the layout.
 *
 * Watches `store.error` and, whenever it becomes a non-empty string, shows a
 * `message.error` toast and then clears the error so it doesn't re-fire.
 * Use once per view that previously rendered `<UAlert v-if="store.error">`.
 *
 * @param {import('pinia').Store<'store', { error: string }, ...>} store
 *   A Pinia store exposing a writable `error` string.
 */
export function useStoreError(store) {
  watch(
    () => store.error,
    (value) => {
      if (!value) {
        return;
      }
      message.error(value);
      // Clear so the same error doesn't re-surface on re-render and so a
      // repeat of the identical failure still triggers the toast.
      store.error = "";
    },
  );
}
