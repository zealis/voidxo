export type PublicOptions = {
  site_name: string;
  site_description: string;
  site_keywords: string;
  language: string;
  [key: string]: string | number | boolean;
};

export async function readPublicOptions(response: Response): Promise<PublicOptions> {
  if (!response.ok) {
    throw new Error(`OPTIONS_REQUEST_FAILED:${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!payload || typeof payload !== 'object' || !('data' in payload)) {
    throw new Error('OPTIONS_RESPONSE_INVALID');
  }
  const data = (payload as { data: unknown }).data;
  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    throw new Error('OPTIONS_RESPONSE_INVALID');
  }
  const options = data as Record<string, unknown>;
  if (typeof options.site_name !== 'string') {
    throw new Error('OPTIONS_RESPONSE_INVALID');
  }
  return options as PublicOptions;
}
