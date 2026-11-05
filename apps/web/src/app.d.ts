import type { HTMLAttributes } from 'svelte/elements';

// M5：mdui 是 Web Components，不是 Svelte 组件。Svelte 模板里的 <mdui-*>
// 需要显式元素声明，否则属性与事件没有类型提示。
// 这里只声明试点用到的三个组件，新增组件时按 mdui 的 d.ts 补齐。
type MduiButtonAttributes = HTMLAttributes<HTMLElement> & {
  variant?: 'elevated' | 'filled' | 'tonal' | 'outlined' | 'text';
  href?: string;
  target?: string;
  disabled?: boolean;
  loading?: boolean;
  type?: 'button' | 'submit' | 'reset';
  name?: string;
  value?: string;
  fullWidth?: boolean;
};

type MduiButtonIconAttributes = HTMLAttributes<HTMLElement> & {
  variant?: 'standard' | 'filled' | 'tonal' | 'outlined';
  href?: string;
  disabled?: boolean;
  loading?: boolean;
  icon?: string;
};

type MduiTextFieldAttributes = HTMLAttributes<HTMLElement> & {
  variant?: 'filled' | 'outlined';
  value?: string;
  label?: string;
  name?: string;
  type?: string;
  placeholder?: string;
  helper?: string;
  error?: string;
  autocomplete?: string;
  maxlength?: number;
  minlength?: number;
  inputmode?: string;
  pattern?: string;
  disabled?: boolean;
  readonly?: boolean;
  required?: boolean;
  clearable?: boolean;
  rows?: number;
};

declare global {
  namespace App {
    // 请求级用户状态将在认证模块接入时放入 Locals，禁止使用模块级全局用户 Store。
    // interface Locals {}
  }

  namespace svelteHTML {
    interface IntrinsicElements {
      'mdui-button': MduiButtonAttributes;
      'mdui-button-icon': MduiButtonIconAttributes;
      'mdui-text-field': MduiTextFieldAttributes;
      'mdui-checkbox': HTMLAttributes<HTMLElement> & {
        checked?: boolean;
        indeterminate?: boolean;
        disabled?: boolean;
        required?: boolean;
        name?: string;
        value?: string;
      };
      'mdui-switch': HTMLAttributes<HTMLElement> & {
        checked?: boolean;
        disabled?: boolean;
        required?: boolean;
        name?: string;
        value?: string;
      };
      'mdui-radio-group': HTMLAttributes<HTMLElement> & {
        value?: string;
        name?: string;
        disabled?: boolean;
        required?: boolean;
      };
      'mdui-radio': HTMLAttributes<HTMLElement> & { value?: string; disabled?: boolean };
      'mdui-select': HTMLAttributes<HTMLElement> & {
        value?: string | string[];
        name?: string;
        label?: string;
        placeholder?: string;
        helper?: string;
        variant?: 'filled' | 'outlined';
        disabled?: boolean;
        multiple?: boolean;
        clearable?: boolean;
      };
      'mdui-menu-item': HTMLAttributes<HTMLElement> & {
        value?: string;
        disabled?: boolean;
        selected?: boolean;
      };
      'mdui-dialog': HTMLAttributes<HTMLElement> & {
        open?: boolean;
        headline?: string;
        description?: string;
        fullscreen?: boolean;
        closeOnEsc?: boolean;
        closeOnOverlayClick?: boolean;
      };
      'mdui-snackbar': HTMLAttributes<HTMLElement> & {
        open?: boolean;
        placement?: string;
        action?: string;
        closeable?: boolean;
        autoCloseDelay?: number;
      };
      'mdui-tooltip': HTMLAttributes<HTMLElement> & {
        content?: string;
        headline?: string;
        placement?: string;
        trigger?: 'click' | 'hover' | 'focus' | string;
        variant?: 'plain' | 'rich';
        disabled?: boolean;
        openDelay?: number;
        closeDelay?: number;
      };
      'mdui-tabs': HTMLAttributes<HTMLElement> & {
        value?: string;
        variant?: 'primary' | 'secondary';
        placement?: string;
        fullWidth?: boolean;
      };
      'mdui-tab': HTMLAttributes<HTMLElement> & { value?: string; icon?: string };
      'mdui-tab-panel': HTMLAttributes<HTMLElement> & { value?: string; slot?: string };
      'mdui-chip': HTMLAttributes<HTMLElement> & {
        variant?: 'assist' | 'filter' | 'input' | 'suggestion';
        selectable?: boolean;
        selected?: boolean;
        deletable?: boolean;
        elevated?: boolean;
      };
      'mdui-badge': HTMLAttributes<HTMLElement> & { variant?: 'small' | 'large' };
      'mdui-card': HTMLAttributes<HTMLElement> & {
        variant?: 'elevated' | 'filled' | 'outlined';
        clickable?: boolean;
        disabled?: boolean;
      };
    }
  }
}

export {};
