openssl req  -nodes -newkey rsa:2048 -sha256 -keyout party-iflearner-secret.key -reqexts req_ext -out party-iflearner-secret.csr -subj "/CN=*.party.iflearner.com" -config party-san.cnf

openssl x509 -req -in party-iflearner-secret.csr -extfile party-san.cnf -extensions req_ext -signkey party-iflearner-secret.key -days 3650 -out party-iflearner-secret.crt

kubectl create secret tls party-iflearner-secret --key party-iflearner-secret.key --cert party-iflearner-secret.crt 
