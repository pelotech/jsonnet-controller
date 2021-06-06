// A simple whoami application that can be configured
// with external variables.
local kube = import 'https://github.com/bitnami-labs/kube-libsonnet/raw/v1.14.6/kube.libsonnet';

{
    local this = self,
    local port = std.extVar('port'),
    local name = std.extVar('name'),

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
                            image: 'containous/whoami',
                            imagePullPolicy: 'IfNotPresent',
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