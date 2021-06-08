// A simple whoami application that can be configured only by extending via jsonnet
local kube = import 'https://github.com/bitnami-labs/kube-libsonnet/raw/v1.14.6/kube.libsonnet';

{
    local this = self,
    port:: 8080,
    name:: 'whoami',
    image:: 'containous/whoami',
    pullPolicy:: 'IfNotPresent',

    labels:: { app: this.name },

    deployment: kube.Deployment(this.name + '-deployment') {
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
                            image: this.image,
                            imagePullPolicy: this.pullPolicy,
                            securityContext: { runAsUser: 1000 },
                            args: [
                                '--port=' + this.port,
                                '--name=' + this.name,
                            ],
                            ports_+: { http: { containerPort: this.port } },
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