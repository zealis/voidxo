// Vx* 封装层对外的数据形状。业务页面只传数据，不直接写 <mdui-*> 标签。

export type VxOption = {
  value: string;
  label: string;
  disabled?: boolean;
};

/** 试点阶段的 Tabs 面板为纯文本；需要富文本时再扩展为 Snippet。 */
export type VxTabItem = VxOption & {
  content: string;
};
