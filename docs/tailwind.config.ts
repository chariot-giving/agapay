import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        background: "var(--background)",
        foreground: "var(--foreground)",
      },
      fontFamily: {
        // Poppins
        poppinsLight: ["Poppins Light", "sans-serif"], // 300
        poppins: ["Poppins", "sans-serif"], // 400
        poppinsMed: ["Poppins Medium", "sans-serif"], // 500
        poppinsSemibold: ["Poppins Semibold", "sans-serif"], // 600
        poppinsBold: ["Poppins Bold", "sans-serif"], // 700

        // Matter
        matterLight: ["Matter Light", "sans-serif"], // 300
        matter: ["Matter", "sans-serif"], // 400
        matterMedium: ["Matter Medium", "sans-serif"], // 500
        matterSemibold: ["Matter Semibold", "sans-serif"], // 600

        // SF Pro
        sfProLight: ["SF-Pro-Text-Light", "sans-serif"],
        sfProRegular: ["SF-Pro-Text-Regular", "sans-serif"],
        sfProMedium: ["SF-Pro-Text-Medium", "sans-serif"],
        sfProSemiBold: ["SF-Pro-Text-Semibold", "sans-serif"],
        sfProBold: ["SF-Pro-Text-Bold", "sans-serif"],

        // Others
        courier: ["Courier", "sans-serif"],
        whitin: ["Whitin", "serif"],
        merriweather: ["Merriweather", "serif"],
        latoBold: ["Lato Bold", "sans-serif"],
        graphik: ["Graphik Medium", "sans-serif"],
        sofiaProCondensed: ["SofiaPro Condensed", "sans-serif"],
      },
    },
  },
  plugins: [],
};
export default config;
