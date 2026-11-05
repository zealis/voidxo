import { describe, expect, it, vi } from 'vitest';
import { listMessages, markConversationRead, sendMessage } from './inbox';

describe('inbox API', () => {
  it('会话消息携带对方信息并归一化空列表', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: {
        counterpart: { id: 2, username: 'tester', avatar_key: '' },
        items: null,
        page: 1,
        page_size: 30,
        has_more: false
      }
    }), { status: 200 }));

    const result = await listMessages(2, 1, 30, fetchFn);
    expect(fetchFn).toHaveBeenCalledWith('/api/v1/inbox/2?page=1&page_size=30', { credentials: 'same-origin' });
    expect(result.counterpart.username).toBe('tester');
    expect(result.items).toEqual([]);
  });

  it('会话缺少对方信息视为格式错误', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: { items: [], page: 1, page_size: 30, has_more: false }
    }), { status: 200 }));

    await expect(listMessages(2, 1, 30, fetchFn)).rejects.toMatchObject({ code: 'invalid_response' });
  });

  it('发送私信先取 CSRF 再提交', async () => {
    const fetchFn = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { csrf_token: 'x'.repeat(40) } }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        data: {
          message: {
            id: 9, sender_id: 1, sender_name: 'admin', receiver_id: 2,
            content_markdown: 'hi', content_rendered: '<p>hi</p>', read: false,
            created_at: '2026-11-05T00:00:00Z'
          }
        }
      }), { status: 201 }));

    const message = await sendMessage(2, 'hi', fetchFn);
    expect(fetchFn).toHaveBeenCalledTimes(2);
    expect(fetchFn.mock.calls[1][0]).toBe('/api/v1/inbox');
    expect(fetchFn.mock.calls[1][1]).toMatchObject({ method: 'POST' });
    expect(message.id).toBe(9);
  });

  it('已读接口返回更新条数与剩余未读', async () => {
    const fetchFn = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { csrf_token: 'x'.repeat(40) } }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { updated: 2, unread: 0 } }), { status: 200 }));

    const result = await markConversationRead(2, fetchFn);
    expect(result).toEqual({ updated: 2, unread: 0 });
  });

  it('限流错误转成可展示异常', async () => {
    const fetchFn = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { csrf_token: 'x'.repeat(40) } }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        error: { code: 'inbox_rate_limited', message: '发送过于频繁，请稍后再试' }
      }), { status: 429 }));

    await expect(sendMessage(2, 'hi', fetchFn)).rejects.toMatchObject({
      status: 429,
      code: 'inbox_rate_limited'
    });
  });
});
