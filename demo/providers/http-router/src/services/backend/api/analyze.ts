import {ApiError, ApiSuccessResponse} from '@/services/backend/types';
import {getBaseUrl} from '@/services/backend/utils/getBaseUrl';
import {isApiErrorResponse, isApiSuccessResponse} from '@/services/backend/utils/typeGuards';
import {ConfigResponse} from '@/services/config/context';

type AnalyzeResponse = ApiSuccessResponse<{
  jobId: string;
}>;

function analyze(config: ConfigResponse) {
  return async (file: File) => {
    const res = await fetch(getBaseUrl(config)('/analyze'), {
      method: 'POST',
      body: file.stream(),
      headers: {
        'Content-Type': 'application/octet-stream',
      },
    });

    if (!res.ok) {
      throw new Error(`Failed to upload file: ${res.statusText}`);
    }

    const data = await res.json();

    if (isApiErrorResponse(data)) {
      throw new ApiError(data.error, res);
    }

    if (!isAnalyzeResponse(data)) {
      throw new Error('Invalid API response');
    }

    return data;
  };
}

function isAnalyzeResponse(res: unknown): res is AnalyzeResponse {
  return isApiSuccessResponse(res, {jobId: 'string'});
}

export {analyze};
