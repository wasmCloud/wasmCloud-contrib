import {ApiSuccessResponse} from '@/services/backend/types';
import {ConfigResponse} from '@/services/config/context';

export type Task = {
  id: string;
  title: string;
  description: string;
  completed: boolean;
};

type TasksResponse = ApiSuccessResponse<{
  tasks: Task[];
}>;

function tasks(config: ConfigResponse): () => Promise<TasksResponse> {
  // TODO: implement
}

type TaskResponse = ApiSuccessResponse<Task>;

function task(config: ConfigResponse): (id: string) => Promise<TaskResponse> {
  // TODO: implement
}

export {tasks, task};
