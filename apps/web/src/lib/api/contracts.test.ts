import { describe, expect, it } from 'vitest';
import { readPublicOptions } from './contracts';

describe('readPublicOptions', () => {
  it('读取统一响应中的公开配置', async () => {
    const response = new Response(JSON.stringify({ data: { site_name: 'VOIDXO', question_can_edit: false } }), { status: 200 });
    await expect(readPublicOptions(response)).resolves.toMatchObject({ site_name: 'VOIDXO', question_can_edit: false });
  });

  it('拒绝失败响应和非法契约', async () => {
    await expect(readPublicOptions(new Response('{}', { status: 500 }))).rejects.toThrow('OPTIONS_REQUEST_FAILED:500');
    await expect(readPublicOptions(new Response(JSON.stringify({ data: {} }), { status: 200 }))).rejects.toThrow('OPTIONS_RESPONSE_INVALID');
  });
});
