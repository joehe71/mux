import svelte from 'eslint-plugin-svelte'

export default [
  ...svelte.configs['flat/recommended'],
  {
    ignores: ['dist/**', 'node_modules/**', 'wailsjs/**'],
  },
]
