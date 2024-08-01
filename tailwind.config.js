/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "pkg/web/templates/*.html",
    "pkg/web/templates/**/*.html"
  ],
  theme: {
    extend: {},
  },
  plugins: [require("daisyui")],
  safelist: [
    "text-blue-700",
    "text-gray-500",
    "text-yellow-700",
    "text-warning",
    "text-success",
  ],
  daisyui: {
    themes: [
      {
        light: {
          ...require("daisyui/src/theming/themes")["light"],
          primary: "#55ACD8",
          secondary: "#517994",
          success: "#15803D",
          warning: "#B91C1C",
          neutral: "#D9D9D9",
        }
      }
    ]
  }
}

