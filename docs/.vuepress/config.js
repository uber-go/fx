const { description } = require('../package')

module.exports = {
  // We're deploying to https://uber-go.github.io/fx/
  // so base should be /fx/.
  base: '/fx/',
  /**
   * Ref：https://v1.vuepress.vuejs.org/config/#title
   */
  title: 'Fx',
  /**
   * Ref：https://v1.vuepress.vuejs.org/config/#description
   */
  description: description,

  dest: 'dist', // Publish built website to dist. We'll feed this to GitHub.

  /**
   * Extra tags to be injected to the page HTML `<head>`
   *
   * ref：https://v1.vuepress.vuejs.org/config/#head
   */
  head: [
    ['meta', { name: 'theme-color', content: '#3eaf7c' }],
    ['meta', { name: 'apple-mobile-web-app-capable', content: 'yes' }],
    ['meta', { name: 'apple-mobile-web-app-status-bar-style', content: 'black' }],
    ['script',
      {
        async: true,
        src: 'https://www.googletagmanager.com/gtag/js?id=G-4YWLTPJ46M'
      }
    ],
    ['script', {},
      [
        "window.dataLayer = window.dataLayer || [];\n"+
        "function gtag(){dataLayer.push(arguments);}\n"+
        "gtag('js', new Date());\n"+
        "gtag('config', 'G-4YWLTPJ46M');"
      ]
    ]
  ],

  /**
   * Theme configuration, here is the default theme configuration for VuePress.
   *
   * ref：https://v1.vuepress.vuejs.org/theme/default-theme-config.html
   */
  themeConfig: {
    repo: 'uber-go/fx',
    editLinks: true,
    docsDir: 'docs',
    lastUpdated: true,
    nav: [
      {
        text: 'Guide',
        link: '/intro',
      },
      {
        text: 'API Reference',
        link: 'https://pkg.go.dev/go.uber.org/fx'
      }
    ],
    sidebar: [
      {
        title: 'Get Started',
        path: '/get-started/',
        children: [
          'get-started/minimal.md',
          'get-started/http-server.md',
          'get-started/echo-handler.md',
          'get-started/logger.md',
          'get-started/registration.md',
          'get-started/another-handler.md',
          'get-started/many-handlers.md',
          'get-started/conclusion.md',
        ],
      },
      'intro.md',
      {
        title: 'Concepts',
        children: [
          'container.md',
          ['lifecycle.md', 'Lifecycle'],
          'modules.md',
        ],
      },
      {
        title: 'Features',
        children: [
          'parameter-objects.md',
          'result-objects.md',
          'annotate.md',
          {
            title: 'Value Groups',
            path: '/value-groups/',
            children: [
              {
                title: 'Feeding',
                path: 'value-groups/feed.md',
              },
              {
                title: 'Consuming',
                path: 'value-groups/consume.md',
              },
            ],
          },
        ],
      },
      ['faq.md', 'FAQ'],
      {
        title: 'Community',
        children: [
          'contributing.md',
        ],
      },
      {
        title: 'Release notes',
        path: 'changelog.md',
      },
    ]
  },

  /**
   * Apply plugins，ref：https://v1.vuepress.vuejs.org/zh/plugin/
   */
  plugins: [
    '@vuepress/plugin-back-to-top',
    '@vuepress/plugin-medium-zoom',
    'fulltext-search',
    'vuepress-plugin-mermaidjs',
    'vuepress-plugin-code-copy',
  ]
}
