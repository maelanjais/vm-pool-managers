// src/stores/fetchServerpools.ts
import { writable, get } from 'svelte/store';
import type { Writable } from 'svelte/store';
import { authStore } from './authStore'; // ton store auth existant

export interface User {
  id: string;
  name: string;
  email: string;
}

interface ServerpoolStore {
  user: User | null;
  serverpools: any[];
  error: string | null;
}

function createServerpoolStore() {
  const { subscribe, set, update }: Writable<ServerpoolStore> = writable({
    user: null,
    serverpools: [],
    error: null
  });

  async function fetchServerpools() {
    const token = get(authStore); // récupère la valeur actuelle du token

    try {
      const res = await fetch('http://localhost:8080/users/me', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (!res.ok) {
        update(state => ({ ...state, error: "Impossible de récupérer le profil" }));
        return;
      }

      const data = await res.json();
      update(state => ({ ...state, user: { id: data.id, name: data.name, email: data.email }, error: null }));

      const res2 = await fetch('http://localhost:8080/serverpool/mysp', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (!res2.ok) {
        update(state => ({ ...state, error: "Impossible de récupérer les serverpools" }));
        return;
      }

      const data2 = await res2.json();
      update(state => ({ ...state, serverpools: data2.serverpools || [], error: null }));

    } catch (err) {
      update(state => ({ ...state, error: "Erreur backend" }));
      console.error(err);
    }
  }

  async function fetchServersInServerpool(serverpoolId: string) {
  const token = get(authStore);
  try {
    const res = await fetch(`http://localhost:8080/serverpool/mysp/${serverpoolId}`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    if (!res.ok) {
      throw new Error("Impossible de récupérer les serveurs du serverpool");
    }
    const data = await res.json();
    return data.servers || [];
  } catch (err) {
    console.error(err);
    return [];
  }
}

  return {
    subscribe,
    fetchServerpools,
    fetchServersInServerpool,
  };
}

export const serverpoolStore = createServerpoolStore();
