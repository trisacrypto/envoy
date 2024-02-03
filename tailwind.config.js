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
}

