import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import {
  ContentRequestError,
  getTopic,
  listArticles,
  listQuestions,
  type ArticleSummary,
  type QuestionSummary,
  type Topic
} from '#lib/api/content.js';
import { getFollowState, type FollowSummary } from '#lib/api/engagement.js';

export const load: PageServerLoad = async ({ fetch, params, parent }): Promise<{
  topic: Topic;
  questions: QuestionSummary[];
  articles: ArticleSummary[];
  follow: FollowSummary;
  loadError: boolean;
}> => {
  const id = Number(params.id);
  if (!Number.isInteger(id) || id <= 0) {
    error(404, '话题不存在');
  }

  let topic: Topic;
  try {
    topic = await getTopic(id, fetch);
  } catch (cause) {
    if (cause instanceof ContentRequestError && cause.status === 404) {
      error(404, '话题不存在');
    }
    error(500, '话题加载失败');
  }

  const { siteUser } = await parent();
  const [questionsResult, articlesResult, followResult] = await Promise.allSettled([
    listQuestions(1, 10, id, fetch, 'recent'),
    listArticles(1, 10, id, fetch, 'recent'),
    siteUser
      ? getFollowState('topic', id, fetch)
      : Promise.resolve({ following: false, follower_count: topic.follower_count })
  ]);

  return {
    topic,
    questions: questionsResult.status === 'fulfilled' ? questionsResult.value.items : [],
    articles: articlesResult.status === 'fulfilled' ? articlesResult.value.items : [],
    follow:
      followResult.status === 'fulfilled'
        ? followResult.value
        : { following: false, follower_count: topic.follower_count },
    loadError: questionsResult.status === 'rejected' || articlesResult.status === 'rejected'
  };
};
