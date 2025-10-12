#!/bin/bash

# Generate test TLS certificates for OpenUSP API Gateway
# This creates self-signed certificates for development/testing only
# Includes proper key usage extensions to fix ERR_SSL_KEY_USAGE_INCOMPATIBLE

CERT_DIR="certs"
DOMAIN="openusp.plumepoc.net"

echo "🔐 Generating test TLS certificates for OpenUSP API Gateway..."
echo "   └── Includes Digital Signature extension for HTTPS compatibility"

# Create certs directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Generate certificate with proper key usage extensions using single OpenSSL command
echo "   └── Generating RSA 4096-bit private key and certificate..."
openssl req -x509 -newkey rsa:4096 -keyout "$CERT_DIR/server.key" -out "$CERT_DIR/server.crt" -days 365 -nodes \
  -subj "/C=IN/ST=TG/L=Hyderabad/O=OpenUSP/OU=Development/CN=$DOMAIN" \
  -extensions v3_req -config <(
cat <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C=IN
ST=TG
L=Hyderabad
O=OpenUSP
OU=Development
CN=$DOMAIN

[v3_req]
keyUsage = keyEncipherment, dataEncipherment, digitalSignature
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = $DOMAIN
DNS.2 = *.plumepoc.net
DNS.3 = localhost
DNS.4 = *.localhost
IP.1 = 44.227.203.156
IP.2 = 127.0.0.1
IP.3 = ::1
EOF
)

# Set appropriate permissions
chmod 600 "$CERT_DIR/server.key"
chmod 644 "$CERT_DIR/server.crt"

echo ""
echo "✅ Test certificates generated successfully with HTTPS-compatible key usage:"
echo "   └── Private Key: $CERT_DIR/server.key (RSA 4096-bit)"
echo "   └── Certificate: $CERT_DIR/server.crt (365 days validity)"
echo ""
echo "🔑 Key Usage Extensions (fixes ERR_SSL_KEY_USAGE_INCOMPATIBLE):"
openssl x509 -in "$CERT_DIR/server.crt" -text -noout | grep -A 3 "X509v3 Key Usage"
echo ""
echo "🌐 Subject Alternative Names:"
openssl x509 -in "$CERT_DIR/server.crt" -text -noout | grep -A 2 "X509v3 Subject Alternative Name"
echo ""
echo "📋 Certificate valid for:"
echo "   └── Domains: $DOMAIN, *.plumepoc.net, localhost, *.localhost"
echo "   └── IP Addresses: 44.227.203.156, 127.0.0.1, ::1"
echo ""
echo "⚠️  WARNING: These are self-signed certificates for testing only!"
echo "   Browsers will show security warnings that you need to accept."
echo ""
echo "🚀 To restart API Gateway with new certificates:"
echo "   pkill -f api-gateway"
echo "   source configs/openusp.env && nohup ./bin/api-gateway > logs/api-gateway.log 2>&1 &"
echo ""
echo "� To verify certificate details:"
echo "   openssl x509 -in $CERT_DIR/server.crt -text -noout"
