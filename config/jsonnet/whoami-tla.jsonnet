// A simple whoami application that can be configured with top-level arguments.
local kube = import 'https://github.com/bitnami-labs/kube-libsonnet/raw/v1.14.6/kube.libsonnet';

function(
    name, 
    port=8080,
    user=1000,
    image='containous/whoami',
    pullPolicy='IfNotPresent',
    expose=false,
    hostname='localhost',
    ingress_class='') {
    local this = self,

    labels:: { app: name },

    deployment: kube.Deployment(name + '-deployment') {
        local deployment = self,
        metadata+: {
            labels: this.labels,
            annotations: {
                exposed: std.toString(expose),
            },
        },
        spec+: {
            replicas: 1,
            template+: {
                metadata+: { 
                    labels: this.labels,
                },
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

    service: kube.Service(name + '-service') {
        metadata+: { labels: this.labels },
        target_pod: this.deployment.spec.template,
        spec+: { type: 'ClusterIP' },
    },

    ingress: if expose then kube._Object('networking.k8s.io/v1', 'Ingress', name + '-ingress') {
        metadata+: { labels: this.labels },
        spec: {
            rules: [
                {
                    host: hostname,
                    http: {
                        paths: [ 
                            { 
                                path: '/', 
                                pathType: 'Prefix',
                                backend: { 
                                    service: { name: this.service.metadata.name, port:  { name: 'http' } } 
                                } 
                            } 
                        ],
                    },
                },
            ],
        },
    }
}