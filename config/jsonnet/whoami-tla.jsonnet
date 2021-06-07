// A simple whoami application that can be configured
// with external variables.
local kube = import 'https://github.com/bitnami-labs/kube-libsonnet/raw/v1.14.6/kube.libsonnet';

function(name, port=8080, labels={}){
    local this = self,

    labels:: { app: 'whoami' },
    all_labels:: this.labels + labels,

    deployment: kube.Deployment('whoami-deployment') {
        local deployment = self,
        metadata+: {
            labels: this.all_labels,
        },
        spec+: {
            replicas: 1,
            selector: {
                matchLabels: this.labels,
            },
            template+: {
                metadata+: { labels: this.all_labels },
                spec+: {
                    securityContext: { runAsNonRoot: true },
                    containers_+: {
                        app: kube.Container('app') {
                            image: 'containous/whoami',
                            imagePullPolicy: 'IfNotPresent',
                            securityContext: { runAsUser: 1000 },
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
        metadata+: { labels: this.all_labels },
        target_pod: this.deployment.spec.template,
        spec+: { type: 'ClusterIP' },
    },
}