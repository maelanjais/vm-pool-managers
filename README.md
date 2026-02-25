# vm-pool-managers

## Installation

### Sur Openstack

Dans Instances -> Lancer une instance
Source: ubuntu-2404.amd64-genericcloud.20260108 ou plus récent
Gabarit: au moins vd.2
Réseaux: public-2
Groupes de sécurité : ouvrir le port 22, 80 et 5173
Configuration: copier le fichier cloud-init_script

Se connecter en ssh a la machine.
Attendre que la commande ```cloud-init status --long``` soit fini (status: done).

### Configuration de la database

```sh
sudo -i -u postgres
psql
```

Dans psql :

```sh
CREATE DATABASE control_center;
CREATE USER admin WITH PASSWORD 'password_psql';
ALTER ROLE admin SET client_encoding TO 'utf8';
ALTER ROLE admin SET default_transaction_isolation TO 'read committed';
ALTER ROLE admin SET timezone TO 'UTC';
GRANT ALL PRIVILEGES ON DATABASE control_center TO admin;
```

Puis :

```sh
\c control_center;
ALTER SCHEMA public OWNER TO admin;
GRANT ALL ON SCHEMA public TO admin;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO admin;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO admin;
```

### Configuration du projet

