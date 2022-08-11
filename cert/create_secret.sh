openssl req -newkey rsa:2048 -nodes -sha256 -keyout server-iflearner-secret.key -x509 -days 3650 -out server-iflearner-secret.crt -subj "/CN=*.server.iflearner.com"

kubectl create secret tls server-iflearner-secret --key server-iflearner-secret.key --cert server-iflearner-secret.crt 

