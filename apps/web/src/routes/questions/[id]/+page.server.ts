import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';
import {
  ContentRequestError,
  getQuestion,
  listAnswers,
  listComments,
  type Answer,
  type CommentItem,
  type QuestionDetail
} from '#lib/api/content.js';
import { getFollowState, getVoteState, type FollowSummary, type VoteSummary } from '#lib/api/engagement.js';

export const load: PageServerLoad = async ({ fetch, params }): Promise<{
  question: QuestionDetail | null;
  answers: Answer[];
  comments: CommentItem[];
  user: AuthUser | null;
  vote: VoteSummary;
  follow: FollowSummary;
  loadError: boolean;
}> => {
  const id = Number(params.id);
  if (!Number.isSafeInteger(id) || id <= 0) {
    error(404, '问题不存在');
  }
  try {
    const [question, answers, comments, user] = await Promise.all([
      getQuestion(id, fetch),
      listAnswers(id, 1, 50, fetch),
      listComments('question', id, 1, 50, fetch),
      fetchCurrentUser(fetch)
    ]);

    let vote: VoteSummary = {
      vote_count: question.vote_count,
      vote_up_count: 0,
      vote_down_count: 0,
      my_vote: ''
    };
    let follow: FollowSummary = { following: false, follower_count: question.follower_count };
    if (user) {
      const [voteResult, followResult] = await Promise.allSettled([
        getVoteState('question', id, fetch),
        getFollowState('question', id, fetch)
      ]);
      if (voteResult.status === 'fulfilled') vote = voteResult.value;
      if (followResult.status === 'fulfilled') follow = followResult.value;
    }

    return { question, answers: answers.items, comments: comments.items, user, vote, follow, loadError: false };
  } catch (cause) {
    if (cause instanceof ContentRequestError && cause.status === 404) {
      error(404, '问题不存在');
    }
    return {
      question: null,
      answers: [],
      comments: [],
      user: null,
      vote: { vote_count: 0, vote_up_count: 0, vote_down_count: 0, my_vote: '' },
      follow: { following: false, follower_count: 0 },
      loadError: true
    };
  }
};
