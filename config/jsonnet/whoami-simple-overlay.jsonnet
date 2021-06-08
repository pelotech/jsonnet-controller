local whoami = import './whoami-simple.jsonnet';

whoami {
    local this = self,

    port:: 8081,
    name:: 'whoami-two',
    labels:: {
        app: this.name,
        release: 'development'
    },

    deployment+: { spec+: { replicas: 2 } }
}