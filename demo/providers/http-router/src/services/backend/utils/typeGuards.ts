import {ApiResponse, ApiErrorResponse, ApiSuccessResponse} from '@/services/backend/types';
import {hasProperty, isObject, isShapeShallow} from '@/utils/typeGuards';

function isApiErrorResponse(res: unknown): res is ApiErrorResponse {
  return isApiResponse(res) && hasProperty(res, 'error') && typeof res.error !== 'undefined';
}

function isApiSuccessResponse<Data extends Record<string, unknown>>(
  res: unknown,
  data: Data,
): res is ApiSuccessResponse<Data> {
  return !isApiErrorResponse(res) && isApiResponse(res, data);
}

function isApiResponse<Data extends Record<string, unknown>>(
  res: unknown,
  data?: Data,
): res is ApiResponse<Data> {
  // TODO(lachieh): Better way to check for shape of data.
  // isShapeShallow only checks on the first level of the object.
  return (
    isObject(res) &&
    hasProperty(res, 'data') &&
    isObject(res.data) &&
    (!data || isShapeShallow(res.data, data))
  );
}

export {isApiErrorResponse, isApiSuccessResponse, isApiResponse};
