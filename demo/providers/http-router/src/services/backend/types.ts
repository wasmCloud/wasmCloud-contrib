type ApiResponse<T> = {
  error?: string;
  data?: T;
};

type ApiErrorResponse = Required<ApiResponse<never>>;

type ApiSuccessResponse<T> = Required<Omit<ApiResponse<T>, 'error'>>;

class ApiError extends Error {
  constructor(message: string, public response: Response) {
    super(message);
  }
}

export type {ApiResponse, ApiErrorResponse, ApiSuccessResponse};

export {ApiError};
