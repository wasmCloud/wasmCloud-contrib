import {ImmerStateCreator} from '@/features/core/state/types';
import {Task} from '@/services/backend/api/tasks';

type TasksState = {
  tasks: Task[];
};

type TasksActions = {
  addTask: (task: Task) => void;
  deleteTask: (id: string) => void;
  updateTask: (task: Task) => void;
};

type TasksSlice = TasksState & TasksActions;

const createTasksSlice: ImmerStateCreator<TasksSlice> = (set) => ({
  tasks: [],
  addTask(task: Task) {
    set((state) => {
      state.tasks.push(task);
    });
  },
  deleteTask(id) {
    set((state) => {
      state.tasks = state.tasks.filter((task) => task.id !== id);
    });
  },
  updateTask(task) {
    set((state) => {
      const index = state.tasks.findIndex((t) => t.id === task.id);
      state.tasks[index] = task;
    });
  },
});

export {createTasksSlice, type TasksSlice};
