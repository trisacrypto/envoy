/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["pkg/server/templates/**/*.jet"],
  theme: {
    extend: {},
  },
  plugins: [require("daisyui")],
}

