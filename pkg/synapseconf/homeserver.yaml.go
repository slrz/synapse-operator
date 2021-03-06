// DO NOT EDIT ** This file was generated with the bake tool ** DO NOT EDIT //

package synapseconf

const homeserverYAMLTemplateText = `# Synapse configuration file template
---

server_name: "{{ .ServerName }}"
pid_file: /data/homeserver.pid
signing_key_path: /data/homeserver.signing.key
log_config: /data/homeserver.log.config
media_store_path: /data/media
uploads_path: /data/uploads

{{ with .WebClientLocation }}
web_client_location: {{ . }}
{{ end }}

public_baseurl: "https://{{ .ServerName }}/"

listeners:
  - port: 8008
    tls: false
    type: http
    x_forwarded: true
    resources:
      - names: [client, federation]
        compress: false

{{ with .AdminContact }}
admin_contact: {{ . }}
{{ end }}

# These are verified by other Matrix servers. Synapse cannot publish the
# correct fingerprints itself when running behind a reverse proxy.  We could
# update the fingerprints as necessary but for now, just punt on it.
#tls_fingerprints: [{"sha256": "<base64_encoded_sha256_fingerprint>"}]

# Taken from default homeserver.yaml
federation_ip_range_blacklist:
  - '127.0.0.0/8'
  - '10.0.0.0/8'
  - '172.16.0.0/12'
  - '192.168.0.0/16'
  - '100.64.0.0/10'
  - '169.254.0.0/16'
  - '::1/128'
  - 'fe80::/64'
  - 'fc00::/7'

{{ with .PostgresConfig }}
database:
  name: "psycopg2"
  args:
    user: "{{ or .User "synapse" }}"
    password: "{{ .Password }}"
    database: "{{ or .DB "synapse" }}"
    host: "{{ .Host }}"
    port: "{{ or .Port "5432" }}"
    cp_min: 5
    cp_max: 10
{{ else }}
database:
  name: "sqlite3"
  args:
    database: "/data/homeserver.db"
{{ end }}

registration_shared_secret: "{{ .RegistrationSharedSecret }}"
macaroon_secret_key: "{{ .MacaroonSecretKey }}"
form_secret: "{{ .FormSecret }}"

enable_metrics: True
{{ if .ReportStats }}
report_stats: True
{{ else }}
report_stats: False
{{ end }}

trusted_key_servers:
  - server_name: "matrix.org"


{{ printf "%s" .IncludeConfigYAML }}
`
