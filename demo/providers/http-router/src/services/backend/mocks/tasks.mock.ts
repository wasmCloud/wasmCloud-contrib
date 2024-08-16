import type {task as apiTaskFn, tasks as apiTasksFn} from '@/services/backend/api/tasks';

type TasksFunction = typeof apiTasksFn;
const tasks: TasksFunction = () => {
  return () => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          data: {
            tasks: [
              {
                id: '1',
                title: 'Task 1',
                description: 'Description 1',
                completed: false,
              },
              {
                id: '2',
                title: 'Task 2',
                description: 'Description 2',
                completed: true,
              },
              {
                id: '3',
                title: 'Task 3',
                description: 'Description 3',
                completed: false,
              },
            ],
          },
        });
      }, 500);
    });
  };
};

type TaskFunction = typeof apiTaskFn;
const task: TaskFunction = () => (id: string) => {
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve({
        data: {
          id: id,
          title: 'Task ' + id,
          description: 'Description ' + id,
          completed: false,
        },
      });
    }, 500);
  });
};

export {tasks, task};
