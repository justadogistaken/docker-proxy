openssl req \
    -newkey rsa:4096 -nodes -sha256 -keyout ca.key \
    -x509 -days 365 -out ca.crt
openssl req \
    -newkey rsa:4096 -nodes -sha256 -keyout miproxy.key \
    -out miproxy.csr
openssl x509 -req -days 365 -in miproxy.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out miproxy.crt