#!/bin/bash

# Generate test TLS certificates for OpenUSP API Gateway
# This creates self-signed certificates for development/testing only

CERT_DIR="certs"
DOMAIN="openusp.plumepoc.net"

echo "🔐 Generating test TLS certificates for OpenUSP API Gateway..."

# Create certs directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Generate private key
echo "   └── Generating private key..."
openssl genrsa -out "$CERT_DIR/server.key" 2048

# Generate certificate signing request
echo "   └── Generating certificate signing request..."
openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" -subj "/C=IN/ST=TG/L=Hyderabad/O=OpenUSP/OU=Development/CN=$DOMAIN"

# Create config file for certificate with Subject Alternative Names
cat > "$CERT_DIR/server.conf" << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = US
ST = CA
L = San Francisco
O = OpenUSP
OU = Development
CN = $DOMAIN

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = $DOMAIN
DNS.2 = 44.227.203.156
IP.1 = 44.227.203.156

EOF

# Generate self-signed certificate
echo "   └── Generating self-signed certificate..."
openssl x509 -req -in "$CERT_DIR/server.csr" -signkey "$CERT_DIR/server.key" -out "$CERT_DIR/server.crt" -days 365 -extensions v3_req -extfile "$CERT_DIR/server.conf"

# Clean up temporary files
rm "$CERT_DIR/server.csr" "$CERT_DIR/server.conf"

# Set appropriate permissions
chmod 600 "$CERT_DIR/server.key"
chmod 644 "$CERT_DIR/server.crt"

echo "✅ Test certificates generated successfully:"
echo "   └── Private Key: $CERT_DIR/server.key"
echo "   └── Certificate: $CERT_DIR/server.crt"
echo "   └── Valid for: $DOMAIN, localhost, 127.0.0.1"
echo ""
echo "⚠️  WARNING: These are self-signed certificates for testing only!"
echo "   Browsers will show security warnings that you need to accept."
echo ""
echo "📋 To verify certificate:"
echo "   openssl x509 -in $CERT_DIR/server.crt -text -noout"
