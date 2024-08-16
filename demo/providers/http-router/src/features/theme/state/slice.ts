import {ImmerStateCreator} from '@/features/core/state/types';

type ThemeState = {
  theme: {
    mode: 'light' | 'dark' | 'system';
  };
};

type ThemeActions = {
  toggleTheme: () => void;
};

type ThemeSlice = ThemeState & ThemeActions;

const createThemeSlice: ImmerStateCreator<ThemeSlice> = (set) => ({
  theme: {
    mode: 'system',
  },
  toggleTheme() {
    set((state) => {
      state.theme.mode = state.theme.mode === 'light' ? 'dark' : 'light';
    });
  },
});

export {createThemeSlice, type ThemeSlice};
