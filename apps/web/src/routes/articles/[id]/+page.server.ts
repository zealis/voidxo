import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';
import {
  ContentRequestError,
  getArticle,
  listComments,
  type ArticleDetail,
  type CommentItem
} from '#lib/api/content.js';
import { getFollowState, getVoteState, type FollowSummary, type VoteSummary } from '#lib/api/engagement.js';

export const load: PageServerLoad = async ({ fetch, params }): Promise<{
  article: ArticleDetail | null;
  comments: CommentItem[];
  user: AuthUser | null;
  vote: VoteSummary;
  follow: FollowSummary;
  loadError: boolean;
}> => {
  const id = Number(params.id);
  if (!Number.isSafeInteger(id) || id <= 0) {
    error(404, '文章不存在');
  }
  try {
    const [article, comments, user] = await Promise.all([
      getArticle(id, fetch),
      listComments('article', id, 1, 50, fetch),
      fetchCurrentUser(fetch)
    ]);

    let vote: VoteSummary = {
      vote_count: article.vote_count,
      vote_up_count: 0,
      vote_down_count: 0,
      my_vote: ''
    };
    let follow: FollowSummary = { following: false, follower_count: article.follower_count };
    if (user) {
      const [voteResult, followResult] = await Promise.allSettled([
        getVoteState('article', id, fetch),
        getFollowState('article', id, fetch)
      ]);
      if (voteResult.status === 'fulfilled') vote = voteResult.value;
      if (followResult.status === 'fulfilled') follow = followResult.value;
    }

    return { article, comments: comments.items, user, vote, follow, loadError: false };
  } catch (cause) {
    if (cause instanceof ContentRequestError && cause.status === 404) {
      error(404, '文章不存在');
    }
    return {
      article: null,
      comments: [],
      user: null,
      vote: { vote_count: 0, vote_up_count: 0, vote_down_count: 0, my_vote: '' },
      follow: { following: false, follower_count: 0 },
      loadError: true
    };
  }
};
