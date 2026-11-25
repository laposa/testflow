# Local development settings for `make run` and `make docker-up` (spec §5).
# Copy to .env (git-ignored). The binary itself never reads this file.
# Real secrets come from 1Password; never commit them.

ADDR=:8080
BASE_URL=http://localhost:8080
ENVIRONMENT=development
LOG_LEVEL=debug
# Development only: serve /static/ from the source tree instead of the files
# embedded in the binary, so CSS and JS edits show on reload without a restart.
STATIC_DIR=internal/web/static
DATABASE_URL=postgres://testflow:testflow@localhost:5432/testflow?sslmode=disable
# Optional: credentials kept out of the URL (they override any in DATABASE_URL).
# Handy when the password has characters like @ : / that would need URL-encoding.
#   DATABASE_URL=postgres://db.example.com:5432/testflow?sslmode=require
#   DATABASE_USER=testflow
#   DATABASE_PASSWORD=
# Or, instead of DATABASE_URL, give the host (port defaults to 5432, name to testflow):
#   DATABASE_HOST=localhost
#   DATABASE_PORT=5432
#   DATABASE_NAME=testflow
#   DATABASE_USER=testflow
#   DATABASE_PASSWORD=
SESSION_LIFETIME=120m

# GitHub App "laposa-test-portal-dev" (org settings → GitHub Apps)
# Its "Callback URL" list must contain BASE_URL + /auth/callback
# (here http://localhost:8080/auth/callback) for the login to work.
GITHUB_CLIENT_ID=Iv23liaHFw1JSb9s2tC6 #  https://github.com/organizations/laposa/settings/apps/laposa-test-portal-dev
GITHUB_CLIENT_SECRET="op://laposa-secrets-dev/testflow/GITHUB_CLIENT_SECRET"
GITHUB_APP_INSTALLATION_ID=51847197
# PEM private key as ONE line in double quotes, line breaks written as \n:
#   GITHUB_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----\n"
# Convert a downloaded key with:
#   printf 'GITHUB_PRIVATE_KEY="%s"\n' "$(awk '{printf "%s\\n", $0}' key.pem)"
GITHUB_PRIVATE_KEY="op://laposa-secrets-dev/testflow/GITHUB_PRIVATE_KEY"

TEST_SESSION_ENVIRONMENTS=preprod,beta,prod
SYNC_INTERVAL=30s

MAIL_MAILER=log
# Only when MAIL_MAILER=smtp:
# SMTP_HOST=
# SMTP_PORT=587
# SMTP_USERNAME=
# SMTP_PASSWORD=
# MAIL_FROM=testflow@example.com
# Development only, with SMTP_HOST=localhost: send without STARTTLS, e.g. to
# Mailpit (see README, "Testing email locally with Mailpit").
# SMTP_ALLOW_PLAINTEXT=false
