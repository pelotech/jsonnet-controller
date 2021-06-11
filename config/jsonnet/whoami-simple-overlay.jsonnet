local whoami = import './whoami-simple.jsonnet';

whoami {
    name:: 'whoami-two',
    port:: 8081,

    app+: { deployment+: { spec+: { replicas: 2 } } }
}