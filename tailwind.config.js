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
  daisyui: {
    themes: [
      {
        envoyTheme: {
          "info": "#55ACD8",
        },
      },
      "light",
    ]
  }
}

