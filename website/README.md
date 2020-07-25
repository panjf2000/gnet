# Website

This website is built using Docusaurus 2, a modern static website generator.

## Installation

```bash
yarn
```

## Local Development

```bash
yarn start
```

This command starts a local development server and open up a browser window.
Most changes are reflected live without having to restart the server.

## Build

```bash
yarn build
```

This command generates static content into the `build` directory and can be
served using any static contents hosting service.

## Deployment

This website is automatically deployed to `GitHub Pages`. Whenever a new commit lands in `master`, Travis CI will run the suite of tests and if everything passes, this website will be deployed via the yarn deploy script.
