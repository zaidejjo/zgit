// @ts-check
import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';
import starlight from '@astrojs/starlight';
import sitemap from '@astrojs/sitemap';

// https://astro.build/config
export default defineConfig({
  site: 'https://zgit.pages.dev',
  trailingSlash: 'never',

  vite: {
    plugins: [tailwindcss()],
  },

  integrations: [
    starlight({
      title: 'Zgit',
      logo: {
        src: './src/assets/zgit-logo.svg',
        replacesTitle: true,
      },
      social: [
        {
          icon: 'code-branch',
          label: 'GitHub',
          href: 'https://github.com/zaidejjo/zgit',
        },
      ],
      sidebar: [
        { slug: 'docs' },
        {
          label: 'Getting Started',
          items: [
            { slug: 'docs/getting-started' },
            { slug: 'docs/installation' },
          ],
        },
        {
          label: 'Configuration',
          items: [
            { slug: 'docs/configuration/theme-keybindings' },
            { slug: 'docs/configuration/git-profile' },
          ],
        },
        {
          label: 'AI Features',
          items: [
            { slug: 'docs/ai/provider-setup' },
            { slug: 'docs/ai/smart-pr-commit' },
          ],
        },
        {
          label: 'Workflows',
          items: [
            { slug: 'docs/workflows/branch-stash-log' },
          ],
        },
      ],
      customCss: [
        './src/styles/starlight.css',
      ],
    }),
    sitemap({
      filter: (page) => !page.includes('/404'),
    }),
  ],
});
