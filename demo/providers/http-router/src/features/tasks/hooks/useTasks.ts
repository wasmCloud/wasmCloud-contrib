import {useAppStore} from '@/features/core/state/store';

function useTask() {
  const tasks = useAppStore((state) => state.tasks);

  return tasks;
}
