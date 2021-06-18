local whoami = import './whoami-simple.jsonnet';

whoami {
    value:: 'hello',
    name:: 'whoami-%s' % hello,
    port:: 8081,

    app+: { deployment+: { spec+: { replicas: 2 } } }
}