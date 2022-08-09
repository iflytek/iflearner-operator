openssl req -newkey rsa:2048 -nodes -sha256 -keyout party-iflearner-secret.key -x509 -days 3650 -out party-iflearner-secret.crt -subj "/CN=*.party.iflearner.com"

kubectl create secret tls party-iflearner-secret --key party-iflearner-secret.key --cert party-iflearner-secret.crt 
