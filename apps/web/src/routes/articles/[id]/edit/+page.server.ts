import { error, redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';
import { ContentRequestError, getArticle, listTopics, type ArticleDetail, type Topic } from '#lib/api/content.js';

export const load: PageServerLoad = async ({ fetch, params }): Promise<{
  user: AuthUser;
  article: ArticleDetail;
  topics: Topic[];
}> => {
  const user = await fetchCurrentUser(fetch);
  if (!user) {
    redirect(303, '/login');
  }
  const id = Number(params.id);
  if (!Number.isSafeInteger(id) || id <= 0) {
    error(404, '文章不存在');
  }

  let article: ArticleDetail;
  let topics: Topic[];
  try {
    [article, topics] = await Promise.all([getArticle(id, fetch), listTopics(50, fetch)]).then(
      ([detail, list]) => [detail, list.items] as [ArticleDetail, Topic[]]
    );
  } catch (cause) {
    if (cause instanceof ContentRequestError && cause.status === 404) {
      error(404, '文章不存在');
    }
    throw cause;
  }
  return { user, article, topics };
};
