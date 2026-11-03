# Setup after an installation-only CLI release

This guide applies to packages installed with the installation-only skills-hub
binary. It supersedes retained README instructions that require skills-hub auth,
doctor, smoke, or run-agent. Those commands belong to the deferred readiness
profile; they are not prerequisites for the direct MCP setup below. Package
archives and their READMEs remain unchanged for version and checksum integrity.

## Choose the setup path

- Instruction-only skills: connect the installed skill directory to your adopting
  application using that application's skill-discovery settings.
- Executable tools: install their declared runtime dependencies, configure the
  required bindings in the application's secret/environment settings, then
  register their declared entrypoint. The installer does not acquire credentials
  or configure an application's MCP registration for you.
- Agents and plugins: inspect the generated registration files and dependency
  requirements. Workflows that require the deferred readiness profile remain
  outside this installer release; installing their files does not enable them.

Do not copy credentials into package files, MCP configuration JSON, source control
or shell command history. Supply secrets through your adopting application's
secret store or its process environment. Preserve the package's account, scope,
allowlist and budget restrictions. A successful launch or healthcheck proves
neither provider authentication nor permission to operate on an account.

## GA4 MCP Connector 0.2.0: direct MCP configuration

The retained analytics/ga4-mcp-connector@0.2.0 README contains an "Install and
configure readiness" section with a skills-hub auth configure command. Use this
replacement path for the installation-only release:

1. Make Node.js 22 or newer available to the adopting application. Enable the
   Google Analytics Data API in the intended Google Cloud project and give the
   credential read access to the intended GA4 properties with the
   `https://www.googleapis.com/auth/analytics.readonly` scope.
2. In the application's secret/environment settings, supply GA4_CREDENTIAL as
   the JSON credential envelope described in the retained README's "Credential
   envelope" section: access_token, authorized_user or service_account. Use real
   credential values and current expiry information; example values and dates
   are placeholders. Supply GA4_ALLOWED_PROPERTY_IDS as a comma-separated list of
   the permitted numeric property IDs.
3. Bind the credential directly to the server's GA4_CREDENTIAL environment name.
   If an existing secret is named GA4_CREDENTIAL_SOURCE, forward its value under
   GA4_CREDENTIAL. The README's generation variable and CLI auth-profile command
   are for the deferred readiness workflow and are not needed for this direct
   server registration.
4. Replace `<absolute-installed-ga4-directory>` below with the installed package
   directory reported by the installer. Translate this registration into your
   MCP client's settings and explicitly pass through GA4_CREDENTIAL and
   GA4_ALLOWED_PROPERTY_IDS from its secret/environment settings. Do not put
   credential values into this JSON.

```json
{
  "mcpServers": {
    "ga4": {
      "type": "stdio",
      "command": "node",
      "args": ["--preserve-symlinks-main", "<absolute-installed-ga4-directory>/scripts/ga4_mcp_server.mjs"]
    }
  }
}
```

Using the absolute script path avoids depending on the client's working
directory. The --preserve-symlinks-main flag keeps the retained server's entrypoint
check working when the installed path contains a symlink. See the
[Node.js flag reference](https://nodejs.org/api/cli.html#--preserve-symlinks-main).
The package's .mcp.json instead uses a relative script path and must
be launched with its working directory set to the installed package root.

For a credential-free, non-network launch check, run the same registered command
with --healthcheck appended. From the installed package directory, that is:

```sh
node --preserve-symlinks-main scripts/ga4_mcp_server.mjs --healthcheck
```

The expected JSON identifies status "ok", server "ga4-mcp-connector", version
"0.2.0" and transport "stdio". This verifies process launch only. Starting the
registered server without --healthcheck enables MCP communication; an actual
GA4 report uses the supplied credentials and contacts Google. Live report checks
require your separately approved provider context and remain outside the
installer's verification. Do not substitute a healthcheck for that evidence.

## Other retained tools

For warehouse/bigquery-mcp-query-runner@0.2.0, supply BQ_CREDENTIAL and BQ_POLICY
through the runtime environment and register scripts/bigquery_mcp_server.mjs
under Node.js 22+. The retained README's "Configure" section and TOOL.md describe
credential envelopes and the project, dataset, billing, location and query limits.

For ads/meta-ads-mcp-connector@0.2.0, supply META_ADS_CREDENTIAL and META_ADS_POLICY
and register scripts/meta_ads_mcp_server.mjs under Node.js 22+. Use the retained
README and TOOL.md for the credential envelope and business/account policy.

Both packages can be registered directly without a CLI auth profile. Retain all
of their declared authority restrictions. Their provider evidence/promotion
requirements still apply; the installer does not perform or certify those checks.

For another package, use its declared execution command, required credential
binding names and configuration sections. If it documents only a deferred
readiness workflow, keep it installed but do not enable that workflow with this
release. No model keys or readiness evidence are required merely to install it.
