import {useAppStore} from '@/features/core/state/store';
import React from 'react';

function useThemeClass() {
  const [mode, toggleTheme] = useAppStore((state) => [state.theme.mode, state.toggleTheme]);
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  const theme = mode === 'system' ? (prefersDark ? 'dark' : 'light') : mode;

  const updateClasses = React.useCallback((theme: 'dark' | 'light') => {
    document.documentElement.classList.remove('dark', 'light');
    document.documentElement.classList.add(theme);
  }, []);

  React.useEffect(() => {
    updateClasses(theme);
  }, [theme, updateClasses]);

  React.useEffect(() => {
    if (mode === 'system') {
      const query = window.matchMedia('(prefers-color-scheme: dark)');
      const handler = (e: MediaQueryListEvent) => updateClasses(e.matches ? 'dark' : 'light');

      query.addEventListener('change', handler);

      return () => {
        query.removeEventListener('change', handler);
      };
    }
  }, [mode, toggleTheme, updateClasses]);
}

export {useThemeClass};
