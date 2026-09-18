/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: "class",
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./lib/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        canvas: {
          DEFAULT: "#0a0c10",
          raised: "#10141c",
          overlay: "#161c27",
        },
        ink: {
          DEFAULT: "#e8edf5",
          muted: "#8b97ab",
          faint: "#5c677a",
        },
        accent: {
          DEFAULT: "#3d9cf0",
          soft: "#1a3a5c",
        },
        line: "#1e2633",
      },
      fontFamily: {
        sans: ["var(--font-geist-sans)", "Segoe UI", "sans-serif"],
        display: ["var(--font-space)", "Segoe UI", "sans-serif"],
      },
      boxShadow: {
        glow: "0 0 40px rgba(61, 156, 240, 0.12)",
      },
      keyframes: {
        fadeUp: {
          "0%": { opacity: "0", transform: "translateY(8px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        pulseSoft: {
          "0%, 100%": { opacity: "0.55" },
          "50%": { opacity: "1" },
        },
      },
      animation: {
        "fade-up": "fadeUp 0.45s ease-out both",
        "pulse-soft": "pulseSoft 2.4s ease-in-out infinite",
      },
    },
  },
  plugins: [],
};
