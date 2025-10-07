import { writable, get } from 'svelte/store';
import type { Writable } from 'svelte/store';
import { authStore } from './authStore';
import { connectWebSocket, disconnectWebSocket } from '$lib/websocket';


export interface User {
  id: string;
  name: string;
  email: string;
}

export interface Serverpool {
  serverpool_id: string;
  image_ref: string;
  flavor_ref: string;
  networks: string[];
  min_vm: number;
  max_vm: number;
  pending_jobs: number;
}

export interface Server {
  id: string;
  name: string;
  status: string;
  flavor: { id: string; name: string | null };
  image: { id: string; name: string | null };
  addresses: Record<string, { addr: string }[]>;
  created: string;
  updated?: string;
  host_id?: string;
  progress?: number;
}

interface ServerpoolStore {
    user: User | null;
    serverpools: Serverpool[];
    servers: Record<string, Server[]>; // Clé : serverpool_id
    error: string | null;
}

async function fetchServersForAllServerpools(token: string, serverpools: Serverpool[]) {
  const servers: Record<string, Server[]> = {};

  for (const sp of serverpools) {
    try {
      const res = await fetch(`http://localhost:8080/serverpool/mysp/${sp.serverpool_id}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (!res.ok) throw new Error(`Impossible de récupérer les serveurs du serverpool ${sp.serverpool_id}`);
      const data = await res.json();
      servers[sp.serverpool_id] = data.servers || [];
    } catch (err) {
      console.error(err);
      servers[sp.serverpool_id] = [];
    }
  }

  return servers;
}

function createServerpoolStore() {
    const { subscribe, set, update } = writable<ServerpoolStore>({
        user: null,
        serverpools: [],
        servers: {},
        error: null
    });

    async function fetchInitData(token?: string) {
        if (!token) {
            const t = get(authStore);
            if (!t) return;       // Si token null ou undefined, on arrête
            token = t;            // Ici token est maintenant bien un string
        }
        
        try {
            const resUser = await fetch('http://localhost:8080/users/me', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (!resUser.ok) throw new Error('Erreur lors de la récupération des informations utilisateur');
            const userData: User = await resUser.json();
            console.log("User data fetched:", userData);

            const resPools = await fetch('http://localhost:8080/serverpool/mysp', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (!resPools.ok) throw new Error('Erreur lors de la récupération des serverpools');
            const poolsData: Serverpool[] = (await resPools.json()).serverpools || [];
            console.log("Serverpools fetched:", poolsData);

            const serversData = await fetchServersForAllServerpools(token, poolsData);
            console.log("Servers for all serverpools fetched:", serversData);

            set({ user: userData, serverpools: poolsData, servers: serversData, error: null });

        } catch (err: any) {
            console.error(err);
            update(state => ({ ...state, error: err.message}));
        }
    }

    function reset() {
    set({
      user: null,
      serverpools: [],
      servers: {},
      error: null
    });
  }

    return {
        subscribe,
        fetchInitData,
        reset
    };
}

export const serverpoolStore = createServerpoolStore();