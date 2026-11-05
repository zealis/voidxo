import type { PageServerLoad } from './$types';
import { listArticles, listQuestions, listTopics, type ArticleSummary, type QuestionSummary, type Topic } from '#lib/api/content.js';
import { fetchSiteStats, type SiteStats } from '#lib/api/stats.js';

// 首页五个区块与旧站 index.php 一一对应：推荐话题 + 最近/热门提问 + 最新/热门文章。
export const load: PageServerLoad = async ({ fetch }): Promise<{
  topics: Topic[];
  questionsRecent: QuestionSummary[];
  questionsPopular: QuestionSummary[];
  articlesRecent: ArticleSummary[];
  articlesPopular: ArticleSummary[];
  stats: SiteStats | null;
  loadError: boolean;
}> => {
  const results = await Promise.allSettled([
    listTopics(12, fetch, 'recommended'),
    listQuestions(1, 5, undefined, fetch, 'recent'),
    listQuestions(1, 5, undefined, fetch, 'popular'),
    listArticles(1, 5, undefined, fetch, 'recent'),
    listArticles(1, 5, undefined, fetch, 'popular'),
    fetchSiteStats(fetch)
  ]);

  const items = <T>(index: number): T[] => {
    const result = results[index];
    if (result.status !== 'fulfilled') return [];
    return ((result.value as unknown as { items?: T[] }).items ?? []) as T[];
  };

  const statsResult = results[5];
  return {
    topics: items<Topic>(0),
    questionsRecent: items<QuestionSummary>(1),
    questionsPopular: items<QuestionSummary>(2),
    articlesRecent: items<ArticleSummary>(3),
    articlesPopular: items<ArticleSummary>(4),
    stats: statsResult.status === 'fulfilled' ? (statsResult.value as SiteStats) : null,
    // 任一内容区块失败都不阻断整页，但记录一次以便展示提示。
    loadError: results.slice(0, 5).some((result) => result.status === 'rejected')
  };
};
