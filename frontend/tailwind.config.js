/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    fontFamily: {
      sans: ['-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Noto Sans', 'Helvetica', 'Arial', 'sans-serif'],
      mono: ['SF Mono', 'Fira Code', 'Fira Mono', 'Menlo', 'Consolas', 'monospace'],
    },
    extend: {},
  },
  plugins: [],
}
