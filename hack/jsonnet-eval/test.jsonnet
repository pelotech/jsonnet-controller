local utils = import 'internal://utils.libsonnet';

{
    chart: utils.helmTemplate('example', './chart', {
        values: { replicaCount: 2 },
    })
}