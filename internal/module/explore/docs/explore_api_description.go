package docs

// APIDescription is rendered in the OpenAPI info section.
const APIDescription = `NaraLabs Atlas read API for Soroban event exploration.

All list endpoints accept an optional ` + "`network`" + ` query parameter. When omitted, the server default from the ` + "`NETWORK`" + ` environment variable is used.

Interactive docs persist the Bearer token in browser storage so authorization is kept across page reloads.`
