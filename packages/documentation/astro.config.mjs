// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: '4R Docs',
			description:
				'Documentation for 4R — self-hosted AI code review for GitLab merge requests, built around the Risk · Readability · Reliability · Resilience framework.',
			logo: { src: './src/assets/logo.svg', alt: '4R' },
			customCss: ['./src/styles/theme.css'],
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/yebt/4r-ai-mr-reviewer' },
			],
			sidebar: [
				{
					label: 'Getting started',
					items: [
						{ label: 'Introduction', slug: 'getting-started/introduction' },
						{ label: 'Quick start', slug: 'getting-started/quick-start' },
						{ label: 'Your first review', slug: 'getting-started/first-review' },
					],
				},
				{
					label: 'Guides',
					items: [
						{ label: 'The 4R framework', slug: 'guides/the-4r-framework' },
						{ label: 'Running reviews', slug: 'guides/running-reviews' },
						{ label: 'Providers & models', slug: 'guides/providers-and-models' },
						{ label: 'Humanize — your voice', slug: 'guides/humanize' },
						{ label: 'Creating merge requests', slug: 'guides/merge-requests' },
						{ label: 'Release routines', slug: 'guides/release-routines' },
						{ label: 'Telegram notifications', slug: 'guides/notifications' },
					],
				},
				{
					label: 'Self-hosting',
					items: [
						{ label: 'Deploy', slug: 'self-hosting/deploy' },
						{ label: 'Reverse proxy & TLS', slug: 'self-hosting/reverse-proxy' },
						{ label: 'Authentication', slug: 'self-hosting/authentication' },
						{ label: 'Backups', slug: 'self-hosting/backups' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'Configuration', slug: 'reference/configuration' },
						{ label: 'CLI & make targets', slug: 'reference/cli' },
						{ label: 'HTTP API', slug: 'reference/api' },
					],
				},
			],
		}),
	],
});
