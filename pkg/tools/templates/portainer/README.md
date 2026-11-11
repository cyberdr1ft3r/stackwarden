# Portainer installation disabled

StackWarden does not provide a new Portainer installation bundle. The previous
template published management ports and mounted the host Docker socket, which
would grant the container high authority over the host.

Existing Portainer deployments and their volumes are not changed or removed by
this policy. StackWarden retains status inspection and the existing explicit,
authorized uninstall path for already-staged deployments.
