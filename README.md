# Iflearner-Operator
Iflearner-operator is the controller of the kubernetes IflearnerJob crd.

## Description

As you known, horizontal federated learning has two roles, party and server. Iflearner-operator can create different kubernetes objects based on roles, involving ingress, service and pod. The relationship is as follows:

![iflearner-operator](./doc/image/iflearner-operator.png)

Between the parties and the server, we communicate using the grpc protocol and use SSL. All traffic will go to the server ingress, which will route traffic to different services based on the subdomain name. The aggregator behind the service will handle the traffic with business logic.

Between parties, we also use the same means of communication to transfer data. Of course, this is an option and the communication between parties can be ignored if not required.

## Getting Started
You need a Kubernetes cluster to run, and iflearner-operator relies on [ingress-nginx](https://github.com/kubernetes/ingress-nginx) to implement ingress. So you need to install [ingress-nginx](https://github.com/kubernetes/ingress-nginx) before starting.

### Install ingress-nginx
You can follow the [Official Installation Guide](https://kubernetes.github.io/ingress-nginx/deploy/) to install [ingress-nginx](https://github.com/kubernetes/ingress-nginx).

You can also install [ingress-nginx](https://github.com/kubernetes/ingress-nginx) as follows:

```sh
kubectl create -f ingress-nginx/deploy.yaml
```

### Install CRD
You can install CRD as follows:

```sh
bin/kustomize build config/crd | kubectl apply -f -
```

### Install controller
You can install controller as follows:

```sh
cd config/manager && ../../bin/kustomize edit set image controller=ghcr.io/iflytek/iflearner-operator:0.2.0
cd ../.. && bin/kustomize build config/default | kubectl apply -f -
```

### Configure DNS
We use domain names to connect to ingress, so you need to configure your Kubernetes DNS. If you are using the coredns component, you can configure as follows:

Firstly, you need to enter edit mode.

```sh
kubectl -n kube-system edit configmap/coredns
```

Then, you need to add some template configurations. The sever uses the domain name ***server.iflearner.com*** and the party uses the domain name ****.party.iflearner.com***.

```sh
data:
  Corefile: |
    .:53 {
        errors
        health {
           lameduck 5s
        }
        ready
        kubernetes cluster.local in-addr.arpa ip6.arpa {
           pods insecure
           fallthrough in-addr.arpa ip6.arpa
           ttl 30
        }
        prometheus :9153
        forward . /etc/resolv.conf
        cache 30
        loop
        reload
        loadbalance
        template IN A server.iflearner.com {
            match .*\.server\.iflearner\.com
            answer "{{ .Name }} 60 IN A 172.31.164.52"
            fallthrough
        }
        template IN A a.party.iflearner.com {
            match .*\.a\.party\.iflearner\.com
            answer "{{ .Name }} 60 IN A 172.31.164.53"
            fallthrough
        }
        template IN A b.party.iflearner.com {
            match .*\.b\.party\.iflearner\.com
            answer "{{ .Name }} 60 IN A 172.31.164.54"
            fallthrough
        }
    }
```

> Note: The real ip depends on your environment.


Finally, restart the coredns to make the configuration take effect.

```sh
kubectl -n kube-system rollout restart deployment coredns
```

### Generate certificate
After configuring DNS, you need to generate certificates for those domains.

1. generate server certificate

    ```sh
    openssl req -newkey rsa:2048 -nodes -sha256 -keyout server-iflearner-secret.key -x509 -days 3650 -out server-iflearner-secret.crt -subj "/CN=*.server.iflearner.com"
    ```

2. generate party certificate

    You need to edit the ***cert/party-san.cnf*** file and add DNS records according to your environment.

    ```sh
    openssl req  -nodes -newkey rsa:2048 -sha256 -keyout party-iflearner-secret.key -reqexts req_ext -out party-iflearner-secret.csr -subj "/CN=*.party.iflearner.com" -config party-san.cnf

    openssl x509 -req -in party-iflearner-secret.csr -extfile party-san.cnf -extensions req_ext -signkey party-iflearner-secret.key -days 3650 -out party-iflearner-secret.crt
    ```

### Create secret
We expose ingress with TLS, so we need certificate and you can create secret as follows:

```sh
# on server side
kubectl create secret tls server-iflearner-secret --key server-iflearner-secret.key --cert server-iflearner-secret.crt

# on party side
kubectl create secret tls party-iflearner-secret --key party-iflearner-secret.key --cert party-iflearner-secret.crt 
```

### Create configmap (just on party side)
We need a certificate to connect to the ingress, so we mount the configmap into the certificate file, you can create the secret as follows:

```sh
kubectl create configmap server-iflearner-crt --from-file=ingress-nginx/server-iflearner-secret.crt

kubectl create configmap party-iflearner-crt --from-file=ingress-nginx/party-iflearner-secret.crt
```


## Uninstall
You can uninstall everything as follows:

```sh
# delete configmap
kubectl delete configmap party-iflearner-crt
kubectl delete configmap server-iflearner-crt

# delete secret
kubectl delete secret party-iflearner-secret
kubectl delete secret server-iflearner-secret

# delete controller
bin/kustomize build config/default | kubectl delete --ignore-not-found=true -f -

# delete crd
bin/kustomize build config/crd | kubectl delete --ignore-not-found=true -f -
```

## How to use
You can create a server and two parties as follows:

* Server

    ```sh
    apiVersion: git.iflytek.com/v1
    kind: IflearnerJob
    metadata:
    name: iflearnerjob-server
    spec:
    role: server
    host: job1.server.iflearner.com
    template:
        spec:
        containers:
        - image: registry.turing.com:5000/iflearner/iflearner:0.1.4
            name: iflearnerjob-server
            imagePullPolicy: IfNotPresent
            args:
            - python 
            - iflearner/business/homo/aggregate_server.py 
            - -n=2
            - --epochs=10
    ```

* Party A

    ```sh
    apiVersion: git.iflytek.com/v1
    kind: IflearnerJob
    metadata:
    name: iflearnerjob-client1
    spec:
    role: client
    host: job1.a.party.iflearner.com
    template:
        spec:
        restartPolicy: Never
        containers:
        - image: registry.turing.com:5000/iflearner/iflearner:0.1.4
            name: iflearnerjob-client
            imagePullPolicy: IfNotPresent
            workingDir: /iflearner/examples/homo/quickstart_pytorch
            args:
            - python
            - -u
            - quickstart_pytorch.py   
            - --name=client1
            - --epochs=10
            - --server=job1.server.iflearner.com:30031
            - --cert=/etc/server-iflearner-secret.crt
            - --peers=0.0.0.0:50001;job1.b.party.iflearner.com:32322
            - --peer-cert=/etc/party-iflearner-secret.crt
    ```

* Party B

    ```sh
    apiVersion: git.iflytek.com/v1
    kind: IflearnerJob
    metadata:
    name: iflearnerjob-client2
    spec:
    role: client
    host: job1.b.party.iflearner.com
    template:
        spec:
        restartPolicy: Never
        containers:
        - image: registry.turing.com:5000/iflearner/iflearner:0.1.4
            name: iflearnerjob-client
            imagePullPolicy: IfNotPresent
            workingDir: /iflearner/examples/homo/quickstart_pytorch
            args:      
            - python
            - -u
            - quickstart_pytorch.py   
            - --name=client2
            - --epochs=10
            - --server=job1.server.iflearner.com:30031
            - --cert=/etc/server-iflearner-secret.crt
            - --peers=0.0.0.0:50001;job1.a.party.iflearner.com:30031
            - --peer-cert=/etc/party-iflearner-secret.crt
    ```
