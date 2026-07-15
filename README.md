# Cassini map

**Create ephemeral maps for sharing your location.**
Share locations between users privately.

This repository provides frontend pages and api for the Cassini App.

## Development

### Requirements

This backend is written in Go and rel the gintonic web framework.
The supported database is exclusively sqlite, the code is intended to be specific to this database.
The automations rely on [Mise en place](https://mise.jdx.dev/).

### Mise commands

- Linting `mise run lint`
- Running tests `mise run test`
- Running the server `mise run server`
- Send a fake position `mise run send-position <id-of-the-map>`

### Frontend and API

The frontend pages are written in Go templates and use vanilla javascript.

The API is documented in this openapi file:
[openapi.yaml](openapi.yaml)

### Project structure

- docs: documentation
- website: 
  - apiv1: Implement handlers for the API
  - database: Contains the implementation of the database access
  - js: Contains the JavaScript code for the frontend
  - templates: Contains the templates for the frontend pages
  
