#!/bin/bash
# Regenerate TLS certificate with proper key usage extensions for HTTPS
# This fixes ERR_SSL_KEY_USAGE_INCOMPATIBLE errors

echo "🔐 Regenerating TLS certificate with proper HTTPS key usage..."

openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt -days 365 -nodes \
  -subj "/C=US/ST=CA/L=San Francisco/O=OpenUSP/OU=Development/CN=localhost" \
  -extensions v3_req -config <(
cat <<EOL
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C=US
ST=CA
L=San Francisco
O=OpenUSP
OU=Development
CN=localhost

[v3_req]
keyUsage = keyEncipherment, dataEncipherment, digitalSignature
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
DNS.3 = openusp.plumepoc.net
DNS.4 = *.plumepoc.net
IP.1 = 127.0.0.1
IP.2 = ::1
EOL
)

echo "✅ Certificate generated with proper key usage extensions:"
openssl x509 -in certs/server.crt -text -noout | grep -A 5 "X509v3 Key Usage"
echo ""
echo "🚀 Restart API Gateway to use the new certificate:"
echo "   pkill -f api-gateway"
echo "   source configs/openusp.env && nohup ./bin/api-gateway > logs/api-gateway.log 2>&1 &"
