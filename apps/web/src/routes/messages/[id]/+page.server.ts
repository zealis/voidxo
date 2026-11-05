import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';
import {
  InboxRequestError,
  listMessages,
  markConversationRead,
  type InboxMessage,
  type InboxRecipient
} from '#lib/api/inbox.js';

type LoadResult = {
  user: AuthUser;
  counterpart: InboxRecipient | null;
  messages: InboxMessage[];
  page: number;
  hasMore: boolean;
  // notice 非空时表示无法打开会话，页面直接展示该提示。
  notice: string;
  loadError: boolean;
};

export const load: PageServerLoad = async ({ fetch, params, url }): Promise<LoadResult> => {
  const user = await fetchCurrentUser(fetch);
  if (!user) {
    redirect(303, '/login');
  }
  const counterpartID = Number(params.id);
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);
  const empty: LoadResult = {
    user,
    counterpart: null,
    messages: [],
    page,
    hasMore: false,
    notice: '',
    loadError: false
  };
  if (!Number.isInteger(counterpartID) || counterpartID <= 0) {
    return { ...empty, notice: '该用户不存在或已停用，无法发起私信。' };
  }
  if (counterpartID === user.id) {
    return { ...empty, notice: '不能与自己对话。' };
  }
  try {
    const conversation = await listMessages(counterpartID, page, 30, fetch);
    const hasUnread = conversation.items.some((item) => item.receiver_id === user.id && !item.read);
    if (hasUnread) {
      // 打开会话即视为已读；失败不影响消息展示，未读数会在下次访问时纠正。
      try {
        await markConversationRead(counterpartID, fetch);
      } catch {
        // 忽略：已读失败不应阻断会话阅读。
      }
    }
    return {
      user,
      counterpart: conversation.counterpart,
      messages: conversation.items,
      page: conversation.page,
      hasMore: conversation.has_more,
      notice: '',
      loadError: false
    };
  } catch (cause) {
    if (cause instanceof InboxRequestError && cause.status === 404) {
      return { ...empty, notice: '该用户不存在或已停用，无法发起私信。' };
    }
    return { ...empty, loadError: true };
  }
};
