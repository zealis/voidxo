import { error, redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';
import { ContentRequestError, getQuestion, listTopics, type QuestionDetail, type Topic } from '#lib/api/content.js';

export const load: PageServerLoad = async ({ fetch, params }): Promise<{
  user: AuthUser;
  question: QuestionDetail;
  topics: Topic[];
}> => {
  const user = await fetchCurrentUser(fetch);
  if (!user) {
    redirect(303, '/login');
  }
  const id = Number(params.id);
  if (!Number.isSafeInteger(id) || id <= 0) {
    error(404, '问题不存在');
  }

  let question: QuestionDetail;
  let topics: Topic[];
  try {
    [question, topics] = await Promise.all([getQuestion(id, fetch), listTopics(50, fetch)]).then(
      ([detail, list]) => [detail, list.items] as [QuestionDetail, Topic[]]
    );
  } catch (cause) {
    if (cause instanceof ContentRequestError && cause.status === 404) {
      error(404, '问题不存在');
    }
    throw cause;
  }

  // 权限判断放在 try 之外，避免 error() 抛出的 HttpError 被数据加载的 catch 吞掉。
  const canEdit =
    user.id === question.user_id ||
    user.permissions.includes('content.moderate') ||
    user.permissions.includes('content.manage');
  if (!canEdit) {
    error(403, '没有编辑该问题的权限');
  }
  return { user, question, topics };
};
