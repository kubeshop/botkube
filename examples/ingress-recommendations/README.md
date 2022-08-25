# Ingress recommendations

This example scenario showcases the BotKube recommendations feature around Ingress resources.

## Prerequisites

### Software

Install the following applications:

- [colima](https://github.com/abiosoft/colima)
- [Helm](https://helm.sh/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
- [Mkcert](https://github.com/FiloSottile/mkcert)

### Set up local Kubernetes cluster

Run the command:

```bash
colima start --runtime containerd --kubernetes --cpu 2 --memory 3 --disk 20 --profile botkube
```

### Install Ingress Nginx Controller

Run the command:

```bash
helm upgrade --install ingress-nginx ingress-nginx \
  --repo https://kubernetes.github.io/ingress-nginx \
  --namespace ingress-nginx --create-namespace --wait \
  -f ./examples/ingress-recommendations/ingress-overrides.yaml
```

### Generate and use self-signed certificate

1. Generate the certificate:

  ```bash
  mkcert "*.botkube.local" botkube.local localhost 127.0.0.1 ::1
  ```

1. Create Kubernetes secret:

  ```bash
  CERT_CRT=$(echo -n "$(cat ./_wildcard.botkube.local+4.pem)" | base64)
  CERT_KEY=$(echo -n "$(cat ./_wildcard.botkube.local+4-key.pem)" | base64)
  cat > /tmp/secret.yaml << ENDOFFILE
  apiVersion: v1
  kind: Secret
  metadata:
    name: default-ssl-cert
    namespace: ingress-nginx
  type: kubernetes.io/tls
  data:
    tls.crt: ${CERT_CRT}
    tls.key: ${CERT_KEY}
  ENDOFFILE

  kubectl apply -n ingress-nginx -f /tmp/secret.yaml
  ```

1. Add entry to `/etc/hosts`:

  ```bash
  readonly DOMAIN="botkube.local"
  readonly BOTKUBE_HOSTS=("example")

  LINE_TO_APPEND="127.0.0.1 $(printf "%s.${DOMAIN} " "${BOTKUBE_HOSTS[@]}")"
  HOSTS_FILE="/etc/hosts"

  grep -qF -- "$LINE_TO_APPEND" "${HOSTS_FILE}" || (echo "$LINE_TO_APPEND" | sudo tee -a "${HOSTS_FILE}" > /dev/null)
  ```

1. Navigate to https://example.botkube.local/ - you should see 404 error, but the connection should be secured.

### Deploy BotKube

1. Export required environment variables:

  ```bash
  export SLACK_BOT_TOKEN="{token}"
  export SLACK_CHANNEL="{channel}" # e.g. general
  ```

1. Deploy BotKube:

  ```bash
  helm install botkube --namespace botkube ./helm/botkube -f ./examples/ingress-recommendations/botkube-values.yaml --set communications.default-group.slack.token=$SLACK_BOT_TOKEN --set communications.default-group.slack.channels.default.name=$SLACK_CHANNEL --wait --create-namespace 
  ```

### Deploy example app

Deploy the app with Terminal:

```
kubectl apply -f ./examples/ingress-recommendations/deploy
```

## Scenario

In this scenario, we will expose the example application under the `https://example.botkube.local/meme` endpoint.

  See if the URL works: [https://example.botkube.local/meme](https://example.botkube.local/meme). You should still see 404 error.

1. To expose the app, we need an Ingress resource. Create it with BotKube:

  ```
  @BotKube create ingress meme --class ngnix --rule example.botkube.local/*=meme:80
  ```

1. See the BotKube warning on the Slack channel.
1. Oh, snap! We forgot to create a Service. Let's create it:

  ```
  @BotKube expose deployment meme --name=meme --target-port 9090 --port 8080 --type NodePort
  ```

1. Alright, let's see if the app works now: https://example.botkube.local/meme

  Nope, still 404 error...

1. Let's describe the Ingress:

  ```
  @BotKube describe ingress meme
  ```

1. Oh, there's a typo in the `ingressClassName`! It should be `nginx` instead of `ngnix`! 🤯
1. Let's delete it and create once more:

  ```
  @BotKube delete ingress meme
  @BotKube create ingress meme --class nginx --rule example.botkube.local/*=meme:80
  ```

1. See the BotKube warning on the Slack channel.
1. Confirm with:

  ```
  @BotKube describe service meme
  ```
1. Ah, wrong Service port - in Ingress we referred port `80`, but the service port is `8080`... 🤦 

  Let's delete the service:

  ```
  @BotKube delete service meme
  ```

1. Create the Service again, but this time with proper port:

  ```
  @BotKube expose deployment meme --name=meme --target-port 9090 --port 80 --type NodePort
  ```

1. Navigate to https://example.botkube.local/meme.

It works now! 🥳

## Cleanup

From the terminal remove resources:

  ```bash
  kubectl delete ingress meme
  kubectl delete service meme
  kubectl delete -f ./examples/ingress-recommendations/deploy
  ```

You can also remove whole cluster:

```bash
colima delete botkube
```
