# MicroServices

## Fonctionnement du microservice

<!-- markdownlint-disable MD033 -->
<img src="../../control_center/workflow.png" width="300" height="500" alt="Architecture diagram showing PoolManager workflow: User and Students circles connect to Frontend (Svelte) component, which connects to Caddy proxy, then to Control Center containing gRPC Openstack. Control Center connects to Database Global (postgres) and Microservice Openstack (server gin) which contains gRPC server. Microservice connects to Openstack circle and DB local (sqlite). Arrows indicate bidirectional communication between components.">
<!-- markdownlint-enable MD033 -->

Ce module est chargé de prendre les requêtes du control center et de consommer l'API d'Openstack.
Il fonctionne avec un système de worker et de jobs, et un polling regulier sur Openstack pour garder sa Database à jour.
Les options sont modifiables dans ```main.go```

Le fichier proto permettant la communication entre ce module et control center est [poolmanager.proto](../../proto/poolmanager.proto)
