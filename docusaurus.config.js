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
        { to: "docs/", label: "Documentations", position: "left" },
        { to: "blog/", label: "Blog", position: "left" },
        { to: "highlights/", label: "Highlights", position: "right" },
        { to: "community/", label: "Community", position: "right" },
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
              to: "docs/about/what-is-gnet/",
            },
            {
              label: "The Team",
              to: "community/#team",
            },
            {
              label: "Contact Me",
              to: "contact/",
            },
          ],
        },
        {
          title: "Community",
          items: [
            {
              label: "Chat",
              to: "https://gitter.im/gnet-io/gnet",
            },
            {
              label: "Github",
              to: "https://github.com/panjf2000/gnet",
            },
            {
              label: "Github Organization",
              to: "https://github.com/gnet-io",
            },
            {
              label: "Twitter",
              to: "https://twitter.com/_andy_pan",
            },
            {
              label: "Blog",
              to: "blog/",
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
      apiKey: "a9aa400312a1e718f717e1164d912086",
      indexName: "gnet",
      algoliaOptions: {}, // Optional, if provided by Algolia
    },
  },
  presets: [],
  plugins: [
    [
      "@docusaurus/plugin-content-docs",
      {
        editUrl: "https://github.com/panjf2000/gnet/edit/master/website/",
        sidebarPath: require.resolve("./sidebars.js"),
      },
    ],
    [
      "@docusaurus/plugin-content-blog",
      {
        feedOptions: {
          type: "all",
          copyright: `Copyright © ${new Date().getFullYear()} Andy Pan.`,
        },
      },
    ],
    ["@docusaurus/plugin-content-pages", {}],
    path.resolve(__dirname, "./plugins/highlights"),
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
