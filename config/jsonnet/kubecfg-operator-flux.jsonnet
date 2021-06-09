local kubecfg_operator = import './kubecfg-operator.jsonnet';

kubecfg_operator {
    flux_enabled:: true
}