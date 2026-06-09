# ACM Keycloak Declarative Configuration

This directory contains declarative JSON configuration files for setting up Keycloak for ACM (Advanced Cluster Management) multi-realm token exchange.

## Architecture

- **Hub Realm**: Central realm where users authenticate
- **Managed Cluster Realms**: One realm per managed cluster
- **Token Exchange**: V1 token exchange using `subject_issuer` parameter
  - Same-realm: `mcp-sts` → `mcp-server` within hub realm
  - Cross-realm: Hub realm token → Managed cluster realm token

## Directory Structure

```
dev/acm/config/keycloak/
├── realm/
│   ├── hub-realm-create.json          # Hub realm configuration
│   └── managed-realm-create.json       # Template for managed cluster realms
├── clients/
│   ├── mcp-server.json                # OAuth client (confidential)
│   ├── mcp-client.json                # Browser OAuth client (public)
│   └── mcp-sts.json                   # STS client for token exchange
├── client-scopes/
│   ├── openid.json                    # OpenID Connect scope
│   └── mcp-server.json                # MCP audience scope
├── mappers/
│   ├── mcp-server-audience-mapper.json  # Adds mcp-server to aud claim
│   └── sub-claim-mapper.json          # Maps user ID to sub claim
├── users/
│   └── mcp.json                       # Test user (mcp/mcp)
└── identity-providers/
    └── hub-realm-idp-template.json    # IDP config for cross-realm trust
```

## Configuration Files

### Hub Realm (`realm/hub-realm-create.json`)

- Realm name: `hub`
- User registration: disabled
- Password reset: enabled
- Brute force protection: enabled
- Token lifespans configured for security

### Clients

#### `mcp-server` (Confidential Client)
- Used by MCP server for OAuth authentication
- Direct access grants enabled (password flow)
- Service accounts enabled
- Default scopes: `openid`, `profile`, `email`, `mcp-server`

#### `mcp-client` (Public Client)
- Used by browser-based tools (e.g., MCP Inspector)
- PKCE enabled for security
- Authorization code flow only
- No service accounts

#### `mcp-sts` (STS Client)
- Used for token exchange operations
- Service accounts only (no user login)
- No redirect URIs (not for browser flows)

### Client Scopes

#### `openid`
- Standard OpenID Connect scope
- Provides basic user claims (sub, iss, aud, exp, iat)

#### `mcp-server`
- Custom audience scope
- Adds `mcp-server` to the `aud` claim in access tokens
- Required for token validation

### Protocol Mappers

#### `mcp-server-audience`
- Type: `oidc-audience-mapper`
- Adds `mcp-server` to the audience claim
- Applied to `mcp-server` client scope

#### `sub`
- Type: `oidc-sub-mapper`
- Maps user ID to `sub` claim
- Used for federated identity linking

### Users

#### `mcp` User
- Username: `mcp`
- Password: `mcp`
- Email: `mcp@example.com`
- Full name: MCP User
- Used for testing and development

### Identity Provider

#### Hub Realm IDP Template
- Provider: `oidc` (generic OIDC, not keycloak-oidc)
- Trust email: enabled
- Store token: disabled
- Sync mode: IMPORT (create local users)
- Signature validation: enabled via JWKS URL

## Variable Substitution

JSON templates use `${VARIABLE_NAME}` placeholders that are replaced at runtime:

- `${KEYCLOAK_URL}`: Base Keycloak URL (e.g., `https://keycloak-keycloak.apps.example.com`)
- `${HUB_CLIENT_SECRET}`: Secret for mcp-server client in hub realm
- `${MANAGED_REALM}`: Name of managed cluster realm (e.g., `managed-cluster-one`)

## Usage

These JSON files are applied via the Keycloak Admin REST API using the setup scripts:

1. **Hub Setup**: `hack/acm/acm-keycloak-setup-hub-declarative.sh`
   - Creates hub realm
   - Creates clients (mcp-server, mcp-client, mcp-sts)
   - Creates client scopes (openid, mcp-server)
   - Adds protocol mappers
   - Creates test user
   - Configures same-realm token exchange permissions

2. **Managed Cluster Registration**: `hack/acm/acm-register-managed-cluster-declarative.sh`
   - Creates managed cluster realm
   - Registers identity provider (hub realm)
   - Creates federated user link
   - Configures cross-realm token exchange permissions

## Token Exchange Configuration

### Same-Realm Token Exchange (Hub)

Allows `mcp-sts` client to exchange tokens for `mcp-server` audience within the hub realm.

**Steps** (applied by setup script):
1. Enable management permissions on `mcp-server` client
2. Get token-exchange permission ID
3. Create client policy allowing `mcp-sts`
4. Link policy to token-exchange permission

**Test Command**:
```bash
source .keycloak-config/hub-config.env
./hack/acm/test-same-realm-token-exchange.sh
```

### Cross-Realm Token Exchange (Hub → Managed)

Allows exchanging hub realm token for managed cluster realm token.

**Steps** (applied by setup script):
1. Create identity provider in managed realm pointing to hub realm
2. Create federated identity link (hub user → managed user via `sub` claim)
3. Enable fine-grained permissions on IDP
4. Create client policy allowing hub realm's `mcp-sts`
5. Link policy to token-exchange permission on IDP

**Test Command**:
```bash
source .keycloak-config/hub-config.env
source .keycloak-config/clusters/managed-cluster-one.env
./hack/acm/test-cross-realm-token-exchange.sh
```

## Keycloak Admin API Endpoints

Configuration is applied using these endpoints:

- **Realm**: `POST /admin/realms`
- **Clients**: `POST /admin/realms/{realm}/clients`
- **Client Scopes**: `POST /admin/realms/{realm}/client-scopes`
- **Protocol Mappers**: `POST /admin/realms/{realm}/client-scopes/{scope-id}/protocol-mappers/models`
- **Users**: `POST /admin/realms/{realm}/users`
- **Identity Providers**: `POST /admin/realms/{realm}/identity-provider/instances`
- **Client Permissions**: `PUT /admin/realms/{realm}/clients/{client-id}/management/permissions`
- **Authorization Policies**: `POST /admin/realms/{realm}/clients/{client-id}/authz/resource-server/policy/client`

## Benefits of Declarative Approach

1. **Version Control**: Configuration as code
2. **Repeatability**: Same configuration every time
3. **Testability**: Easy to test in different environments
4. **Documentation**: Self-documenting via JSON structure
5. **Validation**: JSON schema validation possible
6. **Idempotency**: Can reapply without side effects
7. **Debugging**: Easy to compare configurations

## References

- Keycloak Admin REST API: https://www.keycloak.org/docs-api/26.0/rest-api/index.html
- Token Exchange: https://www.keycloak.org/docs/latest/securing_apps/#_token-exchange
- Identity Brokering: https://www.keycloak.org/docs/latest/server_admin/#_identity_broker
