// A simple whoami application that can be configured in different namespaces
local whoami = import './whoami-tla.jsonnet';
local kube = import 'https://github.com/bitnami-labs/kube-libsonnet/raw/v1.14.6/kube.libsonnet';

function(
    name,
    namespace='default',
    create_namespace=true,
    port=8080,
    user=1000,
    image='containous/whoami',
    pullPolicy='IfNotPresent',
    expose=false,
    hostname='localhost',
    ingressClass='default') {

    local this = self,

    ns: if create_namespace then kube.Namespace(namespace) else null,

    app: whoami(name, port=port, user=user, image=image, pullPolicy=pullPolicy, expose=expose, hostname=hostname, ingressClass=ingressClass) {
        deployment+: {
            metadata+: { namespace: namespace },
        },
        service+: {
            metadata+: { namespace: namespace },
        }
    } + if expose then {
        ingress+: {
            metadata+: { namespace: namespace },
        }
    } else {}

}