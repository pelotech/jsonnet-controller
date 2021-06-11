// A simple whoami application that can be configured only by extending via jsonnet
local whoami = import './whoami-tla.jsonnet';

{
    local this = self,

    port:: 8080,
    name:: 'whoami',
    image:: 'containous/whoami',
    pullPolicy:: 'IfNotPresent',

    app: whoami(this.name, port=this.port, image=this.image, pullPolicy=this.pullPolicy),
}