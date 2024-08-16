import type {Config} from 'tailwindcss';

export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        foreground: 'hsl(var(--ui-color-foreground))',
        background: 'hsl(var(--ui-color-background))',
        primary: {
          DEFAULT: 'hsl(var(--ui-color-primary))',
          foreground: 'hsl(var(--ui-color-primary-foreground))',
          contrast: 'hsl(var(--ui-color-primary-contrast))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--ui-color-secondary))',
          foreground: 'hsl(var(--ui-color-secondary-foreground))',
          contrast: 'hsl(var(--ui-color-secondary-contrast))',
        },
        surface: {
          DEFAULT: 'hsl(var(--ui-color-surface))',
          contrast: 'hsl(var(--ui-color-surface-contrast))',
        },
        border: {
          DEFAULT: 'hsl(var(--ui-color-border))',
          surface: 'hsl(var(--ui-color-border-contrast))',
        },
        accent: {
          DEFAULT: 'hsl(var(--ui-color-accent))',
          contrast: 'hsl(var(--ui-color-accent-contrast))',
        },
      },
    },
  },
  plugins: [],
} satisfies Config;
