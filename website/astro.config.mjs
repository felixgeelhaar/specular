import { defineConfig } from 'astro/config';

// https://astro.build/config
export default defineConfig({
  site: 'https://felixgeelhaar.github.io',
  base: '/specular',
  output: 'static',
  build: {
    assets: '_assets'
  }
});
