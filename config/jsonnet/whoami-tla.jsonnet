// A simple whoami application that can be configured with top-level arguments.
local kube = import 'https://github.com/bitnami-labs/kube-libsonnet/raw/v1.14.6/kube.libsonnet';

function(
    name, 
    port=8080,
    user=1000,
    image='containous/whoami',
    pullPolicy='IfNotPresent') {
    local this = self,

    labels:: { app: 'whoami' },

    deployment: kube.Deployment('whoami-deployment') {
        local deployment = self,
        metadata+: {
            labels: this.labels,
        },
        spec+: {
            replicas: 1,
            template+: {
                metadata+: { labels: this.labels },
                spec+: {
                    securityContext: { runAsNonRoot: true },
                    containers_+: {
                        app: kube.Container('app') {
                            image: image,
                            imagePullPolicy: pullPolicy,
                            securityContext: { runAsUser: user },
                            args: [
                                '--port=' + port,
                                '--name=' + name,
                            ],
                            ports_+: { http: { containerPort: port } },
                        },
                    },
                },
            },
        },
    },

    service: kube.Service('whoami-service') {
        metadata+: { labels: this.labels },
        target_pod: this.deployment.spec.template,
        spec+: { type: 'ClusterIP' },
    },
}