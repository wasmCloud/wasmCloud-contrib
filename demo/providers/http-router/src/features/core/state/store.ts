import {createTasksSlice, TasksSlice} from '@/features/tasks/state/slice';
import {createThemeSlice, ThemeSlice} from '@/features/theme/state/slice';
import {create} from 'zustand';
import {persist, devtools, createJSONStorage} from 'zustand/middleware';
import {immer} from 'zustand/middleware/immer';

type RootStore = ThemeSlice & TasksSlice & {};

const appStore = immer<RootStore>((...args) => ({
  ...createThemeSlice(...args),
  ...createTasksSlice(...args),
}));

const useAppStore = create<RootStore>()(
  devtools(
    persist(appStore, {
      name: 'wawesomecloud',
      storage: createJSONStorage(() => localStorage),
    }),
    {
      enabled: true,
    },
  ),
);

export {useAppStore};
export type {RootStore};
