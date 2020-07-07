const path = require("path");

module.exports = {
  title: "Gnet",
  tagline:
    "A high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go",
  url: "https://gnet.host",
  baseUrl: "/",
  favicon: "favicon.ico",
  organizationName: "panjf2000",
  projectName: "gnet",
  customFields: {
    metadata: require("./metadata"),
  },
  themeConfig: {
    navbar: {
      hideOnScroll: true,
      logo: {
        alt: "Gnet",
        src: "img/logo-light.svg",
      },
      links: [
        { href: "https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc", label: "Documentations", position: "left" },
        { href: "https://taohuawu.club/", label: "Blog", position: "left" },
        { href: "https://github.com/gnet-io", label: "Community", position: "right" },
        {
          href: "https://github.com/panjf2000/gnet",
          label: "GitHub",
          position: "right",
        },
      ],
    },
    image: "img/open-graph.png",
    prism: {
      theme: require("prism-react-renderer/themes/github"),
      darkTheme: require("prism-react-renderer/themes/dracula"),
    },
    footer: {
      links: [
        {
          title: "About",
          items: [
            {
              label: "What is Gnet?",
              href: "https://github.com/panjf2000/gnet#-introduction",
            },
            {
              label: "Contact Us",
              to: "contact/",
            },
          ],
        },
        {
          title: "Community",
          items: [
            {
              label: "Github",
              href: "https://github.com/panjf2000/gnet",
            },
            {
              label: "Github Organization",
              href: "https://github.com/gnet-io",
            },
            {
              label: "Twitter",
              href: "https://twitter.com/_andy_pan",
            },
            {
              label: "Blog",
              href: "https://taohuawu.club/",
            },
          ],
        },
      ],
      logo: {
        alt: "Gnet",
        src: "/img/footer-logo.svg",
        href: "https://github.com/panjf2000/gnet/",
      },
      copyright: `Copyright © ${new Date().getFullYear()} Andy Pan`,
    },
    algolia: {
      apiKey: "",
      indexName: "",
      algoliaOptions: {}, // Optional, if provided by Algolia
    },
  },
  presets: [],
  plugins: [
    // [
    //   "@docusaurus/plugin-content-docs",
    //   {
    //     editUrl: "https://github.com/panjf2000/gnet/edit/master/website/",
    //     sidebarPath: require.resolve("./sidebars.js"),
    //   },
    // ],
    // [
    //   "@docusaurus/plugin-content-blog",
    //   {
    //     feedOptions: {
    //       type: "all",
    //       copyright: `Copyright © ${new Date().getFullYear()} Andy Pan.`,
    //     },
    //   },
    // ],
    ["@docusaurus/plugin-content-pages", {}],
    // path.resolve(__dirname, "./plugins/guides"),
    // path.resolve(__dirname, "./plugins/highlights"),
    // path.resolve(__dirname, "./plugins/releases"),
    // [path.resolve(__dirname, "./plugins/sitemap"), {}],
  ],
  scripts: [],
  stylesheets: [
    "https://fonts.googleapis.com/css?family=Ubuntu|Roboto|Source+Code+Pro",
    "https://at-ui.github.io/feather-font/css/iconfont.css",
  ],
  themes: [
    [
      "@docusaurus/theme-classic",
      {
        customCss: require.resolve("./src/css/custom.css"),
      },
    ],
    "@docusaurus/theme-search-algolia",
  ],
};
