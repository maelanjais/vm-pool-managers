// src/stores/fetchServerpools.ts
import { writable, get } from 'svelte/store';
import type { Writable } from 'svelte/store';
import { authStore } from './authStore';

// Types
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
  error: string | null;
}

function createServerpoolStore() {
  const { subscribe, set, update }: Writable<ServerpoolStore> = writable({
    user: null,
    serverpools: [],
    error: null
  });

  async function fetchServerpools() {
    const token = get(authStore);
    try {
      // Récupère le profil utilisateur
      const res = await fetch('http://localhost:8080/users/me', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (!res.ok) {
        update(state => ({ ...state, error: "Impossible de récupérer le profil" }));
        return;
      }
      const data = await res.json();
      update(state => ({
        ...state,
        user: { id: data.id, name: data.name, email: data.email },
        error: null
      }));

      // Récupère les serverpools de l'utilisateur
      const res2 = await fetch('http://localhost:8080/serverpool/mysp', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (!res2.ok) {
        update(state => ({ ...state, error: "Impossible de récupérer les serverpools" }));
        return;
      }
      const data2 = await res2.json();
      update(state => ({
        ...state,
        serverpools: data2.serverpools || [],
        error: null
      }));
    } catch (err) {
      update(state => ({ ...state, error: "Erreur backend" }));
      console.error(err);
    }
  }

  async function fetchServersInServerpool(serverpoolId: string): Promise<Server[]> {
    const token = get(authStore);
    try {
      // Correction de l'URL pour la bonne route backend
      const res = await fetch(`http://localhost:8080/serverpool/mysp/${serverpoolId}`, {
        headers: { 'Authorization': `Bearer ${token}` }
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

// Création d'un serverpool
async function createServerpool(serverpool: {
  namesp: string;
  image_ref: string;
  flavor_ref: string;
  networks: string[];
  min_vm: number;
  max_vm: number;
}) {
  const token = get(authStore);
  try {
    
    const res = await fetch(`http://localhost:8080/serverpool`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(serverpool)
    });
    if (!res.ok) {
      throw new Error("Impossible de créer le serverpool");
    } else {
      // Actualiser la liste des serverpools après la création
      await serverpoolStore.fetchServerpools();
    }
  } catch (err) {
    console.error(err);
    throw err;
  }
}

export interface ImageOption {
  value: string;
  name: string;
  status: string;
  Mindisk: number;
  Minram: number;
}

async function fetchAllImages(): Promise<ImageOption[]> {
  const token = get(authStore);
  try {
    const res = await fetch('http://localhost:8080/serverpool/images', {
      headers: { 'Authorization': `Bearer ${token}` } // Ajout du token d'authentification
    });
    if (!res.ok) {
      throw new Error("Impossible de récupérer les images");
    }
    const data = await res.json();
    return (data.images || []).map((img: any) => ({
      value: img.id,
      name: img.name || img.id,
      status: img.status,
      Mindisk: img.min_disk,
      Minram: img.min_ram
    }));
  } catch (err) {
    console.error(err);
    return [];
  }
}

export interface FlavorOption {
  value: string;
  name: string;
  disk: number;
  ram: number;
  vcpus: number;
  rxtx_factor: number;
}

async function fetchAllFlavors(): Promise<FlavorOption[]> {
  const token = get(authStore);
  try {
    const res = await fetch('http://localhost:8080/serverpool/flavor', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) {
      throw new Error("Impossible de récupérer les flavors");
    }
    const data = await res.json();
    return (data.flavors || []).map((flavor: any) => ({
      value: flavor.id,
      name: flavor.name || flavor.id,
      disk: flavor.disk,
      ram: flavor.ram,
      vcpus: flavor.vcpus,
      rxtx_factor: flavor.rxtx_factor
    }));
  } catch (err) {
    console.error(err);
    return [];
  }
}

export interface NetworkOption {
  value: string;
  name: string;
}

async function fetchAllNetworks(): Promise<NetworkOption[]> {
  const token = get(authStore);
  try {
    const res = await fetch('http://localhost:8080/serverpool/networks', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) {
      throw new Error("Impossible de récupérer les réseaux");
    }
    const data = await res.json();
    return (data.networks || []).map((net: any) => ({
      value: net.id,
      name: net.name || net.id,
      disk: net.disk,
      ram: net.ram,
      vcpus: net.vcpus,
      rxtx_factor: net.rxtx_factor
    }));
  } catch (err) {
    console.error(err);
    return [];
  }
}

// Supprimer un serverpool
async function deleteServerpool(serverpoolId: string) {
  const token = get(authStore);
  try {
    const res = await fetch(`http://localhost:8080/serverpool/${serverpoolId}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) {
      throw new Error("Impossible de supprimer le serverpool");
    } else {
      // Actualiser la liste des serverpools après la suppression
      await serverpoolStore.fetchServerpools();
    }
  } catch (err) {
    console.error(err);
    throw err;
  }
}

// Exports

export { createServerpool, fetchAllImages, fetchAllFlavors, fetchAllNetworks, deleteServerpool };
export const serverpoolStore = createServerpoolStore();
