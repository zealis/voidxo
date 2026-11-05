import { AuthRequestError, fetchCSRFToken } from './auth.js';

export type UploadedImage = {
  key: string;
  urls: Record<string, string>;
  width: number;
  height: number;
};

export class MediaRequestError extends Error {
  constructor(message: string, readonly status: number, readonly code: string) {
    super(message);
    this.name = 'MediaRequestError';
  }
}

export async function uploadAvatar(file: File, fetchFn: typeof fetch = fetch): Promise<UploadedImage> {
  return upload('/api/v1/user/avatar', 'avatar', file, fetchFn);
}

export async function uploadCover(file: File, fetchFn: typeof fetch = fetch): Promise<UploadedImage> {
  return upload('/api/v1/user/cover', 'cover', file, fetchFn);
}

export async function uploadContentImage(file: File, fetchFn: typeof fetch = fetch): Promise<UploadedImage> {
  return upload('/api/v1/images', 'image', file, fetchFn);
}

async function upload(path: string, field: string, file: File, fetchFn: typeof fetch): Promise<UploadedImage> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const form = new FormData();
  form.append(field, file, file.name);
  const response = await fetchFn(path, {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': csrfToken },
    body: form
  });

  let payload: { data?: { image?: UploadedImage }; error?: { code?: string; message?: string } } | null = null;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }
  if (!response.ok) {
    throw new MediaRequestError(
      payload?.error?.message ?? '图片上传失败',
      response.status,
      payload?.error?.code ?? 'upload_failed'
    );
  }
  const image = payload?.data?.image;
  if (!image || typeof image.key !== 'string' || !image.urls) {
    throw new MediaRequestError('图片响应格式不正确', response.status, 'invalid_response');
  }
  return image;
}

export { AuthRequestError };
